package repository

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"smart-journal/internal/model"
)

type FirestoreNotesRepository struct {
	client *firestore.Client
}

func NewFirestoreNotesRepository(client *firestore.Client) *FirestoreNotesRepository {
	return &FirestoreNotesRepository{client: client}
}

func (r *FirestoreNotesRepository) Create(ctx context.Context, userID string, note model.Note) (model.Note, error) {
	doc, _, err := r.client.Collection("users").Doc(userID).Collection("notes").Add(ctx, note)
	if err != nil {
		return model.Note{}, err
	}
	note.ID = doc.ID
	normalizeNoteTime(&note)
	return note, nil
}

func (r *FirestoreNotesRepository) List(ctx context.Context, userID string, limit int) ([]model.Note, error) {
	if limit <= 0 {
		limit = 50
	}

	iter := r.client.Collection("users").Doc(userID).Collection("notes").
		OrderBy("created_at", firestore.Desc).
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	var notes []model.Note
	for {
		snapshot, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var note model.Note
		if err := snapshot.DataTo(&note); err != nil {
			return nil, err
		}
		note.ID = snapshot.Ref.ID
		normalizeNoteTime(&note)
		notes = append(notes, note)
	}

	return notes, nil
}

func (r *FirestoreNotesRepository) Get(ctx context.Context, userID string, noteID string) (model.Note, error) {
	snapshot, err := r.client.Collection("users").Doc(userID).Collection("notes").Doc(noteID).Get(ctx)
	if isNotFound(err) {
		return model.Note{}, model.ErrNotFound
	}
	if err != nil {
		return model.Note{}, err
	}

	var note model.Note
	if err := snapshot.DataTo(&note); err != nil {
		return model.Note{}, err
	}
	note.ID = snapshot.Ref.ID
	normalizeNoteTime(&note)
	return note, nil
}

func (r *FirestoreNotesRepository) Update(
	ctx context.Context,
	userID string,
	noteID string,
	updates map[string]any,
) (model.Note, error) {
	ref := r.client.Collection("users").Doc(userID).Collection("notes").Doc(noteID)
	if _, err := ref.Update(ctx, toFirestoreUpdates(updates)); isNotFound(err) {
		return model.Note{}, model.ErrNotFound
	} else if err != nil {
		return model.Note{}, err
	}

	return r.Get(ctx, userID, noteID)
}

func (r *FirestoreNotesRepository) Delete(ctx context.Context, userID string, noteID string) error {
	_, err := r.client.Collection("users").Doc(userID).Collection("notes").Doc(noteID).Delete(ctx)
	return err
}
