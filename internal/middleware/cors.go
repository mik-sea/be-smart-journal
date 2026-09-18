package middleware

import "net/http"

type CORSMiddleware struct {
	allowedOrigins map[string]struct{}
}

func NewCORSMiddleware(origins []string) *CORSMiddleware {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		if origin != "" && origin != "*" {
			allowed[origin] = struct{}{}
		}
	}
	return &CORSMiddleware{allowedOrigins: allowed}
}

func (m *CORSMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := m.allowedOrigins[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
			w.Header().Set("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			if origin != "" {
				if _, ok := m.allowedOrigins[origin]; !ok {
					http.Error(w, "origin not allowed", http.StatusForbidden)
					return
				}
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
