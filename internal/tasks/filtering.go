package tasks

import (
	"sort"
	"time"

	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
)

func FilterToday(tasks []todoist.Task, now time.Time, filterByTime bool) []todoist.Task {
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

		if shouldShowTask(t, now) {
			res = append(res, t)
		}
	}

	sortTasks(res)
	return res
}

func FilterTomorrow(tasks []todoist.Task, now time.Time) []todoist.Task {
	if len(tasks) == 0 {
		return nil
	}

	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	res := make([]todoist.Task, 0, len(tasks))

	for _, t := range tasks {
		if t.Due != nil && t.Due.Date == tomorrow {
			res = append(res, t)
		}
	}

	sortTasks(res)
	return res
}

func FilterUnprioritized(tasks []todoist.Task) []todoist.Task {
	if len(tasks) == 0 {
		return nil
	}

	res := make([]todoist.Task, 0, len(tasks))

	for _, t := range tasks {
		hasTimeLabel := hasAnyTimeLabel(t.Labels)

		if !hasTimeLabel && t.Priority == todoist.P4 {
			res = append(res, t)
		}
	}

	sortTasks(res)
	return res
}

func shouldShowTask(task todoist.Task, now time.Time) bool {
	labels := extractTimeLabels(task)
	hasTimeLabel := len(labels) > 0

	if hasTimeLabel {
		switch {
		case labels["12pm"] && now.Hour() < 12:
			return false
		case labels["3pm"] && now.Hour() < 15:
			return false
		case labels["6pm"] && now.Hour() < 18:
			return false
		case labels["9pm"] && now.Hour() < 21:
			return false
		default:
			return true
		}
	}

	switch task.Priority {
	case todoist.P1, todoist.P4:
		return true
	case todoist.P2:
		return now.Hour() >= 15 //nolint:mnd // 3pm
	case todoist.P3:
		return now.Hour() >= 18 //nolint:mnd // 6pm
	default:
		return false
	}
}

func sortTasks(tasks []todoist.Task) {
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Priority == tasks[j].Priority {
			return tasks[i].ProjectID < tasks[j].ProjectID
		}
		return tasks[i].Priority > tasks[j].Priority
	})
}

func extractTimeLabels(task todoist.Task) map[string]bool {
	res := make(map[string]bool, len(task.Labels))
	for _, l := range task.Labels {
		switch l {
		case "3pm", "6pm", "9pm", "12pm":
			res[l] = true
		}
	}
	return res
}

func hasAnyTimeLabel(labels []string) bool {
	for _, label := range labels {
		if label == "12pm" || label == "3pm" || label == "6pm" || label == "9pm" {
			return true
		}
	}
	return false
}
