package telegram

import (
	"fmt"
	"log"
	"os"
	"soc-assistant/internal/models"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramBot struct {
	api    *tgbotapi.BotAPI
	chatID int64
}

func NewTelegramBot() (*TelegramBot, error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("ошибка чтения токена бота")
	}
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("ошибка плдключения к тг боту^%w", err)
	}
	log.Println("подключились к боту")
	chatIDs := os.Getenv("SOC_ENGINEER_CHAT_ID")
	if chatIDs == "" {
		return nil, fmt.Errorf("ошибка чтения id инженера")
	}
	var chatID int64
	_, err = fmt.Sscanf(chatIDs, "%d", &chatID)
	if err != nil {
		return nil, fmt.Errorf("ошибка преобразования id в число:%w", err)
	}
	return &TelegramBot{
		api:    api,
		chatID: chatID,
	}, nil
}
func (bot *TelegramBot) SendIncidentNotification(eventType string, incident *models.Incident, incidentID string) error {
	eventTime := "-"
	if incident.EventTime != nil {
		eventTime = incident.EventTime.Format("02.01.2006 15:04")
	}
	message := fmt.Sprintf(
		"🚨 *Новый инцидент!*\n\n"+
			" *Событие:* %s\n"+
			"🔴 *Критичность:* %s\n"+
			" *Инициатор:* %s\n"+
			" *IP:* %s\n"+
			" *Время события:* %s\n"+
			"🆔 *ID:* `%s`",
		eventType, incident.Severity, incident.Initiator, incident.SourceIP, eventTime, incidentID,
	)
	msg := tgbotapi.NewMessage(bot.chatID, message)
	msg.ParseMode = "Markdown"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👨‍ В работу", "take:"+incidentID),
			tgbotapi.NewInlineKeyboardButtonData("✅ Решено", "resolve:"+incidentID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Ложная тревога", "dismiss:"+incidentID),
		),
	)
	msg.ReplyMarkup = keyboard
	_, err := bot.api.Send(msg)
	if err != nil {

		return fmt.Errorf("сообщение в тг не отправилось:%w", err)
	}
	log.Println("сообщение успешно отправилось в тг", incident.Description, incidentID)
	return nil
}

func (bot *TelegramBot) AcceptUpdate(handle func(string, string) *models.CallbackResult, messagehandle func(int64, string) *models.MessageResult) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.api.GetUpdatesChan(u)
	log.Println("🔄 Цикл обработки обновлений Telegram запущен...")

	for update := range updates {
		log.Println("пришло событие", update)
		if update.CallbackQuery != nil {

			callbackData := update.CallbackQuery.Data
			parts := strings.Split(callbackData, ":")

			if len(parts) != 2 {
				log.Printf("⚠️ Кривой формат данных: %s", callbackData)
				bot.api.Send(tgbotapi.NewCallback(update.CallbackQuery.ID, "Ошибка формата данных"))
				continue
			}

			action := parts[0]
			incidentID := parts[1]
			log.Printf("🎯 Нажатие: action=%s, id=%s", action, incidentID)
			result := handle(action, incidentID)

			if result.Err != nil {
				log.Printf("❌ Ошибка в handler: %v", result.Err)
				bot.api.Send(tgbotapi.NewCallback(update.CallbackQuery.ID, "Ошибка: "+result.Err.Error()))
				continue
			}
			var finalText string
			if result.EditMode == "append" {
				originalText := update.CallbackQuery.Message.Text
				finalText = originalText + "\n\n🔄 " + result.Text
			} else {
				finalText = result.Text
			}

			msgID := update.CallbackQuery.Message.MessageID

			var editConfig tgbotapi.Chattable
			if result.NewMarkup != nil {
				editConfig = tgbotapi.NewEditMessageTextAndMarkup(bot.chatID, msgID, finalText, *result.NewMarkup)
			} else {
				editConfig = tgbotapi.NewEditMessageText(bot.chatID, msgID, finalText)
			}

			if config, ok := editConfig.(tgbotapi.EditMessageTextConfig); ok {
				config.ParseMode = "Markdown"
				editConfig = config
			}

			_, err := bot.api.Send(editConfig)
			if err != nil {
				log.Printf("Ошибка редактирования сообщения: %v", err)
			} else {
				log.Println("✅ Сообщение в Telegram успешно обновлено!")
			}

			bot.api.Send(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
		}
		if update.Message != nil && update.Message.Text != "" {
			result := messagehandle(update.Message.From.ID, update.Message.Text)
			if result.Err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, string(result.Err.Error()))
				bot.api.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, result.Text)
				msg.ParseMode = "Markdown"
				_, err := bot.api.Send(msg)
				log.Println("ошибка при отправке сообщения: ", err)
			}
		}
	}

}
