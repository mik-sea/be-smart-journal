package repository

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"smart-journal/internal/model"
)

const profileDocumentID = "settings"

type FirestoreProfilesRepository struct {
	client *firestore.Client
}

func NewFirestoreProfilesRepository(client *firestore.Client) *FirestoreProfilesRepository {
	return &FirestoreProfilesRepository{client: client}
}

func (r *FirestoreProfilesRepository) Get(ctx context.Context, userID string) (model.UserProfile, error) {
	snapshot, err := r.client.Collection("users").Doc(userID).Collection("profile").Doc(profileDocumentID).Get(ctx)
	if isNotFound(err) {
		return model.UserProfile{}, model.ErrNotFound
	}
	if err != nil {
		return model.UserProfile{}, err
	}

	var profile model.UserProfile
	if err := snapshot.DataTo(&profile); err != nil {
		return model.UserProfile{}, err
	}
	profile.UserID = userID
	normalizeProfileTime(&profile)
	return profile, nil
}

func (r *FirestoreProfilesRepository) Create(ctx context.Context, userID string, profile model.UserProfile) (model.UserProfile, error) {
	_, err := r.client.Collection("users").Doc(userID).Collection("profile").Doc(profileDocumentID).Set(ctx, profile)
	if err != nil {
		return model.UserProfile{}, err
	}
	profile.UserID = userID
	normalizeProfileTime(&profile)
	return profile, nil
}

func (r *FirestoreProfilesRepository) Update(
	ctx context.Context,
	userID string,
	updates map[string]any,
) (model.UserProfile, error) {
	ref := r.client.Collection("users").Doc(userID).Collection("profile").Doc(profileDocumentID)
	if _, err := ref.Update(ctx, toFirestoreUpdates(updates)); isNotFound(err) {
		return model.UserProfile{}, model.ErrNotFound
	} else if err != nil {
		return model.UserProfile{}, err
	}

	return r.Get(ctx, userID)
}

func (r *FirestoreProfilesRepository) MarkEmailVerified(
	ctx context.Context,
	userID string,
	verifiedAt time.Time,
) (model.UserProfile, error) {
	return r.Update(ctx, userID, map[string]any{
		"email_verified_at": verifiedAt,
		"updated_at":         verifiedAt,
	})
}
