package handler

import (
	"encoding/json"
	"net/http"

	"smart-journal/internal/middleware"
	"smart-journal/internal/service"
)

type JournalHandler struct {
	service *service.JournalService
}

func NewJournalHandler(service *service.JournalService) *JournalHandler {
	return &JournalHandler{service: service}
}

func (h *JournalHandler) Register(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("/api/journals", auth(http.HandlerFunc(h.create)))
}

func (h *JournalHandler) create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var request createJournalRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	result, err := h.service.Create(r.Context(), service.CreateJournalInput{
		UserID:  userID,
		RawText: request.RawText,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

type createJournalRequest struct {
	RawText string `json:"raw_text"`
}
