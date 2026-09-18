package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	firebaseauth "firebase.google.com/go/v4/auth"

	"smart-journal/internal/middleware"
)

type AuthSessionHandler struct {
	client          *firebaseauth.Client
	sessionDuration time.Duration
	secureCookie    bool
}

func NewAuthSessionHandler(
	client *firebaseauth.Client,
	sessionDuration time.Duration,
	secureCookie bool,
) *AuthSessionHandler {
	return &AuthSessionHandler{
		client:          client,
		sessionDuration: sessionDuration,
		secureCookie:    secureCookie,
	}
}

func (h *AuthSessionHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/auth/session", h.session)
}

func (h *AuthSessionHandler) session(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.create(w, r)
	case http.MethodGet:
		h.get(w, r)
	case http.MethodDelete:
		h.delete(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *AuthSessionHandler) create(w http.ResponseWriter, r *http.Request) {
	idToken := bearerToken(r.Header.Get("Authorization"))
	if idToken == "" {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}

	decoded, err := h.client.VerifyIDToken(r.Context(), idToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid bearer token")
		return
	}

	sessionCookie, err := h.client.SessionCookie(r.Context(), idToken, h.sessionDuration)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session cookie")
		return
	}

	csrfToken, err := randomToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create csrf token")
		return
	}

	h.setSessionCookie(w, sessionCookie)
	h.setCSRFCookie(w, csrfToken)
	writeJSON(w, http.StatusOK, sessionResponseFromToken(decoded, csrfToken))
}

func (h *AuthSessionHandler) get(w http.ResponseWriter, r *http.Request) {
	token, err := h.verifySessionCookie(r.Context(), r)
	if err != nil {
		middleware.ClearAuthCookies(w, h.secureCookie)
		writeError(w, http.StatusUnauthorized, "invalid session")
		return
	}

	csrfToken := ""
	if cookie, err := r.Cookie(middleware.CSRFCookieName); err == nil {
		csrfToken = cookie.Value
	}
	if csrfToken == "" {
		generated, err := randomToken()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create csrf token")
			return
		}
		csrfToken = generated
		h.setCSRFCookie(w, csrfToken)
	}

	writeJSON(w, http.StatusOK, sessionResponseFromToken(token, csrfToken))
}

func (h *AuthSessionHandler) delete(w http.ResponseWriter, r *http.Request) {
	middleware.ClearAuthCookies(w, h.secureCookie)
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": false})
}

func (h *AuthSessionHandler) verifySessionCookie(ctx context.Context, r *http.Request) (*firebaseauth.Token, error) {
	cookie, err := r.Cookie(middleware.SessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil, errors.New("missing session cookie")
	}
	return h.client.VerifySessionCookie(ctx, cookie.Value)
}

func (h *AuthSessionHandler) setSessionCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(h.sessionDuration.Seconds()),
		Expires:  time.Now().Add(h.sessionDuration),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthSessionHandler) setCSRFCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CSRFCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(h.sessionDuration.Seconds()),
		Expires:  time.Now().Add(h.sessionDuration),
		HttpOnly: false,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func sessionResponseFromToken(token *firebaseauth.Token, csrfToken string) map[string]any {
	return map[string]any{
		"authenticated": true,
		"csrf_token":   csrfToken,
		"user": map[string]any{
			"uid":            token.UID,
			"email":          stringClaim(token, "email"),
			"email_verified": boolClaim(token, "email_verified"),
		},
	}
}

func bearerToken(header string) string {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func stringClaim(token *firebaseauth.Token, key string) string {
	value, ok := token.Claims[key].(string)
	if !ok {
		return ""
	}
	return value
}

func boolClaim(token *firebaseauth.Token, key string) bool {
	value, ok := token.Claims[key].(bool)
	return ok && value
}
