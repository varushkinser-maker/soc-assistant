package models

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type CallbackResult struct {
	Text      string
	NewMarkup *tgbotapi.InlineKeyboardMarkup
	EditMode  string

	BroadcastText   string
	BroadcastMarkup *tgbotapi.InlineKeyboardMarkup

	Err error
}
type MessageResult struct {
	Text string
	Err  error
}
