package telegram

import (
	"fmt"

	tele "gopkg.in/telebot.v3"
)

func (b *Bot) HandleTasks(c tele.Context) error {
	return b.SendTasks(c.Chat().ID, true)
}

func (b *Bot) SendTasks(chatID int64, manualRequestMode bool) error {
	ctx, cancel := b.context()
	defer cancel()

	b.log.DebugContext(ctx, "received /tasks command", "chat_id", chatID)

	tasks, err := b.tasksService.GetTodayTasks(ctx, true)
	if err != nil {
		return fmt.Errorf("get today tasks: %w", err)
	}

	msg, err := RenderTasks(tasks)
	if err != nil {
		return fmt.Errorf("render today tasks: %w", err)
	}

	if msg == "" {
		if manualRequestMode {
			msg = "No tasks for today! 🎉"
		} else {
			b.log.DebugContext(ctx, "no tasks to send")
			return nil
		}
	}

	if _, err := b.bot.Send(&tele.Chat{ID: chatID}, msg); err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	b.log.DebugContext(ctx, "tasks sent successfully")
	return nil
}
