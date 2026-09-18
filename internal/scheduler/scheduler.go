package scheduler

import (
	"context"
	"log/slog"
	"time"

	"smart-journal/internal/service"
)

type Scheduler struct {
	reminders *service.ReminderService
	interval  time.Duration
	limit     int
	logger    *slog.Logger
}

func New(reminders *service.ReminderService, interval time.Duration, logger *slog.Logger) *Scheduler {
	if interval <= 0 {
		interval = time.Minute
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Scheduler{
		reminders: reminders,
		interval:  interval,
		limit:     50,
		logger:    logger,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.alignToNextMinute(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopped")
			return
		case now := <-ticker.C:
			s.sendDue(ctx, now.UTC())
		}
	}
}

func (s *Scheduler) alignToNextMinute(ctx context.Context) {
	now := time.Now()
	wait := now.Truncate(time.Minute).Add(time.Minute).Sub(now)
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func (s *Scheduler) sendDue(ctx context.Context, now time.Time) {
	sent, err := s.reminders.SendDue(ctx, now, s.limit)
	if err != nil {
		s.logger.Error("send due reminders failed", "error", err)
		return
	}
	if sent > 0 {
		s.logger.Info("due reminders sent", "count", sent)
	}
}
