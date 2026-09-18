package service

import (
	"fmt"
	"strings"
	"time"
)

const DefaultUserTimezone = "Asia/Jakarta"

func normalizeTimezone(value string) (string, error) {
	timezone := strings.TrimSpace(value)
	if timezone == "" {
		return DefaultUserTimezone, nil
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return "", fmt.Errorf("timezone %q is not supported", timezone)
	}
	return timezone, nil
}

func defaultUserLocation() *time.Location {
	location, err := time.LoadLocation(DefaultUserTimezone)
	if err == nil {
		return location
	}
	return time.FixedZone(DefaultUserTimezone, 7*60*60)
}

func normalizeAIReminderTime(remindAt time.Time, userLocation *time.Location, rawText string) time.Time {
	if shouldTreatUTCAsLocalAIOutput(remindAt, userLocation, rawText) {
		utcClock := remindAt.UTC()
		return time.Date(
			utcClock.Year(),
			utcClock.Month(),
			utcClock.Day(),
			utcClock.Hour(),
			utcClock.Minute(),
			utcClock.Second(),
			utcClock.Nanosecond(),
			userLocation,
		).UTC()
	}
	return remindAt.UTC()
}

func shouldTreatUTCAsLocalAIOutput(remindAt time.Time, userLocation *time.Location, rawText string) bool {
	if userLocation == nil || userMentionedUTC(rawText) {
		return false
	}

	_, reminderOffset := remindAt.Zone()
	if reminderOffset != 0 {
		return false
	}

	_, userOffset := remindAt.In(userLocation).Zone()
	return userOffset != 0
}

func userMentionedUTC(value string) bool {
	normalized := strings.ToLower(value)
	return strings.Contains(normalized, "utc") ||
		strings.Contains(normalized, "gmt") ||
		strings.Contains(normalized, "zulu")
}
