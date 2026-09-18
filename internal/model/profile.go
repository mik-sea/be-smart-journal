package model

import "time"

type UserProfile struct {
	UserID          string     `firestore:"-" json:"user_id"`
	Email           string     `firestore:"email" json:"email"`
	EmailVerifiedAt *time.Time `firestore:"email_verified_at,omitempty" json:"email_verified_at,omitempty"`
	Name            string     `firestore:"name" json:"name"`
	AvatarURL       string     `firestore:"avatar_url" json:"avatar_url"`
	Bio             string     `firestore:"bio" json:"bio"`
	Timezone        string     `firestore:"timezone" json:"timezone"`
	CreatedAt       time.Time  `firestore:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `firestore:"updated_at" json:"updated_at"`
}
