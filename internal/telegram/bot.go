package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Roma7-7-7/todoist-notifier/pkg/clock"
	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
	tele "gopkg.in/telebot.v3"
)

const (
	defaultTimeout  = 10 * time.Second
	defaultErrorMsg = "Something went wrong. Please try again later."
)

type (
	TasksService interface {
		GetTodayTasks(context.Context, bool) ([]todoist.Task, error)
		GetTomorrowUnprioritized(context.Context) ([]todoist.Task, error)
		GetProjects(context.Context) ([]todoist.Project, error)
		UpdateTask(context.Context, string, todoist.Priority, []string) (*todoist.Task, error)
	}

	Bot struct {
		bot *tele.Bot

		tasksService TasksService
		clock        clock.Interface

		allowedChatID int64
		states        *prioritizationStore

		log *slog.Logger
	}
)

func NewBot(token string, tasksService TasksService, allowedChatID int64, clock clock.Interface, log *slog.Logger) (*Bot, error) {
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
		tasksService:  tasksService,
		allowedChatID: allowedChatID,
		clock:         clock,
		states: &prioritizationStore{
			states: make(map[string]*taskState),
			clock:  clock,
			mx:     &sync.RWMutex{},
			log:    log,
		},
		log: log,
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
	b.bot.Handle("/tasks", b.HandleTasks)
	b.bot.Handle("/prioritize", b.HandlePrioritize)

	b.bot.Handle(tele.OnCallback, b.handleCallback)
}

func (b *Bot) handleCallback(c tele.Context) error {
	callback := c.Callback()
	if callback == nil {
		return nil
	}

	parts := strings.Split(callback.Data, "|")
	if len(parts) < 3 || parts[0] != callbackPrefix {
		return nil
	}

	action := parts[1]
	taskID := parts[2]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //nolint:mnd // reasonable timeout
	defer cancel()

	b.states.cleanupExpiredStates(ctx)

	switch action {
	case "priority":
		if len(parts) < 4 {
			return b.respondError(c, "Invalid callback data")
		}
		pi, err := strconv.Atoi(parts[3])
		if err != nil {
			return b.respondError(c, "Invalid priority value")
		}
		return b.handlePriorityCallback(ctx, c, taskID, todoist.Priority(pi))
	case "time":
		if len(parts) < 4 {
			return b.respondError(c, "Invalid callback data")
		}
		timeLabel := parts[3]
		return b.handleTimeLabelCallback(ctx, c, taskID, timeLabel)
	case "project":
		if len(parts) < 4 {
			return b.respondError(c, "Invalid callback data")
		}
		projectChoice := parts[3]
		return b.handleProjectCallback(ctx, c, taskID, projectChoice)
	case "project_select":
		if len(parts) < 4 {
			return b.respondError(c, "Invalid callback data")
		}
		projectID := parts[3]
		return b.handleProjectSelectCallback(ctx, c, taskID, projectID)
	default:
		return nil
	}
}

func (b *Bot) context() (context.Context, func()) {
	return context.WithTimeout(context.Background(), defaultTimeout)
}

func (b *Bot) recover(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		defer func() {
			if r := recover(); r != nil {
				b.log.Error("panic recovered", "error", r)
			}
		}()

		return next(c)
	}
}

func (b *Bot) chatIDMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if c.Chat().ID != b.allowedChatID {
			b.log.Warn("unauthorized chat access blocked",
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
			b.log.Error("error occurred", args...)
			return c.Send(defaultErrorMsg)
		}
		return err
	}
}
