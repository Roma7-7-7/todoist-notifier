package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Roma7-7-7/telegram"
	"github.com/Roma7-7-7/todoist-notifier/internal"
	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
	"github.com/go-co-op/gocron/v2"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	exitCode := run(ctx)
	cancel()
	os.Exit(exitCode)
}

func run(ctx context.Context) int {
	conf, err := internal.GetConfig(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get config", "error", err)
		return 1
	}

	log := internal.NewLogger(conf.Dev)

	httpClient := &http.Client{
		Timeout: 5 * time.Second, //nolint:mnd // reasonable timeout
	}

	todoistClient := todoist.NewClient(conf.TodoistToken, httpClient, 5, time.Second, log) //nolint:mnd // reasonable retry config
	messagePublisher := telegram.NewClient(httpClient, conf.TelegramToken)

	notifier, err := internal.NewNotifier(conf, todoistClient, messagePublisher, log)
	if err != nil {
		log.ErrorContext(ctx, "failed to create notifier", "error", err)
		return 1
	}

	loc, err := time.LoadLocation(conf.Location)
	if err != nil {
		log.ErrorContext(ctx, "failed to load timezone", "error", err, "location", conf.Location)
		return 1
	}

	scheduler, err := gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		log.ErrorContext(ctx, "failed to create scheduler", "error", err)
		return 1
	}

	job, err := scheduler.NewJob(
		gocron.CronJob(conf.Schedule, false),
		gocron.NewTask(func() {
			if err := notifier.SendNotification(ctx); err != nil {
				log.ErrorContext(ctx, "failed to send notification", "error", err)
			}
		}),
	)
	if err != nil {
		log.ErrorContext(ctx, "failed to create job", "error", err, "schedule", conf.Schedule)
		return 1
	}

	scheduler.Start()

	nextRun, err := job.NextRun()
	if err != nil {
		log.Warn("failed to get next run time", "error", err)
		log.Info("starting daemon notifier", "schedule", conf.Schedule, "timezone", conf.Location)
	} else {
		log.Info("starting daemon notifier", "schedule", conf.Schedule, "timezone", conf.Location, "next_run", nextRun)
	}
	defer func() {
		if err := scheduler.Shutdown(); err != nil {
			log.ErrorContext(ctx, "failed to shutdown scheduler", "error", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Info("received shutdown signal", "signal", sig)
		return 0
	case <-ctx.Done():
		log.Info("context cancelled")
		return 0
	}
}
