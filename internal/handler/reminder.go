package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"smart-journal/internal/middleware"
	"smart-journal/internal/service"
)

type ReminderHandler struct {
	service *service.ReminderService
}

func NewReminderHandler(service *service.ReminderService) *ReminderHandler {
	return &ReminderHandler{service: service}
}

func (h *ReminderHandler) Register(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("/api/reminders", auth(http.HandlerFunc(h.collection)))
	mux.Handle("/api/reminders/send-due", auth(http.HandlerFunc(h.sendDue)))
	mux.Handle("/api/reminders/", auth(http.HandlerFunc(h.item)))
}

func (h *ReminderHandler) collection(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	switch r.Method {
	case http.MethodGet:
		reminders, err := h.service.List(r.Context(), userID, limitFromRequest(r))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"reminders": reminders})
	case http.MethodPost:
		var request reminderRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}

		reminder, err := h.service.Create(r.Context(), service.CreateReminderInput{
			UserID:   userID,
			NoteID:   request.NoteID,
			RemindAt: request.RemindAt,
			Message:  request.Message,
			Status:   request.Status,
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, reminder)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ReminderHandler) item(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	reminderID, ok := resourceID(r.URL.Path, "/api/reminders/")
	if !ok {
		writeError(w, http.StatusNotFound, "reminder not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		reminder, err := h.service.Get(r.Context(), userID, reminderID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, reminder)
	case http.MethodPatch, http.MethodPut:
		var request updateReminderRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}

		reminder, err := h.service.Update(r.Context(), service.UpdateReminderInput{
			UserID:     userID,
			ReminderID: reminderID,
			NoteID:     request.NoteID,
			RemindAt:   request.RemindAt,
			Message:    request.Message,
			Status:     request.Status,
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, reminder)
	case http.MethodDelete:
		if err := h.service.Delete(r.Context(), userID, reminderID); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ReminderHandler) sendDue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	count, err := h.service.SendDueForUser(r.Context(), userID, time.Now().UTC(), limitFromRequest(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"sent": count})
}

type reminderRequest struct {
	NoteID   string    `json:"note_id"`
	RemindAt time.Time `json:"remind_at"`
	Message  string    `json:"message"`
	Status   string    `json:"status"`
}

type updateReminderRequest struct {
	NoteID   *string    `json:"note_id"`
	RemindAt *time.Time `json:"remind_at"`
	Message  *string    `json:"message"`
	Status   *string    `json:"status"`
}
