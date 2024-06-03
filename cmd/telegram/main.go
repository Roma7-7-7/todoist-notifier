package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/Roma7-7-7/todoist-notifier/internal"
)

var cfg internal.Config

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

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

func init() {
	var (
		schedule       string
		todoistToken   string
		telegramBotID  string
		telegramChatID string
	)
	if schedule = os.Getenv("SCHEDULE"); schedule == "" {
		schedule = "0 9-23 * * *"
	}
	if todoistToken = os.Getenv("TODOIST_TOKEN"); todoistToken == "" {
		panic("TODOIST_TOKEN is not set")
	}
	if telegramBotID = os.Getenv("TELEGRAM_BOT_ID"); telegramBotID == "" {
		panic("TELEGRAM_BOT_ID is not set")
	}
	if telegramChatID = os.Getenv("TELEGRAM_CHAT_ID"); telegramChatID == "" {
		panic("TELEGRAM_CHAT_ID is not set")
	}

	cfg = internal.Config{
		Schedule: schedule,
		Todoist: internal.TodoistConfig{
			Token: todoistToken,
		},
		Telegram: internal.TelegramConfig{
			Token:  telegramBotID,
			ChatID: telegramChatID,
		},
	}
}
