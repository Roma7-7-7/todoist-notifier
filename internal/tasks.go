package internal

import (
	"bytes"
	"fmt"
	"log/slog"
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

var KyivTime *time.Location

func FilterAndSortTasks(tasks []todoist.Task) []todoist.Task {
	if len(tasks) == 0 {
		return nil
	}

	date := time.Now().In(KyivTime).Format("2006-01-02")
	res := make([]todoist.Task, 0, len(tasks))
	for _, t := range tasks {
		if t.Due != nil && t.Due.Date == date {
			res = append(res, t)
		}
	}

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Priority == tasks[j].Priority {
			return tasks[i].ProjectID < tasks[j].ProjectID
		}

		return tasks[i].Priority > tasks[j].Priority
	})

	return res
}

func RenderTasksMessage(tasks []todoist.Task) (string, error) {
	buff := &bytes.Buffer{}
	if err := tasksTemplate.Execute(buff, tasks); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buff.String(), nil
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

func init() {
	loc, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		panic(err)
	}
	KyivTime = loc
	slog.Info("initialized kyiv time location", "current_time", time.Now().In(KyivTime).Format(time.RFC3339))
}
