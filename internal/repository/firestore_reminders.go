package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"smart-journal/internal/model"
)

type FirestoreRemindersRepository struct {
	client *firestore.Client
}

func NewFirestoreRemindersRepository(client *firestore.Client) *FirestoreRemindersRepository {
	return &FirestoreRemindersRepository{client: client}
}

func (r *FirestoreRemindersRepository) Create(ctx context.Context, userID string, reminder model.Reminder) (model.Reminder, error) {
	doc, _, err := r.client.Collection("users").Doc(userID).Collection("reminders").Add(ctx, reminder)
	if err != nil {
		return model.Reminder{}, err
	}
	reminder.ID = doc.ID
	reminder.UserID = userID
	normalizeReminderTime(&reminder)
	return reminder, nil
}

func (r *FirestoreRemindersRepository) List(ctx context.Context, userID string, limit int) ([]model.Reminder, error) {
	if limit <= 0 {
		limit = 50
	}

	iter := r.client.Collection("users").Doc(userID).Collection("reminders").
		OrderBy("created_at", firestore.Desc).
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	var reminders []model.Reminder
	for {
		snapshot, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var reminder model.Reminder
		if err := snapshot.DataTo(&reminder); err != nil {
			return nil, err
		}
		reminder.ID = snapshot.Ref.ID
		reminder.UserID = userID
		normalizeReminderTime(&reminder)
		reminders = append(reminders, reminder)
	}

	return reminders, nil
}

func (r *FirestoreRemindersRepository) Get(ctx context.Context, userID string, reminderID string) (model.Reminder, error) {
	snapshot, err := r.client.Collection("users").Doc(userID).Collection("reminders").Doc(reminderID).Get(ctx)
	if isNotFound(err) {
		return model.Reminder{}, model.ErrNotFound
	}
	if err != nil {
		return model.Reminder{}, err
	}

	var reminder model.Reminder
	if err := snapshot.DataTo(&reminder); err != nil {
		return model.Reminder{}, err
	}
	reminder.ID = snapshot.Ref.ID
	reminder.UserID = userID
	normalizeReminderTime(&reminder)
	return reminder, nil
}

func (r *FirestoreRemindersRepository) Update(
	ctx context.Context,
	userID string,
	reminderID string,
	updates map[string]any,
) (model.Reminder, error) {
	ref := r.client.Collection("users").Doc(userID).Collection("reminders").Doc(reminderID)
	if _, err := ref.Update(ctx, toFirestoreUpdates(updates)); isNotFound(err) {
		return model.Reminder{}, model.ErrNotFound
	} else if err != nil {
		return model.Reminder{}, err
	}

	return r.Get(ctx, userID, reminderID)
}

func (r *FirestoreRemindersRepository) Delete(ctx context.Context, userID string, reminderID string) error {
	_, err := r.client.Collection("users").Doc(userID).Collection("reminders").Doc(reminderID).Delete(ctx)
	return err
}

func (r *FirestoreRemindersRepository) Due(ctx context.Context, now time.Time, limit int) ([]model.Reminder, error) {
	if limit <= 0 {
		limit = 50
	}

	iter := r.client.CollectionGroup("reminders").
		Where("status", "==", model.ReminderStatusPending).
		Where("remind_at", "<=", now).
		OrderBy("remind_at", firestore.Asc).
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	var reminders []model.Reminder
	for {
		snapshot, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var reminder model.Reminder
		if err := snapshot.DataTo(&reminder); err != nil {
			return nil, err
		}

		userID, err := userIDFromReminderPath(snapshot.Ref.Path)
		if err != nil {
			return nil, err
		}
		reminder.ID = snapshot.Ref.ID
		reminder.UserID = userID
		normalizeReminderTime(&reminder)
		reminders = append(reminders, reminder)
	}

	return reminders, nil
}

func (r *FirestoreRemindersRepository) DueForUser(
	ctx context.Context,
	userID string,
	now time.Time,
	limit int,
) ([]model.Reminder, error) {
	if limit <= 0 {
		limit = 50
	}

	iter := r.client.Collection("users").Doc(userID).Collection("reminders").
		Where("status", "==", model.ReminderStatusPending).
		Where("remind_at", "<=", now).
		OrderBy("remind_at", firestore.Asc).
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	var reminders []model.Reminder
	for {
		snapshot, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var reminder model.Reminder
		if err := snapshot.DataTo(&reminder); err != nil {
			return nil, err
		}
		reminder.ID = snapshot.Ref.ID
		reminder.UserID = userID
		normalizeReminderTime(&reminder)
		reminders = append(reminders, reminder)
	}

	return reminders, nil
}

func (r *FirestoreRemindersRepository) MarkSent(ctx context.Context, userID string, reminderID string, sentAt time.Time) error {
	_, err := r.client.Collection("users").Doc(userID).Collection("reminders").Doc(reminderID).Update(ctx, []firestore.Update{
		{Path: "status", Value: model.ReminderStatusSent},
		{Path: "sent_at", Value: sentAt},
		{Path: "updated_at", Value: sentAt},
	})
	return err
}

func userIDFromReminderPath(path string) (string, error) {
	parts := strings.Split(path, "/")
	for index := 0; index+3 < len(parts); index++ {
		if parts[index] == "users" && parts[index+2] == "reminders" {
			userID := strings.TrimSpace(parts[index+1])
			if userID != "" {
				return userID, nil
			}
		}
	}
	return "", fmt.Errorf("invalid reminder path %q", path)
}
