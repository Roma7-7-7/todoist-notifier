package internal

import (
	"context"
	"fmt"

	"github.com/robfig/cron/v3"
)

type Schedule struct {
	schedule string
	log      Logger
}

func NewScheduler(schedule string, log Logger) *Schedule {
	return &Schedule{
		schedule: schedule,
		log:      log,
	}
}

func (s *Schedule) Run(ctx context.Context, f func()) error {
	s.log.InfoContext(ctx, "start scheduler", "schedule", s.schedule)

	scheduler := cron.New()
	_, err := scheduler.AddFunc(s.schedule, f)
	if err != nil {
		return fmt.Errorf("add cron job: %w", err)
	}

	go func() {
		select {
		case <-ctx.Done():
			s.log.InfoContext(ctx, "stop scheduler")
			scheduler.Stop()
		}
	}()

	scheduler.Run()
	return nil
}
