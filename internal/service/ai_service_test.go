package service

import (
	"strings"
	"testing"
	"time"
)

func TestDecodeAIResultStrictRejectsUnknownFields(t *testing.T) {
	payload := `{"intents":["expense"],"title":"Test","financial_data":{"amount":1000,"category":"Food"},"tool_call":"delete_all"}`

	if _, err := decodeAIResultStrict(payload); err == nil {
		t.Fatal("expected unknown field to be rejected")
	}
}

func TestDecodeAIResultStrictRejectsTrailingObjects(t *testing.T) {
	payload := `{"intents":[],"title":""} {"intents":["expense"]}`

	if _, err := decodeAIResultStrict(payload); err == nil {
		t.Fatal("expected trailing json to be rejected")
	}
}

func TestBuildGeminiUserPromptIncludesTimezoneContext(t *testing.T) {
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}

	prompt, err := buildGeminiUserPrompt(
		"ingatkan jam 3 sore lewat 7 menit",
		time.Date(2026, 9, 16, 14, 50, 0, 0, location),
		"Asia/Jakarta",
	)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(prompt, `"current_time":"2026-09-16T14:50:00+07:00"`) {
		t.Fatalf("expected local current_time with +07:00 offset, got %s", prompt)
	}
	if !strings.Contains(prompt, `"current_timezone":"Asia/Jakarta"`) {
		t.Fatalf("expected current_timezone in prompt, got %s", prompt)
	}
}
