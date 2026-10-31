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
        INSERT INTO incidents (severity, description, initiator, source_ip, event_time, raw_payload, status)
    VALUES ($1, $2, $3, $4, $5, $6, 'NEW')
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

func (db *PostgresDB) UpdateIncidentStatus(ctx context.Context, incidentID, newStatus string, telegramID int64) error {
	query := `
		UPDATE incidents
SET status = $1, assigned_to = $3, updated_at = NOW()
WHERE id = $2
	`
	tag, err := db.Pool.Exec(ctx, query, newStatus, incidentID, telegramID)
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
		SELECT status, severity, description, initiator, source_ip, event_time, assigned_to FROM incidents WHERE id = $1
	`
	var incident models.Incident
	var assignedTo *int64
	var status, severity, description, initiator, source_ip string
	var etime *time.Time
	err := db.Pool.QueryRow(ctx, query, id).Scan(&status, &severity, &description, &initiator, &source_ip, &etime, &assignedTo)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения инцидента из бд:%w", err)
	}
	incident.AssignedTo = assignedTo
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

func (db *PostgresDB) GetAllActiveEngineers(ctx context.Context) ([]int64, error) {
	query := `
        SELECT telegram_id FROM engineers 
        WHERE is_active = TRUE
    `
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения инженеров: %w", err)
	}
	defer rows.Close()

	var telegramIDs []int64
	for rows.Next() {
		var telegramID int64
		if err := rows.Scan(&telegramID); err != nil {
			return nil, fmt.Errorf("ошибка чтения telegram_id: %w", err)
		}
		telegramIDs = append(telegramIDs, telegramID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по строкам: %w", err)
	}

	return telegramIDs, nil
}

// AddEngineer добавляет нового инженера в БД
func (db *PostgresDB) AddEngineer(ctx context.Context, telegramID int64, name string) error {
	query := `
        INSERT INTO engineers (telegram_id, name, role, is_active)
        VALUES ($1, $2, 'engineer', TRUE)
        ON CONFLICT (telegram_id) DO NOTHING
    `
	_, err := db.Pool.Exec(ctx, query, telegramID, name)
	if err != nil {
		return fmt.Errorf("ошибка добавления инженера: %w", err)
	}
	return nil
}

// MarkInviteCodeUsed помечает код как использованный
func (db *PostgresDB) MarkInviteCodeUsed(ctx context.Context, code string) error {
	query := `
        UPDATE invite_codes SET used = TRUE WHERE code = $1
    `
	tag, err := db.Pool.Exec(ctx, query, code)
	if err != nil {
		return fmt.Errorf("ошибка обновления кода: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("код не найден: %s", code)
	}
	return nil
}
func (db *PostgresDB) TakeIncident(ctx context.Context, tgID int64, incidentID string) (bool, error) {
	// Диагностика: проверяем текущий статус
	var currentStatus string
	err := db.Pool.QueryRow(ctx, "SELECT status FROM incidents WHERE id = $1", incidentID).Scan(&currentStatus)
	if err != nil {
		log.Printf("🔍 TakeIncident: не удалось получить статус (возможно, инцидент не найден): %v", err)
		return false, nil
	}
	log.Printf("🔍 TakeIncident: текущий статус = '%s', id = '%s', tgID = %d", currentStatus, incidentID, tgID)

	query := `
    UPDATE incidents 
    SET status = 'AT WORK', assigned_to = $1, updated_at = NOW() 
    WHERE id = $2 AND (status = 'NEW' OR (assigned_to = $1 AND status IN ('DECIDED', 'DISMISS')))
`
	tag, err := db.Pool.Exec(ctx, query, tgID, incidentID)
	if err != nil {
		return false, fmt.Errorf("ошибка запроса в бд: %w", err)
	}
	log.Printf("🔍 TakeIncident: RowsAffected = %d", tag.RowsAffected())

	if tag.RowsAffected() == 1 {
		return true, nil
	}
	return false, nil
}
func (db *PostgresDB) GetEngineerName(ctx context.Context, tgID int64) (string, error) {
	var name string
	err := db.Pool.QueryRow(ctx, "SELECT name FROM engineers WHERE telegram_id = $1", tgID).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("ошибка получения имени из бд:%w", err)
	}
	return name, nil

}
func (db *PostgresDB) SaveIncidentMessage(ctx context.Context, incident string, chatID int64, messageID int) error {
	query := `
	INSERT INTO incident_messages (incident_id, chat_id, message_id) 
	VALUES ($1, $2, $3) 
	ON CONFLICT (incident_id, chat_id) DO UPDATE SET message_id = $3
	`
	_, err := db.Pool.Exec(ctx, query, incident, chatID, messageID)
	if err != nil {
		return fmt.Errorf("ошибка добавления информации о чатах куда ушла рассылка: %w", err)
	}
	return nil
}
func (db *PostgresDB) GetIncidentMessages(ctx context.Context, incidentID string) ([]struct {
	ChatID    int64
	MessageID int
}, error) {
	rows, err := db.Pool.Query(ctx, "SELECT chat_id, message_id FROM incident_messages WHERE incident_id = $1", incidentID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса: %w", err)
	}
	defer rows.Close()
	var messages []struct {
		ChatID    int64
		MessageID int
	}
	for rows.Next() {

		var m struct {
			ChatID    int64
			MessageID int
		}
		if err := rows.Scan(&m.ChatID, &m.MessageID); err != nil {
			return nil, fmt.Errorf("ошибка чтения строки: %w", err)
		}

		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации: %w", err)
	}
	return messages, nil
}
