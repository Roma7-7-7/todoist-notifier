package internal

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

const defaultAWSRegion = "eu-central-1"

type Config struct {
	Dev            bool
	TodoistToken   string
	TelegramToken  string
	TelegramChatID string
}

func GetConfig(ctx context.Context) (*Config, error) {
	if os.Getenv("ENV") == "dev" {
		return &Config{
			Dev:            true,
			TodoistToken:   os.Getenv("TODOIST_TOKEN"),
			TelegramToken:  os.Getenv("TELEGRAM_BOT_ID"),
			TelegramChatID: os.Getenv("TELEGRAM_CHAT_ID"),
		}, nil
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = defaultAWSRegion
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	ssmClient := ssm.NewFromConfig(cfg)
	parameters, err := ssmClient.GetParameters(ctx, &ssm.GetParametersInput{
		Names: []string{
			"/todoist-notifier-bot/prod/todoist-token",
			"/todoist-notifier-bot/prod/telegram-token",
			"/todoist-notifier-bot/prod/telegram-chat-id",
		},
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("get parameters: %w", err)
	}

	todoistToken := ""
	telegramToken := ""
	telegramChatID := ""
	for _, param := range parameters.Parameters {
		if param.Name == nil || param.Value == nil {
			continue
		}
		switch *param.Name {
		case "/todoist-notifier-bot/prod/todoist-token":
			todoistToken = *param.Value
		case "/todoist-notifier-bot/prod/telegram-token":
			telegramToken = *param.Value
		case "/todoist-notifier-bot/prod/telegram-chat-id":
			telegramChatID = *param.Value
		}
	}

	errs := make([]string, 0, 3)
	if todoistToken == "" {
		errs = append(errs, "missing todoist token")
	}
	if telegramToken == "" {
		errs = append(errs, "missing telegram token")
	}
	if telegramChatID == "" {
		errs = append(errs, "missing telegram chat id")
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("missing required parameters: %s", errs)
	}

	return &Config{
		TodoistToken:   todoistToken,
		TelegramToken:  telegramToken,
		TelegramChatID: telegramChatID,
	}, nil
}
