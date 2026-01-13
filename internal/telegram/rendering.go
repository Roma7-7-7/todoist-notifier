package telegram

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
)

var todayTasksTemplate = template.Must(template.New("tasks").
	Funcs(template.FuncMap{
		"toCircle": priorityEmoji,
	}).
	Parse(`Uncompleted tasks for today:
{{- range .}}
- {{.Priority | toCircle}} {{ .Content }}
{{- end}}
`))

func RenderTasks(tasks []todoist.Task) (string, error) {
	if len(tasks) == 0 {
		return "", nil
	}

	buff := &bytes.Buffer{}
	if err := todayTasksTemplate.Execute(buff, tasks); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buff.String(), nil
}

func priorityEmoji(priority todoist.Priority) string {
	switch priority {
	case todoist.P1:
		return "🔴"
	case todoist.P2:
		return "🟠"
	case todoist.P3:
		return "🔵"
	case todoist.P4:
		return "⚪"
	default:
		return "⚪"
	}
}
