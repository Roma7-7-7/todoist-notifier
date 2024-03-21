package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"

	"github.com/Roma7-7-7/todoist-notifier/internal"
)

var schedule = flag.String("schedule", "0 8-23 * * *", "Cron schedule")
var todoistToken = flag.String("todoist-token", "", "Todoist API token")
var telegramBotID = flag.String("telegram-bot-id", "", "Telegram bot ID")
var telegramChatID = flag.String("telegram-chat-id", "", "Telegram chat ID")

func main() {
	flag.Parse()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if *todoistToken == "" {
		logger.Error("todoist token is not set")
		os.Exit(1)
	}

	if *telegramBotID == "" {
		logger.Error("telegram bot ID is not set")
		os.Exit(1)
	}

	if *telegramChatID == "" {
		logger.Error("telegram chat ID is not set")
		os.Exit(1)
	}

	if *schedule == "" {
		logger.Error("cron schedule is not set")
		os.Exit(1)
	}

	cfg := internal.Config{
		Schedule: *schedule,
		Todoist: internal.TodoistConfig{
			Token: *todoistToken,
		},
		Telegram: internal.TelegramConfig{
			Token:  *telegramBotID,
			ChatID: *telegramChatID,
		},
	}
	app, err := internal.NewApp(cfg, logger)
	if err != nil {
		logger.Error("new app", "error", err)
		os.Exit(2)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err = app.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("run app", "error", err)
		os.Exit(3)
	}
	logger.Info("app stopped")
}
