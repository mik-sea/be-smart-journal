package model

import "time"

type Note struct {
	ID        string    `json:"id" firestore:"-"`
	RawText   string    `json:"raw_text" firestore:"raw_text"`
	Title     string    `json:"title" firestore:"title"`
	CreatedAt time.Time `json:"created_at" firestore:"created_at"`
	UpdatedAt time.Time `json:"updated_at" firestore:"updated_at"`
}
