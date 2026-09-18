package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"smart-journal/internal/model"
)

type ProfileReader interface {
	Get(ctx context.Context, userID string) (model.UserProfile, error)
}

type ProfileRepository interface {
	ProfileReader
	Create(ctx context.Context, userID string, profile model.UserProfile) (model.UserProfile, error)
	Update(ctx context.Context, userID string, updates map[string]any) (model.UserProfile, error)
	MarkEmailVerified(ctx context.Context, userID string, verifiedAt time.Time) (model.UserProfile, error)
}

type ProfileService struct {
	profiles ProfileRepository
	now      func() time.Time
}

type CreateProfileInput struct {
	UserID    string
	Email     string
	Name      string
	AvatarURL string
	Bio       string
	Timezone  string
}

type UpdateProfileInput struct {
	UserID    string
	Email     *string
	Name      *string
	AvatarURL *string
	Bio       *string
	Timezone  *string
}

func NewProfileService(profiles ProfileRepository) *ProfileService {
	return &ProfileService{
		profiles: profiles,
		now:      time.Now,
	}
}

func (s *ProfileService) Ensure(ctx context.Context, input CreateProfileInput) (model.UserProfile, error) {
	if input.UserID == "" {
		return model.UserProfile{}, errors.New("user id is required")
	}

	existing, err := s.profiles.Get(ctx, input.UserID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, model.ErrNotFound) {
		return model.UserProfile{}, err
	}

	timezone, err := normalizeTimezone(input.Timezone)
	if err != nil {
		return model.UserProfile{}, err
	}

	now := s.now().UTC()
	return s.profiles.Create(ctx, input.UserID, model.UserProfile{
		Email:             strings.TrimSpace(input.Email),
		Name:              strings.TrimSpace(input.Name),
		AvatarURL:         strings.TrimSpace(input.AvatarURL),
		Bio:               strings.TrimSpace(input.Bio),
		Timezone:          timezone,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
}

func (s *ProfileService) Get(ctx context.Context, userID string) (model.UserProfile, error) {
	if userID == "" {
		return model.UserProfile{}, errors.New("user id is required")
	}
	return s.profiles.Get(ctx, userID)
}

func (s *ProfileService) Update(ctx context.Context, input UpdateProfileInput) (model.UserProfile, error) {
	if input.UserID == "" {
		return model.UserProfile{}, errors.New("user id is required")
	}

	updates := map[string]any{
		"updated_at": s.now().UTC(),
	}
	if input.Email != nil {
		updates["email"] = strings.TrimSpace(*input.Email)
		updates["email_verified_at"] = nil
	}
	if input.Name != nil {
		updates["name"] = strings.TrimSpace(*input.Name)
	}
	if input.AvatarURL != nil {
		updates["avatar_url"] = strings.TrimSpace(*input.AvatarURL)
	}
	if input.Bio != nil {
		updates["bio"] = strings.TrimSpace(*input.Bio)
	}
	if input.Timezone != nil {
		timezone, err := normalizeTimezone(*input.Timezone)
		if err != nil {
			return model.UserProfile{}, err
		}
		updates["timezone"] = timezone
	}
	if len(updates) == 1 {
		return s.Get(ctx, input.UserID)
	}

	profile, err := s.profiles.Update(ctx, input.UserID, updates)
	if errors.Is(err, model.ErrNotFound) {
		return s.Ensure(ctx, CreateProfileInput{
			UserID:            input.UserID,
			Email:             stringValue(input.Email),
			Name:              stringValue(input.Name),
			AvatarURL:         stringValue(input.AvatarURL),
			Bio:               stringValue(input.Bio),
			Timezone:          stringValue(input.Timezone),
		})
	}
	return profile, err
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
