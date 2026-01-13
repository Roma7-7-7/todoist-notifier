package internal

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/Roma7-7-7/todoist-notifier/internal"
	"github.com/Roma7-7-7/todoist-notifier/internal/telegram"
	"github.com/Roma7-7-7/todoist-notifier/pkg/clock"
	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
)

type (
	HTTPMessagePublisher interface {
		SendMessage(ctx context.Context, chatID, message string) error
	}

	TasksService interface {
		GetTodayTasks(context.Context, bool) ([]todoist.Task, error)
	}

	Handler struct {
		todoistClient *todoist.Client
		msgPublisher  HTTPMessagePublisher
		tasksService  TasksService

		chatID string

		clock clock.Interface
		log   *slog.Logger
	}
)

func NewHandler(conf *internal.Config, todoistClient *todoist.Client, msgPublisher HTTPMessagePublisher, tasksService TasksService, log *slog.Logger) (*Handler, error) {
	loc, err := time.LoadLocation(conf.Location)
	if err != nil {
		return nil, fmt.Errorf("error loading timezone %q: %w", conf.Location, err)
	}

	return &Handler{
		todoistClient: todoistClient,
		msgPublisher:  msgPublisher,
		tasksService:  tasksService,

		chatID: strconv.FormatInt(conf.TelegramChatID, 10),

		clock: clock.NewZonedClock(loc),
		log:   log,
	}, nil
}

func (h *Handler) HandleRequest(ctx context.Context) error {
	h.log.InfoContext(ctx, "sending notification")

	tasks, err := h.tasksService.GetTodayTasks(ctx, true)
	if err != nil {
		return fmt.Errorf("get today tasks: %w", err)
	}
	msg, err := telegram.RenderTasks(tasks)
	if err != nil {
		return fmt.Errorf("render tasks: %w", err)
	}

	if msg == "" {
		h.log.DebugContext(ctx, "no tasks for today")
		return nil
	}

	if err = h.msgPublisher.SendMessage(ctx, h.chatID, msg); err != nil {
		return fmt.Errorf("publish message: %w", err)
	}

	h.log.InfoContext(ctx, "message sent")
	return nil
}
