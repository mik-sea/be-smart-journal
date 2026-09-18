package model

import "time"

const (
	TransactionIncome  = "income"
	TransactionExpense = "expense"
)

type Transaction struct {
	ID        string    `json:"id" firestore:"-"`
	NoteID    string    `json:"note_id" firestore:"note_id"`
	Type      string    `json:"type" firestore:"type"`
	Amount    float64   `json:"amount" firestore:"amount"`
	Category  string    `json:"category" firestore:"category"`
	CreatedAt time.Time `json:"created_at" firestore:"created_at"`
	UpdatedAt time.Time `json:"updated_at" firestore:"updated_at"`
}
