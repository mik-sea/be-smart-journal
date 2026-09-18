package model

import (
	"testing"
	"time"
)

func TestAIResultValidateRejectsUnsupportedIntent(t *testing.T) {
	result := AIResult{Intents: []string{"note"}}

	if err := result.Validate(); err == nil {
		t.Fatal("expected unsupported intent error")
	}
}

func TestAIResultValidateRejectsIncomeAndExpenseTogether(t *testing.T) {
	result := AIResult{
		Intents: []string{IntentIncome, IntentExpense},
		FinancialData: &FinancialData{
			Amount: 10000,
		},
	}

	if err := result.Validate(); err == nil {
		t.Fatal("expected mutually exclusive intent error")
	}
}

func TestAIResultValidateRequiresFinancialData(t *testing.T) {
	result := AIResult{Intents: []string{IntentExpense}}

	if err := result.Validate(); err == nil {
		t.Fatal("expected financial data error")
	}
}

func TestAIResultValidateAcceptsReminder(t *testing.T) {
	result := AIResult{
		Intents: []string{IntentReminder},
		ReminderData: &ReminderData{
			RemindAt: time.Now().Add(time.Hour),
			Message:  "Bayar tagihan internet",
		},
	}

	if err := result.Validate(); err != nil {
		t.Fatalf("expected valid reminder result, got %v", err)
	}
}
