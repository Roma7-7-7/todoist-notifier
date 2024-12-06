package internal

import (
	"context"
	"fmt"
	"time"
)

type (
	Publisher interface {
		Publish(ctx context.Context, msg string) error
	}

	Job struct {
		todoistClient TodoistClient
		publisher     Publisher
		log           Logger
	}
)

func NewJob(todoistClient TodoistClient, publisher Publisher, log Logger) *Job {
	return &Job{
		todoistClient: todoistClient,
		publisher:     publisher,
		log:           log,
	}
}

func (j *Job) Run(ctx context.Context, notifyIfNoTasks bool) error {
	j.log.InfoContext(ctx, "run job")
	today := time.Now().Format("2006-01-02")

	j.log.DebugContext(ctx, "get tasks", "date", today)
	tasks, err := j.todoistClient.GetTasksV2(ctx, false)
	if err != nil {
		return fmt.Errorf("get tasks: %w", err)
	}
	tasks = FilterAndSortTasks(tasks)

	if len(tasks) == 0 {
		j.log.InfoContext(ctx, "no tasks for today")
		if notifyIfNoTasks {
			if err = j.publisher.Publish(ctx, "No tasks for today"); err != nil {
				return fmt.Errorf("publish message: %w", err)
			}
		}
		return nil
	}
	j.log.DebugContext(ctx, "tasks to be reminded", "count", len(tasks))

	msg, err := RenderTasksMessage(tasks)
	if err != nil {
		return fmt.Errorf("render asks message: %w", err)
	}

	if err = j.publisher.Publish(ctx, msg); err != nil {
		return fmt.Errorf("publish message: %w", err)
	}

	return nil
}
