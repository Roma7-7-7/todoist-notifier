package internal

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
)

type (
	TodoistClient interface {
		GetTasksV2(ctx context.Context, isCompleted bool) ([]todoist.Task, error)
	}

	LambdaHandler struct {
		todoistClient TodoistClient
		msgPublisher  *HTTPMessagePublisher
		chatID        string
		log           *slog.Logger
	}
)

func NewLambdaHandler(todoistClient TodoistClient, msgPublisher *HTTPMessagePublisher, chatID string, log *slog.Logger) *LambdaHandler {
	return &LambdaHandler{
		todoistClient: todoistClient,
		msgPublisher:  msgPublisher,
		chatID:        chatID,
		log:           log,
	}
}

func (h *LambdaHandler) HandleRequest(ctx context.Context) error {
	h.log.InfoContext(ctx, "handle request")

	now := time.Now().In(KyivTime)
	if now.Hour() < 9 || now.Hour() > 23 {
		h.log.DebugContext(ctx, "not a working hour")
		return nil
	}

	tasks, err := h.todoistClient.GetTasksV2(ctx, false)
	if err != nil {
		return fmt.Errorf("get tasks: %w", err)
	}

	tasks = FilterAndSortTasks(tasks)
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
