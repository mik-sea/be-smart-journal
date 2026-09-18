package middleware

import (
	"crypto/subtle"
	"net/http"
)

type CSRFMiddleware struct{}

func NewCSRFMiddleware() *CSRFMiddleware {
	return &CSRFMiddleware{}
}

func (m *CSRFMiddleware) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requiresCSRF(r) {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(CSRFCookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, "missing csrf cookie", http.StatusForbidden)
			return
		}

		header := r.Header.Get("X-CSRF-Token")
		if header == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func requiresCSRF(r *http.Request) bool {
	if r.Method == http.MethodPost && r.URL.Path == "/api/auth/session" {
		return false
	}

	switch r.Method {
	case http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}
