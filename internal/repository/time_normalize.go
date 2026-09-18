package repository

import "smart-journal/internal/model"

func normalizeNoteTime(note *model.Note) {
	note.CreatedAt = note.CreatedAt.UTC()
	note.UpdatedAt = note.UpdatedAt.UTC()
}

func normalizeTransactionTime(transaction *model.Transaction) {
	transaction.CreatedAt = transaction.CreatedAt.UTC()
	transaction.UpdatedAt = transaction.UpdatedAt.UTC()
}

func normalizeReminderTime(reminder *model.Reminder) {
	reminder.RemindAt = reminder.RemindAt.UTC()
	reminder.CreatedAt = reminder.CreatedAt.UTC()
	reminder.UpdatedAt = reminder.UpdatedAt.UTC()
	if reminder.SentAt != nil {
		sentAt := reminder.SentAt.UTC()
		reminder.SentAt = &sentAt
	}
}

func normalizeProfileTime(profile *model.UserProfile) {
	profile.CreatedAt = profile.CreatedAt.UTC()
	profile.UpdatedAt = profile.UpdatedAt.UTC()
	if profile.EmailVerifiedAt != nil {
		verifiedAt := profile.EmailVerifiedAt.UTC()
		profile.EmailVerifiedAt = &verifiedAt
	}
}
