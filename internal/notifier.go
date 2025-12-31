package internal

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

type (
	HTTPMessagePublisher interface {
		SendMessage(ctx context.Context, chatID, message string) error
	}

	Notifier struct {
		todoistClient TodoistClient
		msgPublisher  HTTPMessagePublisher
		chatID        string

		now func() time.Time
		log *slog.Logger
	}
)

func NewNotifier(conf *Config, todoistClient TodoistClient, msgPublisher HTTPMessagePublisher, log *slog.Logger) (*Notifier, error) {
	loc, err := time.LoadLocation(conf.Location)
	if err != nil {
		return nil, fmt.Errorf("error loading timezone %q: %w", conf.Location, err)
	}

	return &Notifier{
		todoistClient: todoistClient,
		msgPublisher:  msgPublisher,
		chatID:        strconv.FormatInt(conf.TelegramChatID, 10),

		now: func() time.Time {
			return time.Now().In(loc)
		},
		log: log,
	}, nil
}

func (n *Notifier) SendNotification(ctx context.Context) error {
	n.log.InfoContext(ctx, "sending notification")

	tasks, err := n.todoistClient.GetTasks(ctx, false)
	if err != nil {
		return fmt.Errorf("get tasks: %w", err)
	}

	tasks = FilterAndSortTasks(tasks, n.now(), true)
	if len(tasks) == 0 {
		n.log.DebugContext(ctx, "no tasks for today")
		return nil
	}

	msg, err := RenderTasksMessage(tasks)
	if err != nil {
		return fmt.Errorf("render asks message: %w", err)
	}

	if err = n.msgPublisher.SendMessage(ctx, n.chatID, msg); err != nil {
		return fmt.Errorf("publish message: %w", err)
	}

	n.log.InfoContext(ctx, "message sent")
	return nil
}
