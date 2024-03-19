package internal

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
)

type (
	Logger interface {
		DebugContext(ctx context.Context, msg string, fields ...any)
		InfoContext(ctx context.Context, msg string, fields ...any)
		WarnContext(ctx context.Context, msg string, fields ...any)
		ErrorContext(ctx context.Context, msg string, fields ...any)

		Info(msg string, fields ...any)
	}

	TodoistClient interface {
		GetTasksV2(ctx context.Context, isCompleted bool) ([]todoist.Task, error)
	}

	TodoistConfig struct {
		Token string
	}

	TelegramConfig struct {
		Token  string
		ChatID string
	}

	Config struct {
		Schedule string
		Todoist  TodoistConfig
		Telegram TelegramConfig
	}

	App struct {
		scheduler *Schedule
		job       *Job
		bot       *Bot
		log       Logger
	}
)

func NewApp(cfg Config, log Logger) (*App, error) {
	bot, err := NewBot(cfg.Telegram.Token, log)
	if err != nil {
		return nil, fmt.Errorf("new bot: %w", err)
	}

	chatID, err := strconv.ParseInt(cfg.Telegram.ChatID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse chat id %q: %w", cfg.Telegram.ChatID, err)
	}

	publisher := &MessagePublisher{
		chatID: chatID,
		bot:    bot.bot,
		log:    log,
	}

	return &App{
		scheduler: NewScheduler(cfg.Schedule, log),
		job:       NewJob(todoist.NewClient(cfg.Todoist.Token, http.DefaultClient, log), publisher, log),
		bot:       bot,
		log:       log,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.log.InfoContext(ctx, "start app")

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := a.scheduler.Run(ctx, func() {
			ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
			defer cancel()
			if err := a.job.Run(ctx); err != nil {
				a.log.ErrorContext(ctx, "job run", "err", err)
			}
		})

		if err != nil {
			a.log.ErrorContext(ctx, "scheduler run", "err", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		a.bot.Run(ctx)
	}()

	wg.Wait()
	a.log.InfoContext(ctx, "app stopped")
	return nil
}