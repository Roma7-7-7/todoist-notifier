package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Roma7-7-7/telegram"

	"github.com/Roma7-7-7/todoist-notifier/internal"
	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) //nolint:mnd // it's ok
	status := run(ctx)
	cancel()
	os.Exit(status)
}

func run(ctx context.Context) int {
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	conf, err := internal.GetConfig(ctx)
	if err != nil {
		log.Error("failed to get config", "error", err)
		return 1
	}

	todoistClient := todoist.NewClient(conf.TodoistToken, httpClient, 5, time.Second, log)
	messagePublisher := telegram.NewClient(httpClient, conf.TelegramToken)
	handler := internal.NewLambdaHandler(todoistClient, messagePublisher, conf.TelegramChatID, log)
	if !conf.Dev {
		lambda.Start(handler.HandleRequest)
		return 0
	}

	err = handler.HandleRequest(ctx)
	if err != nil {
		log.Error("failed to handle request", "error", err)
		return 1
	}

	return 0
}
