package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"soc-assistant/internal/config"
	"soc-assistant/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDB struct {
	Pool *pgxpool.Pool
}

func NewPostgresDB(cfg *config.Config) (*PostgresDB, error) {
	dns := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
	pool, err := pgxpool.New(context.Background(), dns)
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании пула:%w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ошибка при проверке пинга:%w", err)
	}
	log.Println("подключились к бд")
	return &PostgresDB{Pool: pool}, nil
}

func (db *PostgresDB) CreateIncident(ctx context.Context, alert *models.Incident) (string, error) {
	query := `
        INSERT INTO incidents (severity, description, initiator, source_ip, event_time, raw_payload)
        VALUES ($1, $2, $3, $4,$5,$6)
        RETURNING id
    `
	var incidentID string
	err := db.Pool.QueryRow(ctx, query, alert.Severity,
		alert.Description,
		alert.Initiator,
		alert.SourceIP,
		alert.EventTime,
		alert.Raw).Scan(&incidentID)
	if err != nil {
		return "", fmt.Errorf("не получилось добавить инцедент в бд: %w", err)
	}
	log.Println("инцедент успешно добавлен")
	return incidentID, nil
}

func (db *PostgresDB) Close() {
	db.Pool.Close()
	log.Println("соединени с бд закрыто")
}

func (db *PostgresDB) UpdateIncidentStatus(ctx context.Context, incidentID, newStatus string) error {
	query := `
		UPDATE incidents
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`
	tag, err := db.Pool.Exec(ctx, query, newStatus, incidentID)
	if err != nil {
		return fmt.Errorf("не получилось поменять статус:%w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("нет такого инцедента -%s", incidentID)
	}
	return nil

}
func (db *PostgresDB) GetIncident(ctx context.Context, id string) (*models.Incident, error) {
	query := `
		SELECT status, severity, description, initiator, source_ip, event_time FROM incidents WHERE id = $1
	`
	var incident models.Incident
	var status, severity, description, initiator, source_ip string
	var etime *time.Time
	err := db.Pool.QueryRow(ctx, query, id).Scan(&status, &severity, &description, &initiator, &source_ip, &etime)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения инцидента из бд:%w", err)
	}
	incident.Severity = severity
	incident.Description = description
	incident.Initiator = initiator
	incident.SourceIP = source_ip
	incident.EventTime = etime
	incident.Status = status
	log.Println("успешно получили инцедент и бд")
	return &incident, nil
}

func (db *PostgresDB) GenerateInviteCode(ctx context.Context, idAdmin int64) (string, error) {
	code, err := generateInviteCode()
	if err != nil {
		return "", fmt.Errorf("не получилось сгенерировать код:%w", err)
	}
	maxTime := time.Now().Add(24 * time.Hour)
	query := `
		INSERT INTO invite_codes (code, created_by, expires_at, used) VALUES ($1, $2, $3, FALSE)
	`
	_, err = db.Pool.Exec(ctx, query, code, idAdmin, maxTime)
	if err != nil {
		return "", fmt.Errorf("ошибка запроса в бд:%w", err)
	}
	return code, nil

}
func (db *PostgresDB) ValidateInviteCode(ctx context.Context, code string) error {
	query := `
		SELECT 1 FROM invite_codes WHERE code = $1 AND used = FALSE AND expires_at > NOW()
	`
	var dymmy int
	err := db.Pool.QueryRow(ctx, query, code).Scan(&dymmy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("код невалиден или истек")
		}
		return fmt.Errorf("ошибка с проверкой кода:%w", err)
	}
	return nil
}
func (db *PostgresDB) IsAdmin(ctx context.Context, telegramID int64) (bool, error) {
	query := `
        SELECT 1 FROM engineers 
        WHERE telegram_id = $1 AND role = 'admin' AND is_active = TRUE
    `
	var dummy int
	err := db.Pool.QueryRow(ctx, query, telegramID).Scan(&dummy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("ошибка проверки прав: %w", err)
	}
	return true, nil
}
