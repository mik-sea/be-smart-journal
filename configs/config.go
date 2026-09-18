package configs

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains all runtime settings needed by the API server.
// Values are loaded from environment variables so the binary can run cleanly on a VPS.
type Config struct {
	AppEnv                    string
	Host                      string
	Port                      string
	AllowedOrigins            []string
	FrontendBaseURL           string
	FirebaseCredentialsFile   string
	FirebaseProjectID         string
	GeminiAPIKey              string
	GeminiModel               string
	GeminiBaseURL             string
	RequestTimeout            time.Duration
	SessionDuration           time.Duration
	EmailProvider             string
	ResendAPIKey              string
	ResendAPIBaseURL          string
	EmailFrom                 string
	NotificationToEmail       string
	SMTPHost                  string
	SMTPPort                  string
	SMTPUsername              string
	SMTPPassword              string
	SMTPImplicitTLS           bool
	EmailVerificationBaseURL  string
	EmailVerificationRedirectURL string
	EmailVerificationDuration time.Duration
	SchedulerEnabled          bool
	SchedulerInterval         time.Duration
}

func Load() Config {
	return Config{
		AppEnv:                    getEnv("APP_ENV", "development"),
		Host:                      getEnv("HOST", "127.0.0.1"),
		Port:                      getEnv("PORT", "8080"),
		AllowedOrigins:            getCSVEnv("CORS_ALLOWED_ORIGINS", "http://127.0.0.1:5173"),
		FrontendBaseURL:           getEnv("FRONTEND_BASE_URL", "http://127.0.0.1:5173"),
		FirebaseCredentialsFile:   getEnv("FIREBASE_CREDENTIALS_FILE", ""),
		FirebaseProjectID:         getEnv("FIREBASE_PROJECT_ID", ""),
		GeminiAPIKey:              getEnv("GEMINI_API_KEY", ""),
		GeminiModel:               getEnv("GEMINI_MODEL", "gemini-2.0-flash"),
		GeminiBaseURL:             getEnv("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta"),
		RequestTimeout:            getDurationEnv("REQUEST_TIMEOUT_SECONDS", 20) * time.Second,
		SessionDuration:           getDurationEnv("SESSION_DURATION_HOURS", 120) * time.Hour,
		EmailProvider:             getEnv("EMAIL_PROVIDER", "resend_api"),
		ResendAPIKey:              getEnv("RESEND_API_KEY", ""),
		ResendAPIBaseURL:          getEnv("RESEND_API_BASE_URL", "https://api.resend.com"),
		EmailFrom:                 getEnv("EMAIL_FROM", "Smart Journal <onboarding@resend.dev>"),
		NotificationToEmail:       getEnv("NOTIFICATION_TO_EMAIL", ""),
		SMTPHost:                  getEnv("SMTP_HOST", "smtp.resend.com"),
		SMTPPort:                  getEnv("SMTP_PORT", "465"),
		SMTPUsername:              getEnv("SMTP_USERNAME", "resend"),
		SMTPPassword:              getEnv("SMTP_PASSWORD", getEnv("RESEND_API_KEY", "")),
		SMTPImplicitTLS:           getBoolEnv("SMTP_IMPLICIT_TLS", true),
		EmailVerificationBaseURL:  getEnv("EMAIL_VERIFICATION_BASE_URL", "http://127.0.0.1:8080"),
		EmailVerificationRedirectURL: getEnv(
			"EMAIL_VERIFICATION_REDIRECT_URL",
			joinURL(getEnv("FRONTEND_BASE_URL", "http://127.0.0.1:5173"), "/email-verified"),
		),
		EmailVerificationDuration: getDurationEnv("EMAIL_VERIFICATION_DURATION_HOURS", 24) * time.Hour,
		SchedulerEnabled:          getBoolEnv("SCHEDULER_ENABLED", true),
		SchedulerInterval:         getDurationEnv("SCHEDULER_INTERVAL_SECONDS", 60) * time.Second,
	}
}

func (c Config) Address() string {
	return c.Host + ":" + c.Port
}

func (c Config) CookieSecure() bool {
	return strings.EqualFold(c.AppEnv, "production") || getBoolEnv("COOKIE_SECURE", false)
}

func (c Config) FirebaseSessionDuration() time.Duration {
	const maxFirebaseSessionDuration = 14 * 24 * time.Hour
	if c.SessionDuration > maxFirebaseSessionDuration {
		return maxFirebaseSessionDuration
	}
	return c.SessionDuration
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getCSVEnv(key string, fallback string) []string {
	value := getEnv(key, fallback)
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" && item != "*" {
			result = append(result, item)
		}
	}
	return result
}

func getDurationEnv(key string, fallback int64) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return time.Duration(fallback)
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return time.Duration(fallback)
	}

	return time.Duration(parsed)
}

func getBoolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func joinURL(base string, path string) string {
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}
