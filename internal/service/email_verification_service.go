package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	firebaseauth "firebase.google.com/go/v4/auth"

	"smart-journal/internal/model"
)

type EmailVerificationRepository interface {
	Create(ctx context.Context, verification model.EmailVerification) (model.EmailVerification, error)
	Get(ctx context.Context, tokenHash string) (model.EmailVerification, error)
	MarkUsed(ctx context.Context, tokenHash string, usedAt time.Time) error
}

type EmailVerificationService struct {
	authClient     *firebaseauth.Client
	profiles       ProfileRepository
	verifications  EmailVerificationRepository
	emailSender    EmailSender
	emailFrom      string
	baseURL        string
	tokenDuration  time.Duration
	now            func() time.Time
}

func NewEmailVerificationService(
	authClient *firebaseauth.Client,
	profiles ProfileRepository,
	verifications EmailVerificationRepository,
	emailSender EmailSender,
	emailFrom string,
	baseURL string,
	tokenDuration time.Duration,
) *EmailVerificationService {
	if tokenDuration <= 0 {
		tokenDuration = 24 * time.Hour
	}

	return &EmailVerificationService{
		authClient:    authClient,
		profiles:      profiles,
		verifications: verifications,
		emailSender:   emailSender,
		emailFrom:     emailFrom,
		baseURL:       strings.TrimRight(baseURL, "/"),
		tokenDuration: tokenDuration,
		now:           time.Now,
	}
}

func (s *EmailVerificationService) Send(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("user id is required")
	}
	if s.emailSender == nil {
		return errors.New("email sender is not configured")
	}

	profile, err := s.profiles.Get(ctx, userID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(profile.Email) == "" {
		return errors.New("profile email is required")
	}

	rawToken, err := randomVerificationToken()
	if err != nil {
		return fmt.Errorf("create verification token: %w", err)
	}

	now := s.now().UTC()
	tokenHash := verificationTokenHash(rawToken)
	verification := model.EmailVerification{
		TokenHash: tokenHash,
		UserID:    userID,
		Email:     profile.Email,
		Status:    model.EmailVerificationStatusPending,
		ExpiresAt: now.Add(s.tokenDuration),
		CreatedAt: now,
	}
	if _, err := s.verifications.Create(ctx, verification); err != nil {
		return err
	}

	link := s.verificationLink(rawToken)
	_, err = s.emailSender.Send(ctx, EmailMessage{
		From:           s.emailFrom,
		To:             profile.Email,
		Subject:        "Verify your Smart Journal email",
		HTML:           verificationEmailHTML(link),
		Text:           "Verify your Smart Journal email: " + link,
		IdempotencyKey: "email-verification/" + tokenHash,
	})
	return err
}

func (s *EmailVerificationService) Verify(ctx context.Context, rawToken string) (model.UserProfile, error) {
	if strings.TrimSpace(rawToken) == "" {
		return model.UserProfile{}, errors.New("verification token is required")
	}

	now := s.now().UTC()
	tokenHash := verificationTokenHash(rawToken)
	verification, err := s.verifications.Get(ctx, tokenHash)
	if err != nil {
		return model.UserProfile{}, err
	}
	if verification.Status != model.EmailVerificationStatusPending {
		return model.UserProfile{}, errors.New("verification token already used")
	}
	if now.After(verification.ExpiresAt) {
		return model.UserProfile{}, errors.New("verification token expired")
	}

	currentProfile, err := s.profiles.Get(ctx, verification.UserID)
	if err != nil {
		return model.UserProfile{}, err
	}
	if !sameEmail(currentProfile.Email, verification.Email) {
		return model.UserProfile{}, errors.New("verification email no longer matches profile email")
	}

	if s.authClient != nil {
		params := (&firebaseauth.UserToUpdate{}).EmailVerified(true)
		if _, err := s.authClient.UpdateUser(ctx, verification.UserID, params); err != nil {
			return model.UserProfile{}, fmt.Errorf("update firebase email verification: %w", err)
		}
	}

	profile, err := s.profiles.MarkEmailVerified(ctx, verification.UserID, now)
	if err != nil {
		return model.UserProfile{}, err
	}
	if err := s.verifications.MarkUsed(ctx, tokenHash, now); err != nil {
		return model.UserProfile{}, err
	}

	return profile, nil
}

func (s *EmailVerificationService) verificationLink(token string) string {
	values := url.Values{}
	values.Set("token", token)
	return s.baseURL + "/api/auth/email-verification?" + values.Encode()
}

func randomVerificationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func verificationTokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func sameEmail(left string, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

func verificationEmailHTML(link string) string {
	escapedLink := html.EscapeString(link)
	return fmt.Sprintf(
		`<h1>Verify your Smart Journal email</h1><p>Click this link to verify your email address:</p><p><a href="%s">Verify email</a></p><p>If you did not request this email, you can ignore it.</p>`,
		escapedLink,
	)
}
