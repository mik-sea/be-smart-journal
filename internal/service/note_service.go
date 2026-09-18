package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"smart-journal/internal/model"
)

type NoteService struct {
	notes NoteRepository
	now   func() time.Time
}

type CreateNoteInput struct {
	UserID  string
	RawText string
	Title   string
}

type UpdateNoteInput struct {
	UserID  string
	NoteID  string
	RawText *string
	Title   *string
}

func NewNoteService(notes NoteRepository) *NoteService {
	return &NoteService{
		notes: notes,
		now:   time.Now,
	}
}

func (s *NoteService) Create(ctx context.Context, input CreateNoteInput) (model.Note, error) {
	rawText := strings.TrimSpace(input.RawText)
	if input.UserID == "" {
		return model.Note{}, errors.New("user id is required")
	}
	if rawText == "" {
		return model.Note{}, errors.New("raw_text is required")
	}

	now := s.now().UTC()
	return s.notes.Create(ctx, input.UserID, model.Note{
		RawText:   rawText,
		Title:     strings.TrimSpace(input.Title),
		CreatedAt: now,
		UpdatedAt: now,
	})
}

func (s *NoteService) List(ctx context.Context, userID string, limit int) ([]model.Note, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	return s.notes.List(ctx, userID, limit)
}

func (s *NoteService) Get(ctx context.Context, userID string, noteID string) (model.Note, error) {
	if userID == "" {
		return model.Note{}, errors.New("user id is required")
	}
	if noteID == "" {
		return model.Note{}, errors.New("note id is required")
	}
	return s.notes.Get(ctx, userID, noteID)
}

func (s *NoteService) Update(ctx context.Context, input UpdateNoteInput) (model.Note, error) {
	if input.UserID == "" {
		return model.Note{}, errors.New("user id is required")
	}
	if input.NoteID == "" {
		return model.Note{}, errors.New("note id is required")
	}

	updates := map[string]any{
		"updated_at": s.now().UTC(),
	}
	if input.RawText != nil {
		rawText := strings.TrimSpace(*input.RawText)
		if rawText == "" {
			return model.Note{}, errors.New("raw_text cannot be empty")
		}
		updates["raw_text"] = rawText
	}
	if input.Title != nil {
		updates["title"] = strings.TrimSpace(*input.Title)
	}
	if len(updates) == 1 {
		return s.Get(ctx, input.UserID, input.NoteID)
	}

	return s.notes.Update(ctx, input.UserID, input.NoteID, updates)
}

func (s *NoteService) Delete(ctx context.Context, userID string, noteID string) error {
	if userID == "" {
		return errors.New("user id is required")
	}
	if noteID == "" {
		return errors.New("note id is required")
	}
	return s.notes.Delete(ctx, userID, noteID)
}
