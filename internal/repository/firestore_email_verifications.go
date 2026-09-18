package repository

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"smart-journal/internal/model"
)

type FirestoreEmailVerificationsRepository struct {
	client *firestore.Client
}

func NewFirestoreEmailVerificationsRepository(client *firestore.Client) *FirestoreEmailVerificationsRepository {
	return &FirestoreEmailVerificationsRepository{client: client}
}

func (r *FirestoreEmailVerificationsRepository) Create(
	ctx context.Context,
	verification model.EmailVerification,
) (model.EmailVerification, error) {
	_, err := r.collection().Doc(verification.TokenHash).Set(ctx, verification)
	if err != nil {
		return model.EmailVerification{}, err
	}
	return verification, nil
}

func (r *FirestoreEmailVerificationsRepository) Get(ctx context.Context, tokenHash string) (model.EmailVerification, error) {
	snapshot, err := r.collection().Doc(tokenHash).Get(ctx)
	if isNotFound(err) {
		return model.EmailVerification{}, model.ErrNotFound
	}
	if err != nil {
		return model.EmailVerification{}, err
	}

	var verification model.EmailVerification
	if err := snapshot.DataTo(&verification); err != nil {
		return model.EmailVerification{}, err
	}
	verification.TokenHash = snapshot.Ref.ID
	return verification, nil
}

func (r *FirestoreEmailVerificationsRepository) MarkUsed(ctx context.Context, tokenHash string, usedAt time.Time) error {
	_, err := r.collection().Doc(tokenHash).Update(ctx, []firestore.Update{
		{Path: "status", Value: model.EmailVerificationStatusUsed},
		{Path: "used_at", Value: usedAt},
	})
	return err
}

func (r *FirestoreEmailVerificationsRepository) collection() *firestore.CollectionRef {
	return r.client.Collection("email_verifications")
}
