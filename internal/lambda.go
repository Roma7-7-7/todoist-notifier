package internal

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
)

type (
	TodoistClient interface {
		GetTasks(ctx context.Context, isCompleted bool) ([]todoist.Task, error)
	}

	HTTPMessagePublisher interface {
		SendMessage(ctx context.Context, chatID, message string) error
	}

	LambdaHandler struct {
		todoistClient TodoistClient
		msgPublisher  HTTPMessagePublisher
		chatID        string

		now func() time.Time
		log *slog.Logger
	}
)

func NewLambdaHandler(todoistClient TodoistClient, msgPublisher HTTPMessagePublisher, chatID string, log *slog.Logger) (*LambdaHandler, error) {
	location := os.Getenv("LOCATION")
	if location == "" {
		location = "Europe/Kyiv"
	}
	loc, err := time.LoadLocation(location)
	if err != nil {
		return nil, fmt.Errorf("error loading timezone %q: %w", location, err)
	}

	return &LambdaHandler{
		todoistClient: todoistClient,
		msgPublisher:  msgPublisher,
		chatID:        chatID,

		now: func() time.Time {
			return time.Now().In(loc)
		},
		log: log,
	}, nil
}

func (h *LambdaHandler) HandleRequest(ctx context.Context) error {
	h.log.InfoContext(ctx, "handle request")

	now := h.now()
	if now.Hour() < 9 || now.Hour() > 23 {
		h.log.DebugContext(ctx, "not a working hour")
		return nil
	}

	tasks, err := h.todoistClient.GetTasks(ctx, false)
	if err != nil {
		return fmt.Errorf("get tasks: %w", err)
	}

	tasks = FilterAndSortTasks(tasks, now)
	if len(tasks) == 0 {
		h.log.DebugContext(ctx, "no tasks for today")
		return nil
	}

	msg, err := RenderTasksMessage(tasks)
	if err != nil {
		return fmt.Errorf("render asks message: %w", err)
	}

	if err = h.msgPublisher.SendMessage(ctx, h.chatID, msg); err != nil {
		return fmt.Errorf("publish message: %w", err)
	}

	h.log.Info("message sent")
	return nil
}
