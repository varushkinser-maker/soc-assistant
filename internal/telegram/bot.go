package telegram

import (
	"context"
	"fmt"
	"log"
	"os"
	"soc-assistant/internal/models"
	"soc-assistant/internal/storage"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramBot struct {
	api    *tgbotapi.BotAPI
	chatID int64
	db     *storage.PostgresDB
}

func NewTelegramBot(db *storage.PostgresDB) (*TelegramBot, error) {
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
		db:     db,
	}, nil
}
func (bot *TelegramBot) SendIncidentNotification(eventType string, incident *models.Incident, incidentID string, engineerTelegramIDs []int64) error {
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
	for _, telegramID := range engineerTelegramIDs {
		msg := tgbotapi.NewMessage(telegramID, message)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		sentMsg, err := bot.api.Send(msg)
		if err != nil {
			log.Printf("⚠️ Не удалось отправить сообщение инженеру %d: %v", telegramID, err)
			// Не прерываем цикл, продолжаем отправку остальным
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := bot.db.SaveIncidentMessage(ctx, incidentID, telegramID, sentMsg.MessageID)
			if err != nil {
				log.Println("ошибка сохранения")
			}
			log.Printf("✅ Сообщение отправлено инженеру %d", telegramID)
		}
	}
	return nil
}

func (bot *TelegramBot) AcceptUpdate(handle func(int64, string, string) *models.CallbackResult, messagehandle func(int64, string, string) *models.MessageResult) {
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
			result := handle(update.CallbackQuery.From.ID, action, incidentID)

			if result.Err != nil {
				log.Printf("❌ Ошибка в handler: %v", result.Err)
				bot.api.Send(tgbotapi.NewCallback(update.CallbackQuery.ID, "Ошибка: "+result.Err.Error()))
				continue
			}
			ctx := context.Background()

			messages, err := bot.db.GetIncidentMessages(ctx, incidentID)
			if err != nil {
				log.Printf("⚠️ Не удалось получить список сообщений: %v", err)

			}
			telegramID := update.CallbackQuery.From.ID // Кто нажал кнопку

			for _, m := range messages {

				var textToUse string
				var markupToUse *tgbotapi.InlineKeyboardMarkup
				var editModeToUse string

				if m.ChatID == telegramID {

					textToUse = result.Text
					markupToUse = result.NewMarkup
					editModeToUse = result.EditMode
				} else {
					if result.BroadcastText == "" {
						continue
					}
					textToUse = result.BroadcastText
					markupToUse = result.BroadcastMarkup
					editModeToUse = "replace"
				}

				var finalText string
				if editModeToUse == "append" {
					originalText := update.CallbackQuery.Message.Text
					finalText = originalText + "\n\n🔄 " + textToUse
				} else {
					finalText = textToUse
				}

				var editConfig tgbotapi.Chattable
				if markupToUse != nil {
					editConfig = tgbotapi.NewEditMessageTextAndMarkup(m.ChatID, m.MessageID, finalText, *markupToUse)
				} else {
					editConfig = tgbotapi.NewEditMessageText(m.ChatID, m.MessageID, finalText)
				}

				// Устанавливаем ParseMode
				if config, ok := editConfig.(tgbotapi.EditMessageTextConfig); ok {
					config.ParseMode = "Markdown"
					editConfig = config
				}

				// Отправляем
				_, err := bot.api.Send(editConfig)
				if err != nil {
					log.Printf("⚠️ Не удалось обновить сообщение в чате %d: %v", m.ChatID, err)
				} else {
					log.Printf("✅ Сообщение обновлено в чате %d", m.ChatID)
				}
			}
			bot.api.Send(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))

		}
		if update.Message != nil && update.Message.Text != "" {
			username := update.Message.From.UserName
			result := messagehandle(update.Message.From.ID, username, update.Message.Text)
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
