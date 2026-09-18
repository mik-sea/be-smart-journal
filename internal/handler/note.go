package handler

import (
	"encoding/json"
	"net/http"

	"smart-journal/internal/middleware"
	"smart-journal/internal/service"
)

type NoteHandler struct {
	service *service.NoteService
}

func NewNoteHandler(service *service.NoteService) *NoteHandler {
	return &NoteHandler{service: service}
}

func (h *NoteHandler) Register(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("/api/notes", auth(http.HandlerFunc(h.collection)))
	mux.Handle("/api/notes/", auth(http.HandlerFunc(h.item)))
}

func (h *NoteHandler) collection(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	switch r.Method {
	case http.MethodGet:
		notes, err := h.service.List(r.Context(), userID, limitFromRequest(r))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"notes": notes})
	case http.MethodPost:
		var request noteRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}

		note, err := h.service.Create(r.Context(), service.CreateNoteInput{
			UserID:  userID,
			RawText: request.RawText,
			Title:   request.Title,
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, note)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *NoteHandler) item(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	noteID, ok := resourceID(r.URL.Path, "/api/notes/")
	if !ok {
		writeError(w, http.StatusNotFound, "note not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		note, err := h.service.Get(r.Context(), userID, noteID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, note)
	case http.MethodPatch, http.MethodPut:
		var request updateNoteRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}

		note, err := h.service.Update(r.Context(), service.UpdateNoteInput{
			UserID:  userID,
			NoteID:  noteID,
			RawText: request.RawText,
			Title:   request.Title,
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, note)
	case http.MethodDelete:
		if err := h.service.Delete(r.Context(), userID, noteID); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

type noteRequest struct {
	RawText string `json:"raw_text"`
	Title   string `json:"title"`
}

type updateNoteRequest struct {
	RawText *string `json:"raw_text"`
	Title   *string `json:"title"`
}
