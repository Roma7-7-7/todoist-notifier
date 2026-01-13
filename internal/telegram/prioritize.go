package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Roma7-7-7/todoist-notifier/pkg/clock"
	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
	tele "gopkg.in/telebot.v3"
)

const (
	callbackPrefix = "prio"
	stateTimeout   = 24 * time.Hour
)

type (
	taskState struct {
		taskID    string
		content   string
		projectID string
		priority  todoist.Priority
		timeLabel string
		createdAt time.Time
	}
)

func (b *Bot) HandlePrioritize(c tele.Context) error {
	return b.Prioritize(c.Chat().ID)
}

func (b *Bot) Prioritize(chatID int64) error {
	ctx, cancel := b.context()
	defer cancel()

	b.log.DebugContext(ctx, "checking unprioritized tasks for tomorrow")

	unprioritized, err := b.tasksService.GetTomorrowUnprioritized(ctx)
	if err != nil {
		return fmt.Errorf("get tomorrow unprioritized: %w", err)
	}

	if len(unprioritized) == 0 {
		b.log.InfoContext(ctx, "no unprioritized tasks for tomorrow")
		return nil
	}

	b.log.InfoContext(ctx, "found unprioritized tasks", "count", len(unprioritized))

	introMsg := fmt.Sprintf("Found %d task(s) for tomorrow that need prioritization:", len(unprioritized))
	if _, err := b.bot.Send(&tele.Chat{ID: chatID}, introMsg); err != nil {
		return fmt.Errorf("send intro message: %w", err)
	}

	for _, task := range unprioritized {
		if err := b.sendPrioritySelection(ctx, chatID, task); err != nil {
			b.log.ErrorContext(ctx, "failed to send priority selection", "task_id", task.ID, "error", err)
			continue
		}
	}

	return nil
}

func (b *Bot) sendPrioritySelection(ctx context.Context, chatID int64, task todoist.Task) error {
	state := &taskState{
		taskID:    task.ID,
		content:   task.Content,
		projectID: task.ProjectID,
		priority:  task.Priority,
		createdAt: b.clock.Now(),
	}
	b.states.saveState(task.ID, state)

	msg := fmt.Sprintf("📋 Task: %s\n\nSelect priority:", task.Content)

	keyboard := &tele.ReplyMarkup{}
	keyboard.Inline(
		keyboard.Row(
			keyboard.Data("🔴 P1", callbackPrefix, "priority", task.ID, "4"),
			keyboard.Data("🟠 P2", callbackPrefix, "priority", task.ID, "3"),
		),
		keyboard.Row(
			keyboard.Data("🔵 P3", callbackPrefix, "priority", task.ID, "2"),
			keyboard.Data("⚪ P4", callbackPrefix, "priority", task.ID, "1"),
		),
	)

	if _, err := b.bot.Send(&tele.Chat{ID: chatID}, msg, keyboard); err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

func (b *Bot) handlePriorityCallback(ctx context.Context, c tele.Context, taskID string, priority todoist.Priority) error {
	state := b.states.getState(taskID)
	if state == nil {
		return b.respondError(c, "Task state expired. Please start again.")
	}

	state.priority = priority

	if err := c.Respond(); err != nil {
		b.log.WarnContext(ctx, "failed to respond to callback", "error", err)
	}

	if priority == 4 || priority == 1 {
		return b.sendTimeLabelSelection(ctx, c, taskID, state.content, priorityEmoji(priority), priority)
	}

	return b.sendProjectSelection(ctx, c, taskID, state.content, priorityEmoji(priority), "")
}

func (b *Bot) sendTimeLabelSelection(ctx context.Context, c tele.Context, taskID, content, priorityStr string, priority todoist.Priority) error {
	msg := fmt.Sprintf("📋 Task: %s\nPriority: %s\n\nSelect time:", content, priorityStr)

	keyboard := &tele.ReplyMarkup{}

	var rows []tele.Row
	if priority == 4 {
		rows = []tele.Row{
			keyboard.Row(
				keyboard.Data("🕛 12PM", callbackPrefix, "time", taskID, "12pm"),
				keyboard.Data("⏭️ None", callbackPrefix, "time", taskID, "none"),
			),
		}
	} else {
		rows = []tele.Row{
			keyboard.Row(
				keyboard.Data("🕘 9PM", callbackPrefix, "time", taskID, "9pm"),
				keyboard.Data("⏭️ None", callbackPrefix, "time", taskID, "none"),
			),
		}
	}

	keyboard.Inline(rows...)

	if _, err := b.bot.Edit(c.Message(), msg, keyboard); err != nil {
		return fmt.Errorf("edit message: %w", err)
	}

	return nil
}

func (b *Bot) handleTimeLabelCallback(ctx context.Context, c tele.Context, taskID, timeLabel string) error {
	state := b.states.getState(taskID)
	if state == nil {
		return b.respondError(c, "Task state expired. Please start again.")
	}

	if timeLabel != "none" {
		state.timeLabel = timeLabel
	}

	if err := c.Respond(); err != nil {
		b.log.WarnContext(ctx, "failed to respond to callback", "error", err)
	}

	timeLabelStr := "None"
	if state.timeLabel != "" {
		timeLabelStr = state.timeLabel
	}

	return b.sendProjectSelection(ctx, c, taskID, state.content, priorityEmoji(state.priority), timeLabelStr)
}

func (b *Bot) sendProjectSelection(ctx context.Context, c tele.Context, taskID, content, priorityStr, timeLabelStr string) error {
	timeInfo := ""
	if timeLabelStr != "" {
		timeInfo = fmt.Sprintf("\nTime: %s", timeLabelStr)
	}

	msg := fmt.Sprintf("📋 Task: %s\nPriority: %s%s\n\nMove to project?", content, priorityStr, timeInfo)

	keyboard := &tele.ReplyMarkup{}
	keyboard.Inline(
		keyboard.Row(
			keyboard.Data("✅ Yes", callbackPrefix, "project", taskID, "yes"),
			keyboard.Data("⏭️ No", callbackPrefix, "project", taskID, "no"),
		),
	)

	if _, err := b.bot.Edit(c.Message(), msg, keyboard); err != nil {
		return fmt.Errorf("edit message: %w", err)
	}

	return nil
}

func (b *Bot) handleProjectCallback(ctx context.Context, c tele.Context, taskID, projectChoice string) error {
	state := b.states.getState(taskID)
	if state == nil {
		return b.respondError(c, "Task state expired. Please start again.")
	}

	if err := c.Respond(); err != nil {
		b.log.WarnContext(ctx, "failed to respond to callback", "error", err)
	}

	if projectChoice == "no" {
		return b.finalizeTask(ctx, c, taskID)
	}

	projects, err := b.tasksService.GetProjects(ctx)
	if err != nil {
		b.log.ErrorContext(ctx, "failed to get projects", "error", err)
		return b.respondError(c, "Failed to load projects. Finalizing with current project.")
	}

	return b.sendProjectList(ctx, c, taskID, state, projects)
}

func (b *Bot) sendProjectList(ctx context.Context, c tele.Context, taskID string, state *taskState, projects []todoist.Project) error {
	timeInfo := ""
	if state.timeLabel != "" {
		timeInfo = fmt.Sprintf("\nTime: %s", state.timeLabel)
	}

	msg := fmt.Sprintf("📋 Task: %s\nPriority: %s%s\n\nSelect project:", state.content, priorityEmoji(state.priority), timeInfo)

	keyboard := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(projects)/2+2)

	for i := 0; i < len(projects); i += 2 {
		if i+1 < len(projects) {
			rows = append(rows, keyboard.Row(
				keyboard.Data(projects[i].Name, callbackPrefix, "project_select", taskID, projects[i].ID),
				keyboard.Data(projects[i+1].Name, callbackPrefix, "project_select", taskID, projects[i+1].ID),
			))
		} else {
			rows = append(rows, keyboard.Row(
				keyboard.Data(projects[i].Name, callbackPrefix, "project_select", taskID, projects[i].ID),
			))
		}
	}

	rows = append(rows, keyboard.Row(
		keyboard.Data("⏭️ Keep Current", callbackPrefix, "project_select", taskID, "current"),
	))

	keyboard.Inline(rows...)

	if _, err := b.bot.Edit(c.Message(), msg, keyboard); err != nil {
		return fmt.Errorf("edit message: %w", err)
	}

	return nil
}

