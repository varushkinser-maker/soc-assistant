package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	detection "soc-assistant/internal/detect"
	"soc-assistant/internal/models"
	"soc-assistant/internal/storage"
	"soc-assistant/internal/telegram"
)

type AlertHandler struct {
	db  *storage.PostgresDB
	bot *telegram.TelegramBot
}

func NewAlertHandler(db *storage.PostgresDB, bot *telegram.TelegramBot) *AlertHandler {
	return &AlertHandler{
		db:  db,
		bot: bot,
	}
}

func (h *AlertHandler) HandelAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	alert, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("ошибка чтения тела запроса:", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"status": "rejected", "reason": "проблема с тело запроса"})
		return
	}
	extevent := models.Extevent{}
	if err := json.Unmarshal(alert, &extevent); err != nil {
		log.Println("Ошибка декодирования тела запроса:", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"status": "rejected", "reason": "проблема декодирования тела запроса"})
		return
	}
	extevent.Raw = string(alert)
	rule := detection.Match(&extevent)
	response := make(map[string]string, 2)
	if rule == nil {
		response["status"] = "ignored"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	incedent := models.Incident{
		Severity:    rule.Severity,
		Description: rule.RuleName,
		Initiator:   extevent.Authenication.SubjectName,
		SourceIP:    extevent.RequestMetadata.RemoteAddress,
		Raw:         extevent.Raw,
	}
	etime, err := time.Parse(time.RFC3339, extevent.EventTime)
	if err != nil {
		incedent.EventTime = nil
	} else {
		incedent.EventTime = &etime
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	id, err := h.db.CreateIncident(ctx, &incedent)
	if err != nil {
		log.Println("ошибка добавления инцидента в БД:", err)
		w.WriteHeader(500)
		return
	}
	log.Println("инцидент добалвен в БД, id: ", id)
	err = h.bot.SendIncidentNotification(extevent.TypeName(), &incedent, id)
	if err != nil {
		log.Println("ошибка отправки сообщения об инциденте в тг:", err)
	}
	response["status"] = "alerted"
	response["incident_id"] = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

}
