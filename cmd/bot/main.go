package main

import (
	"log"
	"net/http"
	"time"

	"soc-assistant/internal/config"
	"soc-assistant/internal/handlers"
	"soc-assistant/internal/storage"
	"soc-assistant/internal/telegram"
)

func main() {
	log.Println("#пошло поехало. . .")
	cfg := config.Load()
	db, err := storage.NewPostgresDB(cfg)
	if err != nil {
		log.Fatal("ошибка покдлючения к бд", err)
	}
	defer db.Close()

	bot, err := telegram.NewTelegramBot(db)
	if err != nil {
		log.Fatal("не получилос ьинициализировать тг бота", err)
	}

	alerthandler := handlers.NewAlertHandler(db, bot)
	buttonhandle := handlers.NewHandleTGButton(db)
	messageHandler := handlers.NewHandlerTGMessage(db)

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/alert", alerthandler.HandelAlert)
	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Println("HTTP-сервер запущен на порту 8080")
	log.Println("ଘ(੭*ˊᵕˋ)੭* ̀ˋ ɪɴᴛᴇʀɴᴇᴛ Эндпоинт готов: POST http://localhost:8080/webhook/alert")
	go bot.AcceptUpdate(buttonhandle.HandlerTG, messageHandler.HandleMessage)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("oшибка сервера: %v", err)
	}
	log.Println(" Программа завершена, проверь БД")
}
