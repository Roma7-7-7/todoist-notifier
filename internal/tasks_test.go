package internal

import (
	"testing"
	"time"

	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
)

func TestFilterAndSortTasks_PriorityFiltering(t *testing.T) {
	date := "2026-01-11"
	tests := []struct {
		name     string
		tasks    []todoist.Task
		hour     int
		expected []string
	}{
		{
			name: "P1 shows all the time",
			tasks: []todoist.Task{
				{ID: "1", Content: "P1 task at 9am", Priority: 4, Due: &todoist.TaskDue{Date: date}},
			},
			hour:     9,
			expected: []string{"P1 task at 9am"},
		},
		{
			name: "P4 shows all the time",
			tasks: []todoist.Task{
				{ID: "2", Content: "P4 task at 9am", Priority: 1, Due: &todoist.TaskDue{Date: date}},
			},
			hour:     9,
			expected: []string{"P4 task at 9am"},
		},
		{
			name: "P2 filtered before 3pm",
			tasks: []todoist.Task{
				{ID: "3", Content: "P2 task", Priority: 3, Due: &todoist.TaskDue{Date: date}},
			},
			hour:     14,
			expected: []string{},
		},
		{
			name: "P2 shows after 3pm",
			tasks: []todoist.Task{
				{ID: "4", Content: "P2 task", Priority: 3, Due: &todoist.TaskDue{Date: date}},
			},
			hour:     15,
			expected: []string{"P2 task"},
		},
		{
			name: "P3 filtered before 6pm",
			tasks: []todoist.Task{
				{ID: "5", Content: "P3 task", Priority: 2, Due: &todoist.TaskDue{Date: date}},
			},
			hour:     17,
			expected: []string{},
		},
		{
			name: "P3 shows after 6pm",
			tasks: []todoist.Task{
				{ID: "6", Content: "P3 task", Priority: 2, Due: &todoist.TaskDue{Date: date}},
			},
			hour:     18,
			expected: []string{"P3 task"},
		},
		{
			name: "time label takes precedence over priority",
			tasks: []todoist.Task{
				{ID: "7", Content: "P2 with 6pm label", Priority: 3, Labels: []string{"6pm"}, Due: &todoist.TaskDue{Date: date}},
			},
			hour:     15,
			expected: []string{},
		},
		{
			name: "time label takes precedence - shows after time",
			tasks: []todoist.Task{
				{ID: "8", Content: "P2 with 6pm label", Priority: 3, Labels: []string{"6pm"}, Due: &todoist.TaskDue{Date: date}},
			},
			hour:     18,
			expected: []string{"P2 with 6pm label"},
		},
		{
			name: "mixed priorities at 10am",
			tasks: []todoist.Task{
				{ID: "9", Content: "P1 task", Priority: 4, Due: &todoist.TaskDue{Date: date}},
				{ID: "10", Content: "P2 task", Priority: 3, Due: &todoist.TaskDue{Date: date}},
				{ID: "11", Content: "P3 task", Priority: 2, Due: &todoist.TaskDue{Date: date}},
				{ID: "12", Content: "P4 task", Priority: 1, Due: &todoist.TaskDue{Date: date}},
			},
			hour:     10,
			expected: []string{"P1 task", "P4 task"},
		},
		{
			name: "mixed priorities at 4pm",
			tasks: []todoist.Task{
				{ID: "13", Content: "P1 task", Priority: 4, Due: &todoist.TaskDue{Date: date}},
				{ID: "14", Content: "P2 task", Priority: 3, Due: &todoist.TaskDue{Date: date}},
				{ID: "15", Content: "P3 task", Priority: 2, Due: &todoist.TaskDue{Date: date}},
				{ID: "16", Content: "P4 task", Priority: 1, Due: &todoist.TaskDue{Date: date}},
			},
			hour:     16,
			expected: []string{"P1 task", "P2 task", "P4 task"},
		},
		{
			name: "mixed priorities at 7pm",
			tasks: []todoist.Task{
				{ID: "17", Content: "P1 task", Priority: 4, Due: &todoist.TaskDue{Date: date}},
				{ID: "18", Content: "P2 task", Priority: 3, Due: &todoist.TaskDue{Date: date}},
				{ID: "19", Content: "P3 task", Priority: 2, Due: &todoist.TaskDue{Date: date}},
				{ID: "20", Content: "P4 task", Priority: 1, Due: &todoist.TaskDue{Date: date}},
			},
			hour:     19,
			expected: []string{"P1 task", "P2 task", "P3 task", "P4 task"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2026, 1, 11, tt.hour, 0, 0, 0, time.UTC)
			result := FilterAndSortTasks(tt.tasks, now, true)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d tasks, got %d", len(tt.expected), len(result))
				return
			}

			for i, expectedContent := range tt.expected {
				if result[i].Content != expectedContent {
					t.Errorf("expected task[%d] to be %q, got %q", i, expectedContent, result[i].Content)
				}
			}
		})
	}
}
