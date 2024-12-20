package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Roma7-7-7/todoist-notifier/internal"
	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	conf, err := internal.GetConfig()
	if err != nil {
		log.Error("failed to get config", "error", err)
		os.Exit(1)
		return
	}

	todoistClient := todoist.NewClient(conf.TodoistToken, httpClient, 5, time.Second, log)
	messagePublisher := internal.NewHTTPMessagePublisher(httpClient, conf.TelegramToken)
	handler := internal.NewLambdaHandler(todoistClient, messagePublisher, conf.TelegramChatID, log)
	lambda.Start(handler.HandleRequest)
}
