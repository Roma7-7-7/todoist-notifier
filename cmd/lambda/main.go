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
	defer cancel()
	os.Exit(run(ctx))
}

func run(ctx context.Context) int {
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	conf, err := internal.GetConfig(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get config", "error", err)
		return 1
	}

	log := internal.NewLogger(conf.Dev)

	todoistClient := todoist.NewClient(conf.TodoistToken, httpClient, 5, time.Second, log)
	messagePublisher := telegram.NewClient(httpClient, conf.TelegramToken)
	handler, err := internal.NewLambdaHandler(conf, todoistClient, messagePublisher, log)
	if err != nil {
		log.ErrorContext(ctx, "failed to create handler", "error", err)
		return 1
	}
	if !conf.Dev {
		lambda.Start(handler.HandleRequest)
		return 0
	}

	err = handler.HandleRequest(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to handle request", "error", err)
		return 1
	}

	return 0
}
