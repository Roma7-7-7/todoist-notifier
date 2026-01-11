package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Roma7-7-7/todoist-notifier/internal"
	"github.com/Roma7-7-7/todoist-notifier/pkg/clock"
	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
	"github.com/go-co-op/gocron/v2"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
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
		slog.ErrorContext(ctx, "failed to get config", "error", err) //nolint:sloglint // logger is not yet initialized
		return 1
	}

	log := internal.NewLogger(conf.Dev)
	log.InfoContext(ctx, "todoist-notifier daemon starting", "version", Version, "build_time", BuildTime)

	httpClient := &http.Client{
		Timeout: 5 * time.Second, //nolint:mnd // reasonable timeout
	}

	todoistClient := todoist.NewClient(conf.TodoistToken, httpClient, 5, time.Second, log) //nolint:mnd // reasonable retry config

	loc, err := time.LoadLocation(conf.Location)
	if err != nil {
		log.ErrorContext(ctx, "failed to load timezone", "error", err, "location", conf.Location)
		return 1
	}
	clock := clock.NewZonedClock(loc)

	bot, err := internal.NewBot(conf.TelegramToken, todoistClient, conf.TelegramChatID, clock, log)
	if err != nil {
		log.ErrorContext(ctx, "failed to create bot", "error", err)
		return 1
	}

	botErrChan := make(chan error, 1)
	go func() {
		if err := bot.Start(ctx); err != nil {
			botErrChan <- err
		}
	}()

	select {
	case err := <-botErrChan:
		log.ErrorContext(ctx, "bot failed to start", "error", err)
		return 1
	case <-time.After(2 * time.Second): //nolint:mnd // reasonable startup timeout
		log.InfoContext(ctx, "bot started successfully")
	}

	scheduler, err := gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		log.ErrorContext(ctx, "failed to create scheduler", "error", err)
		return 1
	}

	job, err := scheduler.NewJob(
		gocron.CronJob(conf.Schedule, false),
		gocron.NewTask(func() {
			if err := bot.SendTasks(conf.TelegramChatID, true); err != nil {
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
		log.WarnContext(ctx, "failed to get next run time", "error", err)
		log.InfoContext(ctx, "starting daemon", "schedule", conf.Schedule, "timezone", conf.Location)
	} else {
		log.InfoContext(ctx, "starting daemon", "schedule", conf.Schedule, "timezone", conf.Location, "next_run", nextRun)
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
		log.InfoContext(ctx, "received shutdown signal", "signal", sig)
		return 0
	case <-ctx.Done():
		log.InfoContext(ctx, "context cancelled")
		return 0
	}
}
