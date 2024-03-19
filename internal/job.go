package internal

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"text/template"
	"time"

	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
)

var tasksTemplate = template.Must(template.New("tasks").
	Funcs(template.FuncMap{
		"toCircle": toCircle,
	}).
	Parse(`Uncompleted tasks for today:
{{- range .}}
- {{.Priority | toCircle}} {{ .Content }}
{{- end}}
`))

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
	tasks = filterByDueDate(tasks, today)

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

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Priority > tasks[j].Priority && tasks[i].ProjectID >= tasks[j].ProjectID
	})

	buff := &bytes.Buffer{}
	if err = tasksTemplate.Execute(buff, tasks); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	if err = j.publisher.Publish(ctx, buff.String()); err != nil {
		return fmt.Errorf("publish message: %w", err)
	}

	return nil
}

func toCircle(priority int) string {
	switch priority {
	case 4:
		return "🔴"
	case 3:
		return "🟠"
	case 2:
		return "🔵"
	default:
		return "⚪"
	}
}

func filterByDueDate(tasks []todoist.Task, date string) []todoist.Task {
	res := make([]todoist.Task, 0, len(tasks))
	for _, t := range tasks {
		if t.Due != nil && t.Due.Date == date {
			res = append(res, t)
		}
	}
	return res
}
