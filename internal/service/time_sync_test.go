package service

import (
	"testing"
	"time"
)

func TestNormalizeAIReminderTimeTreatsUnexpectedUTCAsLocal(t *testing.T) {
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}

	actual := normalizeAIReminderTime(
		time.Date(2026, 9, 16, 15, 7, 0, 0, time.UTC),
		location,
		"ingatkan jam 3 sore lewat 7 menit",
	)
	expected := time.Date(2026, 9, 16, 8, 7, 0, 0, time.UTC)

	if !actual.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected.Format(time.RFC3339), actual.Format(time.RFC3339))
	}
}

func TestNormalizeAIReminderTimeKeepsExplicitUTC(t *testing.T) {
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}

	actual := normalizeAIReminderTime(
		time.Date(2026, 9, 16, 15, 7, 0, 0, time.UTC),
		location,
		"ingatkan jam 15:07 UTC",
	)
	expected := time.Date(2026, 9, 16, 15, 7, 0, 0, time.UTC)

	if !actual.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected.Format(time.RFC3339), actual.Format(time.RFC3339))
	}
}

func TestNormalizeTimezoneRejectsInvalidTimezone(t *testing.T) {
	if _, err := normalizeTimezone("Jakarta"); err == nil {
		t.Fatal("expected invalid timezone to be rejected")
	}
}
