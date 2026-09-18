package model

import "time"

const (
	ReminderStatusPending = "pending"
	ReminderStatusSent    = "sent"
)

type Reminder struct {
	ID        string    `json:"id" firestore:"-"`
	UserID    string    `json:"user_id,omitempty" firestore:"-"`
	NoteID    string    `json:"note_id" firestore:"note_id"`
	RemindAt  time.Time `json:"remind_at" firestore:"remind_at"`
	Message   string    `json:"message" firestore:"message"`
	Status    string    `json:"status" firestore:"status"`
	CreatedAt time.Time `json:"created_at" firestore:"created_at"`
	UpdatedAt time.Time `json:"updated_at" firestore:"updated_at"`
	SentAt     *time.Time `json:"sent_at,omitempty" firestore:"sent_at,omitempty"`
}
