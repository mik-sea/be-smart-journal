package service

import (
	"strings"
	"testing"
	"time"

	"smart-journal/internal/model"
)

func TestReminderEmailHTMLEscapesMessage(t *testing.T) {
	reminder := model.Reminder{
		Message:  `<script>alert("x")</script>`,
		RemindAt: time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC),
	}

	html := reminderEmailHTML(reminder)
	if strings.Contains(html, "<script>") {
		t.Fatalf("expected message to be escaped, got %q", html)
	}
}
