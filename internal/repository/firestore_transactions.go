package repository

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"smart-journal/internal/model"
)

type FirestoreTransactionsRepository struct {
	client *firestore.Client
}

func NewFirestoreTransactionsRepository(client *firestore.Client) *FirestoreTransactionsRepository {
	return &FirestoreTransactionsRepository{client: client}
}

func (r *FirestoreTransactionsRepository) Create(
	ctx context.Context,
	userID string,
	transaction model.Transaction,
) (model.Transaction, error) {
	doc, _, err := r.client.Collection("users").Doc(userID).Collection("transactions").Add(ctx, transaction)
	if err != nil {
		return model.Transaction{}, err
	}
	transaction.ID = doc.ID
	normalizeTransactionTime(&transaction)
	return transaction, nil
}

func (r *FirestoreTransactionsRepository) List(ctx context.Context, userID string, limit int) ([]model.Transaction, error) {
	if limit <= 0 {
		limit = 50
	}

	iter := r.client.Collection("users").Doc(userID).Collection("transactions").
		OrderBy("created_at", firestore.Desc).
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	var transactions []model.Transaction
	for {
		snapshot, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var transaction model.Transaction
		if err := snapshot.DataTo(&transaction); err != nil {
			return nil, err
		}
		transaction.ID = snapshot.Ref.ID
		normalizeTransactionTime(&transaction)
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (r *FirestoreTransactionsRepository) Get(
	ctx context.Context,
	userID string,
	transactionID string,
) (model.Transaction, error) {
	snapshot, err := r.client.Collection("users").Doc(userID).Collection("transactions").Doc(transactionID).Get(ctx)
	if isNotFound(err) {
		return model.Transaction{}, model.ErrNotFound
	}
	if err != nil {
		return model.Transaction{}, err
	}

	var transaction model.Transaction
	if err := snapshot.DataTo(&transaction); err != nil {
		return model.Transaction{}, err
	}
	transaction.ID = snapshot.Ref.ID
	normalizeTransactionTime(&transaction)
	return transaction, nil
}

func (r *FirestoreTransactionsRepository) Update(
	ctx context.Context,
	userID string,
	transactionID string,
	updates map[string]any,
) (model.Transaction, error) {
	ref := r.client.Collection("users").Doc(userID).Collection("transactions").Doc(transactionID)
	if _, err := ref.Update(ctx, toFirestoreUpdates(updates)); isNotFound(err) {
		return model.Transaction{}, model.ErrNotFound
	} else if err != nil {
		return model.Transaction{}, err
	}

	return r.Get(ctx, userID, transactionID)
}

func (r *FirestoreTransactionsRepository) Delete(ctx context.Context, userID string, transactionID string) error {
	_, err := r.client.Collection("users").Doc(userID).Collection("transactions").Doc(transactionID).Delete(ctx)
	return err
}
