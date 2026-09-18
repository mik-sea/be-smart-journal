package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"

	"smart-journal/configs"
	"smart-journal/internal/handler"
	"smart-journal/internal/middleware"
	"smart-journal/internal/repository"
	"smart-journal/internal/scheduler"
	"smart-journal/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	loadDotEnv(logger)

	cfg := configs.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	authClient, firestoreClient, err := newFirebaseClients(ctx, cfg)
	if err != nil {
		logger.Error("firebase initialization failed", "error", err)
		os.Exit(1)
	}
	defer firestoreClient.Close()

	httpClient := &http.Client{Timeout: cfg.RequestTimeout}

	notes := repository.NewFirestoreNotesRepository(firestoreClient)
	transactions := repository.NewFirestoreTransactionsRepository(firestoreClient)
	reminders := repository.NewFirestoreRemindersRepository(firestoreClient)
	profiles := repository.NewFirestoreProfilesRepository(firestoreClient)
	emailVerifications := repository.NewFirestoreEmailVerificationsRepository(firestoreClient)
	emailSender := newEmailSender(cfg, httpClient)

	aiService := service.NewGeminiService(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.GeminiBaseURL, httpClient)
	journalService := service.NewJournalService(aiService, notes, transactions, reminders, profiles)
	noteService := service.NewNoteService(notes)
	transactionService := service.NewTransactionService(transactions)
	reminderService := service.NewReminderService(
		profiles,
		reminders,
		emailSender,
		cfg.EmailFrom,
		cfg.NotificationToEmail,
	)
	profileService := service.NewProfileService(profiles)
	emailVerificationService := service.NewEmailVerificationService(
		authClient,
		profiles,
		emailVerifications,
		emailSender,
		cfg.EmailFrom,
		cfg.EmailVerificationBaseURL,
		cfg.EmailVerificationDuration,
	)

	if cfg.SchedulerEnabled {
		go scheduler.New(reminderService, cfg.SchedulerInterval, logger).Run(ctx)
	}

	mux := http.NewServeMux()
	secureCookie := cfg.CookieSecure()
	authMiddleware := middleware.NewFirebaseAuthMiddleware(authClient, secureCookie)
	handler.NewAuthSessionHandler(authClient, cfg.FirebaseSessionDuration(), secureCookie).Register(mux)
	handler.NewEmailVerificationHandler(
		emailVerificationService,
		cfg.EmailVerificationRedirectURL,
	).Register(mux, authMiddleware.RequireAuth)
	handler.NewJournalHandler(journalService).Register(mux, authMiddleware.RequireAuth)
	handler.NewNoteHandler(noteService).Register(mux, authMiddleware.RequireAuth)
	handler.NewTransactionHandler(transactionService).Register(mux, authMiddleware.RequireAuth)
	handler.NewReminderHandler(reminderService).Register(mux, authMiddleware.RequireAuth)
	handler.NewProfileHandler(profileService).Register(mux, authMiddleware.RequireAuth)
	mux.HandleFunc("/healthz", health)

	rootHandler := middleware.NewCORSMiddleware(cfg.AllowedOrigins).Handle(
		middleware.NewCSRFMiddleware().Protect(mux),
	)

	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           rootHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("server started", "address", cfg.Address())
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", "error", err)
	}
}

func newEmailSender(cfg configs.Config, httpClient *http.Client) service.EmailSender {
	switch strings.ToLower(strings.TrimSpace(cfg.EmailProvider)) {
	case "smtp", "resend_smtp":
		return service.NewSMTPEmailSender(
			cfg.SMTPHost,
			cfg.SMTPPort,
			cfg.SMTPUsername,
			cfg.SMTPPassword,
			cfg.SMTPImplicitTLS,
		)
	default:
		return service.NewResendAPIEmailSender(cfg.ResendAPIKey, cfg.ResendAPIBaseURL, httpClient)
	}
}

func loadDotEnv(logger *slog.Logger) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		logger.Warn("failed to load .env file", "error", err)
	}
}

func newFirebaseClients(ctx context.Context, cfg configs.Config) (*auth.Client, *firestore.Client, error) {
	var opts []option.ClientOption
	if cfg.FirebaseCredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.FirebaseCredentialsFile))
	}

	var firebaseConfig *firebase.Config
	if cfg.FirebaseProjectID != "" {
		firebaseConfig = &firebase.Config{ProjectID: cfg.FirebaseProjectID}
	}

	app, err := firebase.NewApp(ctx, firebaseConfig, opts...)
	if err != nil {
		return nil, nil, err
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, nil, err
	}

	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		return nil, nil, err
	}

	return authClient, firestoreClient, nil
}

func health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
