package internal

import (
	"bytes"
	"fmt"
	"sort"
	"text/template"
	"time"

	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
)

type Priority int

const (
	P1 Priority = 4
	P2 Priority = 3
	P3 Priority = 2
	P4 Priority = 1
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

func FilterAndSortTasks(tasks []todoist.Task, now time.Time, filterByTime bool) []todoist.Task {
	if len(tasks) == 0 {
		return nil
	}

	date := now.Format("2006-01-02")
	res := make([]todoist.Task, 0, len(tasks))
	for _, t := range tasks {
		if t.Due == nil || t.Due.Date != date {
			continue
		}

		if !filterByTime {
			res = append(res, t)
			continue
		}

		labels := timeLabels(t)
		hasTimeLabel := len(labels) > 0

		if hasTimeLabel {
			switch {
			case labels["12pm"] && now.Hour() < 12: //nolint:mnd // 12pm
				continue
			case labels["3pm"] && now.Hour() < 15: //nolint:mnd // 3pm
				continue
			case labels["6pm"] && now.Hour() < 18: //nolint:mnd // 6pm
				continue
			case labels["9pm"] && now.Hour() < 21: //nolint:mnd // 9pm
				continue
			default:
				res = append(res, t)
			}
		} else {
			switch Priority(t.Priority) {
			case P1, P4:
				res = append(res, t)
			case P2:
				if now.Hour() >= 15 { //nolint:mnd // 3pm
					res = append(res, t)
				}
			case P3:
				if now.Hour() >= 18 { //nolint:mnd // 6pm
					res = append(res, t)
				}
			}
		}
	}

	sort.Slice(res, func(i, j int) bool {
		if res[i].Priority == res[j].Priority {
			return res[i].ProjectID < res[j].ProjectID
		}

		return res[i].Priority > res[j].Priority
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

func timeLabels(task todoist.Task) map[string]bool {
	res := make(map[string]bool, len(task.Labels))
	for _, l := range task.Labels {
		switch l {
		case "3pm", "6pm", "9pm", "12pm":
			res[l] = true
		default:
			continue
		}
	}
	return res
}
