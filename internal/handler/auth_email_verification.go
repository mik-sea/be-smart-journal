package handler

import (
	"net/http"
	"net/url"
	"strings"

	"smart-journal/internal/middleware"
	"smart-journal/internal/service"
)

type EmailVerificationHandler struct {
	service     *service.EmailVerificationService
	redirectURL string
}

func NewEmailVerificationHandler(
	service *service.EmailVerificationService,
	redirectURL string,
) *EmailVerificationHandler {
	return &EmailVerificationHandler{
		service:     service,
		redirectURL: strings.TrimSpace(redirectURL),
	}
}

func (h *EmailVerificationHandler) Register(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	protectedSend := auth(http.HandlerFunc(h.send))
	mux.HandleFunc("/api/auth/email-verification", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.verify(w, r)
		case http.MethodPost:
			protectedSend.ServeHTTP(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
}

func (h *EmailVerificationHandler) send(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.service.Send(r.Context(), userID); err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"sent": true})
}

func (h *EmailVerificationHandler) verify(w http.ResponseWriter, r *http.Request) {
	_, err := h.service.Verify(r.Context(), r.URL.Query().Get("token"))
	if err != nil {
		h.redirectVerificationResult(w, r, "failed", verificationFailureReason(err))
		return
	}

	h.redirectVerificationResult(w, r, "success", "")
}

func (h *EmailVerificationHandler) redirectVerificationResult(
	w http.ResponseWriter,
	r *http.Request,
	status string,
	reason string,
) {
	if h.redirectURL == "" {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": status,
			"reason": reason,
		})
		return
	}

	redirectURL, err := url.Parse(h.redirectURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid email verification redirect url")
		return
	}

	query := redirectURL.Query()
	query.Set("status", status)
	if reason != "" {
		query.Set("reason", reason)
	}
	redirectURL.RawQuery = query.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusSeeOther)
}

func verificationFailureReason(err error) string {
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "expired"):
		return "expired"
	case strings.Contains(message, "already used"):
		return "already_used"
	case strings.Contains(message, "no longer matches"):
		return "email_changed"
	default:
		return "invalid"
	}
}
