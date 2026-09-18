package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"smart-journal/internal/model"
)

const MaxOmniInputLength = 4000

type NoteRepository interface {
	Create(ctx context.Context, userID string, note model.Note) (model.Note, error)
	List(ctx context.Context, userID string, limit int) ([]model.Note, error)
	Get(ctx context.Context, userID string, noteID string) (model.Note, error)
	Update(ctx context.Context, userID string, noteID string, updates map[string]any) (model.Note, error)
	Delete(ctx context.Context, userID string, noteID string) error
}

type TransactionRepository interface {
	Create(ctx context.Context, userID string, transaction model.Transaction) (model.Transaction, error)
	List(ctx context.Context, userID string, limit int) ([]model.Transaction, error)
	Get(ctx context.Context, userID string, transactionID string) (model.Transaction, error)
	Update(ctx context.Context, userID string, transactionID string, updates map[string]any) (model.Transaction, error)
	Delete(ctx context.Context, userID string, transactionID string) error
}

type ReminderRepository interface {
	Create(ctx context.Context, userID string, reminder model.Reminder) (model.Reminder, error)
	List(ctx context.Context, userID string, limit int) ([]model.Reminder, error)
	Get(ctx context.Context, userID string, reminderID string) (model.Reminder, error)
	Update(ctx context.Context, userID string, reminderID string, updates map[string]any) (model.Reminder, error)
	Delete(ctx context.Context, userID string, reminderID string) error
	Due(ctx context.Context, now time.Time, limit int) ([]model.Reminder, error)
	DueForUser(ctx context.Context, userID string, now time.Time, limit int) ([]model.Reminder, error)
	MarkSent(ctx context.Context, userID string, reminderID string, sentAt time.Time) error
}

type JournalService struct {
	aiParser      AIParser
	notes         NoteRepository
	transactions  TransactionRepository
	reminders     ReminderRepository
	profiles      ProfileReader
	now           func() time.Time
}

type CreateJournalInput struct {
	UserID  string
	RawText string
}

type CreateJournalResult struct {
	Note         model.Note          `json:"note"`
	AIResult     model.AIResult      `json:"ai_result"`
	Transactions []model.Transaction `json:"transactions"`
	Reminders    []model.Reminder    `json:"reminders"`
}

func NewJournalService(
	aiParser AIParser,
	notes NoteRepository,
	transactions TransactionRepository,
	reminders ReminderRepository,
	profiles ProfileReader,
) *JournalService {
	return &JournalService{
		aiParser:     aiParser,
		notes:        notes,
		transactions: transactions,
		reminders:    reminders,
		profiles:     profiles,
		now:          time.Now,
	}
}

func (s *JournalService) Create(ctx context.Context, input CreateJournalInput) (CreateJournalResult, error) {
	input.RawText = strings.TrimSpace(input.RawText)
	if input.UserID == "" {
		return CreateJournalResult{}, errors.New("user id is required")
	}
	if input.RawText == "" {
		return CreateJournalResult{}, errors.New("raw_text is required")
	}
	if len(input.RawText) > MaxOmniInputLength {
		return CreateJournalResult{}, fmt.Errorf("raw_text must be at most %d characters", MaxOmniInputLength)
	}

	now := s.now()
	nowUTC := now.UTC()
	userTimezone, userLocation := s.userTimezone(ctx, input.UserID)
	aiResult, err := s.aiParser.Parse(ctx, input.RawText, now.In(userLocation), userTimezone)
	if err != nil {
		return CreateJournalResult{}, err
	}
	if aiResult.HasIntent(model.IntentReminder) && aiResult.ReminderData != nil {
		aiResult.ReminderData.RemindAt = normalizeAIReminderTime(
			aiResult.ReminderData.RemindAt,
			userLocation,
			input.RawText,
		)
	}

	note, err := s.notes.Create(ctx, input.UserID, model.Note{
		RawText:   input.RawText,
		Title:     aiResult.Title,
		CreatedAt: nowUTC,
		UpdatedAt: nowUTC,
	})
	if err != nil {
		return CreateJournalResult{}, err
	}

	result := CreateJournalResult{
		Note:     note,
		AIResult: aiResult,
	}

	if aiResult.HasIntent(model.IntentIncome) {
		transaction, err := s.createTransaction(ctx, input.UserID, note.ID, model.TransactionIncome, *aiResult.FinancialData, nowUTC)
		if err != nil {
			return CreateJournalResult{}, err
		}
		result.Transactions = append(result.Transactions, transaction)
	}

	if aiResult.HasIntent(model.IntentExpense) {
		transaction, err := s.createTransaction(ctx, input.UserID, note.ID, model.TransactionExpense, *aiResult.FinancialData, nowUTC)
		if err != nil {
			return CreateJournalResult{}, err
		}
		result.Transactions = append(result.Transactions, transaction)
	}

	if aiResult.HasIntent(model.IntentReminder) {
		reminder, err := s.reminders.Create(ctx, input.UserID, model.Reminder{
			NoteID:    note.ID,
			RemindAt:  aiResult.ReminderData.RemindAt.UTC(),
			Message:   aiResult.ReminderData.Message,
			Status:    model.ReminderStatusPending,
			CreatedAt: nowUTC,
			UpdatedAt: nowUTC,
		})
		if err != nil {
			return CreateJournalResult{}, err
		}
		result.Reminders = append(result.Reminders, reminder)
	}

	return result, nil
}

func (s *JournalService) createTransaction(
	ctx context.Context,
	userID string,
	noteID string,
	transactionType string,
	financialData model.FinancialData,
	now time.Time,
) (model.Transaction, error) {
	category := strings.TrimSpace(financialData.Category)
	if category == "" {
		category = "uncategorized"
	}

	return s.transactions.Create(ctx, userID, model.Transaction{
		NoteID:    noteID,
		Type:      transactionType,
		Amount:    financialData.Amount,
		Category:  category,
		CreatedAt: now,
		UpdatedAt: now,
	})
}

func (s *JournalService) userTimezone(ctx context.Context, userID string) (string, *time.Location) {
	timezone := DefaultUserTimezone
	if s.profiles != nil {
		profile, err := s.profiles.Get(ctx, userID)
		if err == nil && strings.TrimSpace(profile.Timezone) != "" {
			timezone = strings.TrimSpace(profile.Timezone)
		}
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = defaultUserLocation()
		timezone = DefaultUserTimezone
	}

	return timezone, location
}
