package repository

import (
	"errors"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"smart-journal/internal/model"
)

func isNotFound(err error) bool {
	return errors.Is(err, model.ErrNotFound) || status.Code(err) == codes.NotFound
}

func toFirestoreUpdates(values map[string]any) []firestore.Update {
	updates := make([]firestore.Update, 0, len(values))
	for path, value := range values {
		updates = append(updates, firestore.Update{Path: path, Value: value})
	}
	return updates
}
