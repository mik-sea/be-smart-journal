package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"smart-journal/internal/model"
)

type TransactionService struct {
	transactions TransactionRepository
	now          func() time.Time
}

type CreateTransactionInput struct {
	UserID   string
	NoteID   string
	Type     string
	Amount   float64
	Category string
}

type UpdateTransactionInput struct {
	UserID        string
	TransactionID string
	NoteID        *string
	Type          *string
	Amount        *float64
	Category      *string
}

func NewTransactionService(transactions TransactionRepository) *TransactionService {
	return &TransactionService{
		transactions: transactions,
		now:          time.Now,
	}
}

func (s *TransactionService) Create(ctx context.Context, input CreateTransactionInput) (model.Transaction, error) {
	if input.UserID == "" {
		return model.Transaction{}, errors.New("user id is required")
	}

	transactionType, err := normalizeTransactionType(input.Type)
	if err != nil {
		return model.Transaction{}, err
	}
	if input.Amount <= 0 {
		return model.Transaction{}, errors.New("amount must be greater than zero")
	}

	now := s.now().UTC()
	return s.transactions.Create(ctx, input.UserID, model.Transaction{
		NoteID:    strings.TrimSpace(input.NoteID),
		Type:      transactionType,
		Amount:    input.Amount,
		Category:  normalizeCategory(input.Category),
		CreatedAt: now,
		UpdatedAt: now,
	})
}

func (s *TransactionService) List(ctx context.Context, userID string, limit int) ([]model.Transaction, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	return s.transactions.List(ctx, userID, limit)
}

func (s *TransactionService) Get(ctx context.Context, userID string, transactionID string) (model.Transaction, error) {
	if userID == "" {
		return model.Transaction{}, errors.New("user id is required")
	}
	if transactionID == "" {
		return model.Transaction{}, errors.New("transaction id is required")
	}
	return s.transactions.Get(ctx, userID, transactionID)
}

func (s *TransactionService) Update(ctx context.Context, input UpdateTransactionInput) (model.Transaction, error) {
	if input.UserID == "" {
		return model.Transaction{}, errors.New("user id is required")
	}
	if input.TransactionID == "" {
		return model.Transaction{}, errors.New("transaction id is required")
	}

	updates := map[string]any{
		"updated_at": s.now().UTC(),
	}
	if input.NoteID != nil {
		updates["note_id"] = strings.TrimSpace(*input.NoteID)
	}
	if input.Type != nil {
		transactionType, err := normalizeTransactionType(*input.Type)
		if err != nil {
			return model.Transaction{}, err
		}
		updates["type"] = transactionType
	}
	if input.Amount != nil {
		if *input.Amount <= 0 {
			return model.Transaction{}, errors.New("amount must be greater than zero")
		}
		updates["amount"] = *input.Amount
	}
	if input.Category != nil {
		updates["category"] = normalizeCategory(*input.Category)
	}
	if len(updates) == 1 {
		return s.Get(ctx, input.UserID, input.TransactionID)
	}

	return s.transactions.Update(ctx, input.UserID, input.TransactionID, updates)
}

func (s *TransactionService) Delete(ctx context.Context, userID string, transactionID string) error {
	if userID == "" {
		return errors.New("user id is required")
	}
	if transactionID == "" {
		return errors.New("transaction id is required")
	}
	return s.transactions.Delete(ctx, userID, transactionID)
}

func normalizeTransactionType(value string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case model.TransactionIncome:
		return model.TransactionIncome, nil
	case model.TransactionExpense:
		return model.TransactionExpense, nil
	default:
		return "", errors.New("type must be income or expense")
	}
}

func normalizeCategory(value string) string {
	category := strings.TrimSpace(value)
	if category == "" {
		return "uncategorized"
	}
	return category
}
