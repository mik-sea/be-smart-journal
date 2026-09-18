package repository

import "testing"

func TestUserIDFromReminderPathSupportsShortPath(t *testing.T) {
	userID, err := userIDFromReminderPath("users/user-123/reminders/reminder-456")
	if err != nil {
		t.Fatal(err)
	}
	if userID != "user-123" {
		t.Fatalf("expected user-123, got %s", userID)
	}
}

func TestUserIDFromReminderPathSupportsFirestoreResourcePath(t *testing.T) {
	userID, err := userIDFromReminderPath(
		"projects/smartjournalbackend/databases/(default)/documents/users/user-123/reminders/reminder-456",
	)
	if err != nil {
		t.Fatal(err)
	}
	if userID != "user-123" {
		t.Fatalf("expected user-123, got %s", userID)
	}
}
