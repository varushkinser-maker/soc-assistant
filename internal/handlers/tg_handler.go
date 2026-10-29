package handlers

import (
	"context"
	"fmt"
	"soc-assistant/internal/models"
	"soc-assistant/internal/storage"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HandleTGButton struct {
	db *storage.PostgresDB
}

func NewHandleTGButton(db *storage.PostgresDB) *HandleTGButton {
	return &HandleTGButton{
		db: db,
	}
}

func (h HandleTGButton) HandlerTG(action, incidentID string) *models.CallbackResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	newStatus := ""
	switch action {
	case "take":
		newStatus = "AT WORK"
		err := h.db.UpdateIncidentStatus(ctx, incidentID, newStatus)
		if err != nil {
			return &models.CallbackResult{Err: fmt.Errorf("ошибка обновления статуса")}
		}
		markup := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Инцидент решен", "resolve:"+incidentID),
			),
		)
		result := models.CallbackResult{
			Text:      "В РАБОТЕ",
			EditMode:  "append",
			NewMarkup: &markup,
			Err:       nil,
		}
		return &result
	case "resolve":
		newStatus = "DECIDED"
		err := h.db.UpdateIncidentStatus(ctx, incidentID, newStatus)
		if err != nil {
			return &models.CallbackResult{Err: fmt.Errorf("ошибка обновления статуса")}
		}
		markup := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("РАССКРЫТЬ ИНЦИДЕНТ", "reopen:"+incidentID),
			),
		)
		result := models.CallbackResult{
			Text:      "ИНЦИДЕНТ ЗАКРЫТ",
			EditMode:  "replace",
			NewMarkup: &markup,
			Err:       nil,
		}
		return &result
	case "dismiss":
		newStatus = "DISMISS"
		err := h.db.UpdateIncidentStatus(ctx, incidentID, newStatus)
		if err != nil {
			return &models.CallbackResult{Err: fmt.Errorf("ошибка обновления статуса")}
		}
		markup := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("РАССКРЫТЬ ИНЦИДЕНТ", "reopen:"+incidentID),
			),
		)
		result := models.CallbackResult{
			Text:      "ИНЦИДЕНТ ЗАКРЫТ",
			EditMode:  "replace",
			NewMarkup: &markup,
			Err:       nil,
		}
		return &result
	case "reopen":
		incident, err := h.db.GetIncident(ctx, incidentID)
		if err != nil {
			return &models.CallbackResult{Err: fmt.Errorf("ошибка возврата инцидента")}
		}
		eventTime := "-"
		if incident.EventTime != nil {
			eventTime = incident.EventTime.Format("02.01.2006 15:04")
		}
		status := ""
		if incident.Status == "DISMISS" {
			status = "ЛОЖНЫЙ ИНЦИДЕНТ"
		} else {
			status = "ИНЦИДЕНТ РЕШЕН"
		}
		message := fmt.Sprintf(
			"🚨 *Инцидент:*\n\n"+
				" *Событие:* %s\n"+
				"🔴 *Критичность:* %s\n"+
				" *Инициатор:* %s\n"+
				" *IP:* %s\n"+
				" *Время события:* %s\n"+
				"🆔 *ID:* `%s`\n\n"+
				" *Статус:* %s",
			incident.Description, incident.Severity, incident.Initiator, incident.SourceIP, eventTime, incidentID, status,
		)
		markup := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("В работу", "take:"+incidentID),
				tgbotapi.NewInlineKeyboardButtonData("Закрыть", "close:"+incidentID),
			),
		)
		result := models.CallbackResult{
			Text:      message,
			EditMode:  "replace",
			NewMarkup: &markup,
			Err:       nil,
		}
		return &result
	case "close":
		markup := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("РАССКРЫТЬ ИНЦИДЕНТ", "reopen:"+incidentID),
			),
		)
		result := models.CallbackResult{
			Text:      "ИНЦИДЕНТ ЗАКРЫТ",
			EditMode:  "replace",
			NewMarkup: &markup,
			Err:       nil,
		}
		return &result
	default:
		return &models.CallbackResult{Err: fmt.Errorf("неизветсное дейсвие")}
	}
}
