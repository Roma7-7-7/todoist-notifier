package internal

import (
	"context"
	"time"

	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
)

type Clock interface {
	Now() time.Time
}

type TodoistClient interface {
	GetTasks(ctx context.Context, includeCompleted bool) ([]todoist.Task, error)
}
