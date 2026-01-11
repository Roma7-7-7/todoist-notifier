package internal

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	pkgSSM "github.com/Roma7-7-7/todoist-notifier/pkg/ssm"
)

type Config struct {
	Dev            bool
	TodoistToken   string
	TelegramToken  string
	TelegramChatID int64
	Schedule       string
	Location       string
}

func GetConfig(ctx context.Context) (*Config, error) {
	res := &Config{
		Dev:           os.Getenv("ENV") == "dev",
		TodoistToken:  os.Getenv("TODOIST_TOKEN"),
		TelegramToken: os.Getenv("TELEGRAM_BOT_ID"),
		Schedule:      os.Getenv("SCHEDULE"),
		Location:      os.Getenv("LOCATION"),
	}
	telegramChatID := os.Getenv("TELEGRAM_CHAT_ID")
	if res.Schedule == "" {
		res.Schedule = "0 * 9-23 * * *"
	}
	if res.Location == "" {
		res.Location = "Europe/Kyiv"
	}

	// In dev mode or if all required params are set via env vars, skip SSM
	if res.Dev || hasRequiredParams(res, telegramChatID) {
		if err := res.validate(telegramChatID); err != nil {
			return nil, err
		}
		return res, nil
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config (set required env vars to skip SSM): %w", err)
	}
	ssmClient := ssm.NewFromConfig(cfg)

	err = pkgSSM.FetchParameters(ctx, ssmClient, map[string]*string{
		"/todoist-notifier-bot/prod/todoist-token":    &res.TodoistToken,
		"/todoist-notifier-bot/prod/telegram-token":   &res.TelegramToken,
		"/todoist-notifier-bot/prod/telegram-chat-id": &telegramChatID,
	}, pkgSSM.WithDecryption())
	if err != nil {
		return nil, fmt.Errorf("fetch SSM parameters (set required env vars to skip SSM): %w", err)
	}

	if err := res.validate(telegramChatID); err != nil {
		return nil, err
	}

	return res, nil
}

// hasRequiredParams checks if all required parameters are already set via environment variables
func hasRequiredParams(conf *Config, telegramChatID string) bool {
	return conf.TodoistToken != "" && conf.TelegramToken != "" && telegramChatID != ""
}

func (c *Config) validate(telegramChatID string) error {
	var missing []string

	if c.TodoistToken == "" {
		missing = append(missing, "TODOIST_TOKEN")
	}
	if c.TelegramToken == "" {
		missing = append(missing, "TELEGRAM_BOT_ID")
	}
	if telegramChatID == "" {
		missing = append(missing, "TELEGRAM_CHAT_ID")
	}

	if len(missing) > 0 {
		return fmt.Errorf("required environment variables not set: %v", missing)
	}

	var err error
	c.TelegramChatID, err = strconv.ParseInt(telegramChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("parse TELEGRAM_CHAT_ID: %w", err)
	}

	return nil
}
