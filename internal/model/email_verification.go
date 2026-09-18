package model

import "time"

const (
	EmailVerificationStatusPending = "pending"
	EmailVerificationStatusUsed    = "used"
)

type EmailVerification struct {
	TokenHash string     `firestore:"-" json:"token_hash,omitempty"`
	UserID    string     `firestore:"user_id" json:"user_id"`
	Email     string     `firestore:"email" json:"email"`
	Status    string     `firestore:"status" json:"status"`
	ExpiresAt time.Time  `firestore:"expires_at" json:"expires_at"`
	CreatedAt time.Time  `firestore:"created_at" json:"created_at"`
	UsedAt    *time.Time `firestore:"used_at,omitempty" json:"used_at,omitempty"`
}
