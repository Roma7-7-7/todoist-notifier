package internal

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

const (
	defaultTimeout  = 10 * time.Second
	defaultErrorMsg = "Something went wrong. Please try again later."
)

type TaskFormatter interface {
	GetFormattedTasks(ctx context.Context) (string, error)
}

type Bot struct {
	bot *tele.Bot

	todoistClient TodoistClient
	clock         Clock

	allowedChatID int64

	log *slog.Logger
}

func NewBot(token string, todoistClient TodoistClient, allowedChatID int64, clock Clock, log *slog.Logger) (*Bot, error) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}

	bot := &Bot{
		bot:           b,
		todoistClient: todoistClient,
		allowedChatID: allowedChatID,
		clock:         clock,
		log:           log,
	}

	bot.registerHandlers()

	return bot, nil
}

func (b *Bot) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		b.log.InfoContext(ctx, "stopping bot")
		b.bot.Stop()
	}()

	b.log.InfoContext(ctx, "bot started")
	b.bot.Start()

	return nil
}

func (b *Bot) registerHandlers() {
	b.bot.Use(b.recover, b.handleError, b.chatIDMiddleware)
	b.bot.Handle("/tasks", b.handleTasks)
}

func (b *Bot) handleTasks(c tele.Context) error {
	return b.SendTasks(c.Chat().ID, false)
}

func (b *Bot) SendTasks(chatID int64, ignoreNoTasks bool) error {
	ctx, cancel := b.context()
	defer cancel()

	b.log.DebugContext(ctx, "received /tasks command", "chat_id", chatID)

	tasks, err := b.todoistClient.GetTasks(ctx, false)
	if err != nil {
		return fmt.Errorf("get tasks: %w", err)
	}

	tasks = FilterAndSortTasks(tasks, b.clock.Now(), false)

	var msg string
	switch {
	case len(tasks) != 0:
		msg, err = RenderTasksMessage(tasks)
		if err != nil {
			return fmt.Errorf("render tasks message: %w", err)
		}
	case len(tasks) == 0 && !ignoreNoTasks:
		msg = "No tasks for today! 🎉"
	case len(tasks) == 0 && ignoreNoTasks:
		b.log.DebugContext(ctx, "no tasks to send")
		return nil
	}

	if _, err := b.bot.Send(&tele.Chat{ID: chatID}, msg); err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	b.log.DebugContext(ctx, "tasks sent successfully")
	return nil
}

func (b *Bot) context() (context.Context, func()) {
	return context.WithTimeout(context.Background(), defaultTimeout)
}

func (b *Bot) recover(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		defer func() {
			if r := recover(); r != nil {
				b.log.ErrorContext(context.Background(), "panic recovered", "error", r)
			}
		}()

		return next(c)
	}
}

func (b *Bot) chatIDMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if c.Chat().ID != b.allowedChatID {
			b.log.WarnContext(context.Background(), "unauthorized chat access blocked",
				"chat_id", c.Chat().ID,
				"allowed_chat_id", b.allowedChatID,
			)
			return c.Send("Unauthorized")
		}
		return next(c)
	}
}

func (b *Bot) handleError(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		err := next(c)
		if err != nil {
			args := []any{
				"chat_id", c.Chat().ID,
				"error", err.Error(),
			}
			if strings.HasPrefix(c.Message().Text, "/") {
				args = append(args, "command", strings.Split(c.Message().Text, " ")[0])
			}
			b.log.ErrorContext(context.Background(), "error occurred", args...)
			return c.Send(defaultErrorMsg)
		}
		return err
	}
}