func (b *Bot) handleProjectSelectCallback(ctx context.Context, c tele.Context, taskID, projectID string) error {
	state := b.states.getState(taskID)
	if state == nil {
		return b.respondError(c, "Task state expired. Please start again.")
	}

	if projectID != "current" {
		state.projectID = projectID
	}

	if err := c.Respond(); err != nil {
		b.log.WarnContext(ctx, "failed to respond to callback", "error", err)
	}

	return b.finalizeTask(ctx, c, taskID)
}

func (b *Bot) finalizeTask(ctx context.Context, c tele.Context, taskID string) error {
	state := b.states.getState(taskID)
	if state == nil {
		return b.respondError(c, "Task state expired. Please start again.")
	}

	// todo preserve original non-time labels
	labels := []string{}
	if state.timeLabel != "" {
		labels = append(labels, state.timeLabel)
	}

	_, err := b.tasksService.UpdateTask(ctx, taskID, state.priority, labels)
	if err != nil {
		b.log.ErrorContext(ctx, "failed to update task", "task_id", taskID, "error", err)
		return b.respondError(c, "Failed to update task. Please try again.")
	}

	b.states.deleteState(taskID)

	timeInfo := ""
	if state.timeLabel != "" {
		timeInfo = fmt.Sprintf("\nTime: %s", state.timeLabel)
	}

	msg := fmt.Sprintf("✅ Task updated successfully!\n\n📋 Task: %s\nPriority: %s%s", state.content, priorityEmoji(state.priority), timeInfo)

	if _, err := b.bot.Edit(c.Message(), msg); err != nil {
		return fmt.Errorf("edit message: %w", err)
	}

	b.log.InfoContext(ctx, "task prioritized successfully", "task_id", taskID, "priority", state.priority, "time_label", state.timeLabel)

	return nil
}

func (b *Bot) respondError(c tele.Context, message string) error {
	if err := c.Respond(&tele.CallbackResponse{Text: message, ShowAlert: true}); err != nil {
		return fmt.Errorf("respond with error: %w", err)
	}
	return nil
}

type prioritizationStore struct {
	states map[string]*taskState
	clock  clock.Interface
	mx     *sync.RWMutex
	log    *slog.Logger
}

func (s *prioritizationStore) saveState(taskID string, state *taskState) {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.states[taskID] = state
}

func (s *prioritizationStore) getState(taskID string) *taskState {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.states[taskID]
}

func (s *prioritizationStore) deleteState(taskID string) {
	s.mx.Lock()
	defer s.mx.Unlock()
	delete(s.states, taskID)
}

func (s *prioritizationStore) cleanupExpiredStates(ctx context.Context) {
	s.mx.Lock()
	defer s.mx.Unlock()

	now := s.clock.Now()
	for taskID, state := range s.states {
		if now.Sub(state.createdAt) > stateTimeout {
			delete(s.states, taskID)
			s.log.DebugContext(ctx, "cleaned up expired state", "task_id", taskID)
		}
	}
}
