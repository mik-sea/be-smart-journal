package middleware

import (
	"context"
	"net/http"

	firebaseauth "firebase.google.com/go/v4/auth"
)

type contextKey string

const (
	SessionCookieName = "smart_journal_session"
	CSRFCookieName    = "smart_journal_csrf"

	userIDContextKey contextKey = "user_id"
)

type FirebaseAuthMiddleware struct {
	client       *firebaseauth.Client
	secureCookie bool
}

func NewFirebaseAuthMiddleware(client *firebaseauth.Client, secureCookie bool) *FirebaseAuthMiddleware {
	return &FirebaseAuthMiddleware{
		client:       client,
		secureCookie: secureCookie,
	}
}

func (m *FirebaseAuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, "missing session cookie", http.StatusUnauthorized)
			return
		}

		decoded, err := m.client.VerifySessionCookie(r.Context(), cookie.Value)
		if err != nil {
			ClearAuthCookies(w, m.secureCookie)
			http.Error(w, "invalid session cookie", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, decoded.UID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok && userID != ""
}

func ClearAuthCookies(w http.ResponseWriter, secure bool) {
	for _, name := range []string{SessionCookieName, CSRFCookieName} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: name == SessionCookieName,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}
