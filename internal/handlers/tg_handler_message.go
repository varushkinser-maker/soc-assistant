package handlers

import (
	"context"
	"fmt"
	"soc-assistant/internal/models"
	"soc-assistant/internal/storage"
	"strings"
	"time"
)

type HandlerTGMessage struct {
	db      *storage.PostgresDB
	adminID int64
}

func NewHandlerTGMessage(db *storage.PostgresDB) *HandlerTGMessage {
	return &HandlerTGMessage{
		db: db,
	}
}

func (h *HandlerTGMessage) HandleMessage(adminID int64, text string) *models.MessageResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return &models.MessageResult{Text: "Пустое сообщение!"}
	}
	comand := parts[0]
	switch comand {
	case "/generate_invite":
		b, err := h.db.IsAdmin(ctx, adminID)
		if err != nil {
			return &models.MessageResult{Err: fmt.Errorf("ошибка проверки на админа:%w", err)}
		}
		if b {
			code, err := h.db.GenerateInviteCode(ctx, adminID)
			if err != nil {
				return &models.MessageResult{Err: fmt.Errorf("ошибка генерации кода:%w", err)}
			}
			return &models.MessageResult{Text: "Код: " + code + " (срок действия кода 24 часа)"}
		} else {
			return &models.MessageResult{Text: "Для этой команды нужна роль админа!"}
		}
	case "/join":
		if len(parts) != 2 {
			return &models.MessageResult{Text: "Использование команды: /join <ваш код активации>"}
		}
		code := parts[1]
		err := h.db.ValidateInviteCode(ctx, code)
		if err != nil {
			return &models.MessageResult{Err: err}
		}
		return &models.MessageResult{Text: "Код приянт, добро пожаловать!"}
	default:
		return &models.MessageResult{Text: "неизвестная команда!\n`Команды:`\n`/join <ваш код активации>`\n`/generate_invite`"}
	}
}
