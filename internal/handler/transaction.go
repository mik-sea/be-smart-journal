package handler

import (
	"encoding/json"
	"net/http"

	"smart-journal/internal/middleware"
	"smart-journal/internal/service"
)

type TransactionHandler struct {
	service *service.TransactionService
}

func NewTransactionHandler(service *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{service: service}
}

func (h *TransactionHandler) Register(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("/api/transactions", auth(http.HandlerFunc(h.collection)))
	mux.Handle("/api/transactions/", auth(http.HandlerFunc(h.item)))
}

func (h *TransactionHandler) collection(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	switch r.Method {
	case http.MethodGet:
		transactions, err := h.service.List(r.Context(), userID, limitFromRequest(r))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"transactions": transactions})
	case http.MethodPost:
		var request transactionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}

		transaction, err := h.service.Create(r.Context(), service.CreateTransactionInput{
			UserID:   userID,
			NoteID:   request.NoteID,
			Type:     request.Type,
			Amount:   request.Amount,
			Category: request.Category,
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, transaction)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *TransactionHandler) item(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	transactionID, ok := resourceID(r.URL.Path, "/api/transactions/")
	if !ok {
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		transaction, err := h.service.Get(r.Context(), userID, transactionID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, transaction)
	case http.MethodPatch, http.MethodPut:
		var request updateTransactionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}

		transaction, err := h.service.Update(r.Context(), service.UpdateTransactionInput{
			UserID:        userID,
			TransactionID: transactionID,
			NoteID:        request.NoteID,
			Type:          request.Type,
			Amount:        request.Amount,
			Category:      request.Category,
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, transaction)
	case http.MethodDelete:
		if err := h.service.Delete(r.Context(), userID, transactionID); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

type transactionRequest struct {
	NoteID   string  `json:"note_id"`
	Type     string  `json:"type"`
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
}

type updateTransactionRequest struct {
	NoteID   *string  `json:"note_id"`
	Type     *string  `json:"type"`
	Amount   *float64 `json:"amount"`
	Category *string  `json:"category"`
}
