package service

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"smart-journal/internal/model"
)

type ReminderService struct {
	profiles            ProfileReader
	reminders           ReminderRepository
	emailSender         EmailSender
	emailFrom           string
	notificationToEmail string
	now                 func() time.Time
}

func NewReminderService(
	profiles ProfileReader,
	reminders ReminderRepository,
	emailSender EmailSender,
	emailFrom string,
	notificationToEmail string,
) *ReminderService {
	return &ReminderService{
		profiles:            profiles,
		reminders:           reminders,
		emailSender:         emailSender,
		emailFrom:           emailFrom,
		notificationToEmail: notificationToEmail,
		now:                 time.Now,
	}
}

type CreateReminderInput struct {
	UserID   string
	NoteID   string
	RemindAt time.Time
	Message  string
	Status   string
}

type UpdateReminderInput struct {
	UserID     string
	ReminderID string
	NoteID     *string
	RemindAt   *time.Time
	Message    *string
	Status     *string
}

func (s *ReminderService) Create(ctx context.Context, input CreateReminderInput) (model.Reminder, error) {
	if input.UserID == "" {
		return model.Reminder{}, errors.New("user id is required")
	}
	if input.RemindAt.IsZero() {
		return model.Reminder{}, errors.New("remind_at is required")
	}

	message := strings.TrimSpace(input.Message)
	if message == "" {
		return model.Reminder{}, errors.New("message is required")
	}

	status, err := normalizeReminderStatus(input.Status)
	if err != nil {
		return model.Reminder{}, err
	}

	now := s.now().UTC()
	return s.reminders.Create(ctx, input.UserID, model.Reminder{
		NoteID:    strings.TrimSpace(input.NoteID),
		RemindAt:  input.RemindAt.UTC(),
		Message:   message,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	})
}

func (s *ReminderService) List(ctx context.Context, userID string, limit int) ([]model.Reminder, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	return s.reminders.List(ctx, userID, limit)
}

func (s *ReminderService) Get(ctx context.Context, userID string, reminderID string) (model.Reminder, error) {
	if userID == "" {
		return model.Reminder{}, errors.New("user id is required")
	}
	if reminderID == "" {
		return model.Reminder{}, errors.New("reminder id is required")
	}
	return s.reminders.Get(ctx, userID, reminderID)
}

func (s *ReminderService) Update(ctx context.Context, input UpdateReminderInput) (model.Reminder, error) {
	if input.UserID == "" {
		return model.Reminder{}, errors.New("user id is required")
	}
	if input.ReminderID == "" {
		return model.Reminder{}, errors.New("reminder id is required")
	}

	updates := map[string]any{
		"updated_at": s.now().UTC(),
	}
	if input.NoteID != nil {
		updates["note_id"] = strings.TrimSpace(*input.NoteID)
	}
	if input.RemindAt != nil {
		if input.RemindAt.IsZero() {
			return model.Reminder{}, errors.New("remind_at cannot be empty")
		}
		updates["remind_at"] = input.RemindAt.UTC()
	}
	if input.Message != nil {
		message := strings.TrimSpace(*input.Message)
		if message == "" {
			return model.Reminder{}, errors.New("message cannot be empty")
		}
		updates["message"] = message
	}
	if input.Status != nil {
		status, err := normalizeReminderStatus(*input.Status)
		if err != nil {
			return model.Reminder{}, err
		}
		updates["status"] = status
		if status == model.ReminderStatusPending {
			updates["sent_at"] = nil
		}
	}
	if len(updates) == 1 {
		return s.Get(ctx, input.UserID, input.ReminderID)
	}

	return s.reminders.Update(ctx, input.UserID, input.ReminderID, updates)
}

func (s *ReminderService) Delete(ctx context.Context, userID string, reminderID string) error {
	if userID == "" {
		return errors.New("user id is required")
	}
	if reminderID == "" {
		return errors.New("reminder id is required")
	}
	return s.reminders.Delete(ctx, userID, reminderID)
}

func (s *ReminderService) SendDue(ctx context.Context, now time.Time, limit int) (int, error) {
	dueReminders, err := s.reminders.Due(ctx, now.UTC(), limit)
	if err != nil {
		return 0, err
	}

	sent := 0
	var errs []error
	for _, reminder := range dueReminders {
		if err := s.Send(ctx, reminder); err != nil {
			errs = append(errs, err)
			continue
		}
		sent++
	}

	return sent, errors.Join(errs...)
}

func (s *ReminderService) SendDueForUser(ctx context.Context, userID string, now time.Time, limit int) (int, error) {
	if userID == "" {
		return 0, errors.New("user id is required")
	}

	dueReminders, err := s.reminders.DueForUser(ctx, userID, now.UTC(), limit)
	if err != nil {
		return 0, err
	}

	sent := 0
	var errs []error
	for _, reminder := range dueReminders {
		if err := s.Send(ctx, reminder); err != nil {
			errs = append(errs, err)
			continue
		}
		sent++
	}

	return sent, errors.Join(errs...)
}

func (s *ReminderService) Send(ctx context.Context, reminder model.Reminder) error {
	if reminder.ID == "" {
		return errors.New("reminder id is required")
	}
	if reminder.UserID == "" {
		return errors.New("reminder user id is required")
	}
	if s.emailSender == nil {
		return errors.New("email sender is not configured")
	}

	profile, err := s.profiles.Get(ctx, reminder.UserID)
	if err != nil {
		return err
	}
	toEmail := s.notificationRecipient(profile)
	if toEmail == "" {
		return fmt.Errorf("notification email is not configured for user %s", profile.UserID)
	}

	message := EmailMessage{
		From:           s.emailFrom,
		To:             toEmail,
		Subject:        "Smart Journal Reminder",
		HTML:           reminderEmailHTML(reminder),
		Text:           reminder.Message,
		IdempotencyKey: "reminder/" + reminder.UserID + "/" + reminder.ID,
	}
	if _, err := s.emailSender.Send(ctx, message); err != nil {
		return fmt.Errorf("send reminder email: %w", err)
	}

	return s.reminders.MarkSent(ctx, reminder.UserID, reminder.ID, s.now().UTC())
}

func normalizeReminderStatus(value string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", model.ReminderStatusPending:
		return model.ReminderStatusPending, nil
	case model.ReminderStatusSent:
		return model.ReminderStatusSent, nil
	default:
		return "", errors.New("status must be pending or sent")
	}
}

func (s *ReminderService) notificationRecipient(profile model.UserProfile) string {
	if strings.TrimSpace(s.notificationToEmail) != "" {
		return strings.TrimSpace(s.notificationToEmail)
	}
	return strings.TrimSpace(profile.Email)
}

func reminderEmailHTML(reminder model.Reminder) string {
	return fmt.Sprintf(
		`<h1>Smart Journal Reminder</h1><p>%s</p><p><small>Reminder time: %s</small></p>`,
		html.EscapeString(reminder.Message),
		html.EscapeString(reminder.RemindAt.Format(time.RFC3339)),
	)
}
