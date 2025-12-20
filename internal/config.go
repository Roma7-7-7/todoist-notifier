package internal

import (
	"context"
	"fmt"
	"os"

	pkgSSM "github.com/Roma7-7-7/todoist-notifier/pkg/ssm"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type Config struct {
	Dev            bool
	TodoistToken   string
	TelegramToken  string
	TelegramChatID string
	Schedule       string
	Location       string
}

func GetConfig(ctx context.Context) (*Config, error) {
	res := &Config{
		Dev:            os.Getenv("ENV") == "dev",
		TodoistToken:   os.Getenv("TODOIST_TOKEN"),
		TelegramToken:  os.Getenv("TELEGRAM_BOT_ID"),
		TelegramChatID: os.Getenv("TELEGRAM_CHAT_ID"),
		Schedule:       os.Getenv("SCHEDULE"),
		Location:       os.Getenv("LOCATION"),
	}
	if res.Schedule == "" {
		res.Schedule = "0 * 9-23 * * *"
	}
	if res.Location == "" {
		res.Location = "Europe/Kyiv"
	}
	if res.Dev && os.Getenv("FORCE_SSM") != "true" {
		if err := res.validate(); err != nil {
			return nil, err
		}
		return res, nil
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	ssmClient := ssm.NewFromConfig(cfg)

	err = pkgSSM.FetchParameters(ctx, ssmClient, map[string]*string{
		"/todoist-notifier-bot/prod/todoist-token":    &res.TodoistToken,
		"/todoist-notifier-bot/prod/telegram-token":   &res.TelegramToken,
		"/todoist-notifier-bot/prod/telegram-chat-id": &res.TelegramChatID,
	}, pkgSSM.WithDecryption())
	if err != nil {
		return nil, fmt.Errorf("fetch SSM parameters: %w", err)
	}

	if err := res.validate(); err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Config) validate() error {
	var missing []string

	if c.TodoistToken == "" {
		missing = append(missing, "TODOIST_TOKEN")
	}
	if c.TelegramToken == "" {
		missing = append(missing, "TELEGRAM_BOT_ID")
	}
	if c.TelegramChatID == "" {
		missing = append(missing, "TELEGRAM_CHAT_ID")
	}

	if len(missing) > 0 {
		return fmt.Errorf("required environment variables not set: %v", missing)
	}

	return nil
}
