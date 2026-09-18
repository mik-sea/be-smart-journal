package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	IntentIncome   = "income"
	IntentExpense  = "expense"
	IntentReminder = "reminder"

	MaxAITitleLength           = 120
	MaxAICategoryLength        = 80
	MaxAIReminderMessageLength = 300
)

type AIResult struct {
	Intents       []string       `json:"intents"`
	Title         string         `json:"title"`
	FinancialData *FinancialData `json:"financial_data,omitempty"`
	ReminderData  *ReminderData  `json:"reminder_data,omitempty"`
}

type FinancialData struct {
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
}

type ReminderData struct {
	RemindAt time.Time `json:"remind_at"`
	Message  string    `json:"message"`
}

func (r AIResult) HasIntent(intent string) bool {
	for _, current := range r.Intents {
		if current == intent {
			return true
		}
	}
	return false
}

func (r AIResult) Validate() error {
	if len(r.Intents) == 0 {
		return nil
	}

	if len(strings.TrimSpace(r.Title)) > MaxAITitleLength {
		return fmt.Errorf("title must be at most %d characters", MaxAITitleLength)
	}

	seen := make(map[string]bool, len(r.Intents))
	for _, intent := range r.Intents {
		switch intent {
		case IntentIncome, IntentExpense, IntentReminder:
			if seen[intent] {
				return fmt.Errorf("duplicate intent %q", intent)
			}
			seen[intent] = true
		default:
			return fmt.Errorf("unsupported intent %q", intent)
		}
	}

	if seen[IntentIncome] && seen[IntentExpense] {
		return errors.New("income and expense cannot be used together")
	}

	if seen[IntentIncome] || seen[IntentExpense] {
		if r.FinancialData == nil {
			return errors.New("financial_data is required for income or expense intent")
		}
		if r.FinancialData.Amount <= 0 {
			return errors.New("financial_data.amount must be greater than zero")
		}
		if len(strings.TrimSpace(r.FinancialData.Category)) > MaxAICategoryLength {
			return fmt.Errorf("financial_data.category must be at most %d characters", MaxAICategoryLength)
		}
	}

	if seen[IntentReminder] {
		if r.ReminderData == nil {
			return errors.New("reminder_data is required for reminder intent")
		}
		if r.ReminderData.RemindAt.IsZero() {
			return errors.New("reminder_data.remind_at is required")
		}
		message := strings.TrimSpace(r.ReminderData.Message)
		if message == "" {
			return errors.New("reminder_data.message is required")
		}
		if len(message) > MaxAIReminderMessageLength {
			return fmt.Errorf("reminder_data.message must be at most %d characters", MaxAIReminderMessageLength)
		}
	}

	return nil
}
