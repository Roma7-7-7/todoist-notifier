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

var (
	todoistToken   string
	telegramBotID  string
	telegramChatID string
)

func main() {
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	todoistClient := todoist.NewClient(todoistToken, httpClient, 5, time.Second, log)
	messagePublisher := internal.NewHTTPMessagePublisher(httpClient, telegramBotID)
	handler := internal.NewLambdaHandler(todoistClient, messagePublisher, telegramChatID, log)
	lambda.Start(handler.HandleRequest)
}

func init() {
	if todoistToken = os.Getenv("TODOIST_TOKEN"); todoistToken == "" {
		panic("TODOIST_TOKEN is not set")
	}
	if telegramBotID = os.Getenv("TELEGRAM_BOT_ID"); telegramBotID == "" {
		panic("TELEGRAM_BOT_ID is not set")
	}
	if telegramChatID = os.Getenv("TELEGRAM_CHAT_ID"); telegramChatID == "" {
		panic("TELEGRAM_CHAT_ID is not set")
	}
}
