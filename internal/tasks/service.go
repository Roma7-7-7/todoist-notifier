package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/Roma7-7-7/todoist-notifier/pkg/cache"
	"github.com/Roma7-7-7/todoist-notifier/pkg/clock"
	"github.com/Roma7-7-7/todoist-notifier/pkg/todoist"
)

type TodoistClient interface {
	GetTasks(ctx context.Context, includeCompleted bool) ([]todoist.Task, error)
	GetProjects(ctx context.Context) ([]todoist.Project, error)
	UpdateTask(ctx context.Context, taskID string, req todoist.UpdateTaskRequest) (*todoist.Task, error)
}

type Service struct {
	todoistClient TodoistClient
	projectsCache *cache.TTL[[]todoist.Project]
	clock         clock.Interface
}

func NewService(todoistClient TodoistClient, clock clock.Interface, projectCacheTTL time.Duration) *Service {
	projectsCache := cache.NewTTL(func(ctx context.Context) ([]todoist.Project, error) {
		return todoistClient.GetProjects(ctx)
	}, projectCacheTTL)

	return &Service{
		todoistClient: todoistClient,
		projectsCache: projectsCache,
		clock:         clock,
	}
}

func (s *Service) GetTodayTasks(ctx context.Context, filterByTime bool) ([]todoist.Task, error) {
	allTasks, err := s.todoistClient.GetTasks(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("get tasks: %w", err)
	}
	return FilterToday(allTasks, s.clock.Now(), filterByTime), nil
}

func (s *Service) GetTomorrowTasks(ctx context.Context) ([]todoist.Task, error) {
	allTasks, err := s.todoistClient.GetTasks(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("get tasks: %w", err)
	}
	return FilterTomorrow(allTasks, s.clock.Now()), nil
}

func (s *Service) GetTomorrowUnprioritized(ctx context.Context) ([]todoist.Task, error) {
	allTasks, err := s.todoistClient.GetTasks(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("get tasks: %w", err)
	}
	tomorrowTasks := FilterTomorrow(allTasks, s.clock.Now())
	return FilterUnprioritized(tomorrowTasks), nil
}

func (s *Service) GetProjects(ctx context.Context) ([]todoist.Project, error) {
	return s.projectsCache.Get(ctx)
}

func (s *Service) UpdateTask(ctx context.Context, taskID string, priority todoist.Priority, labels []string) (*todoist.Task, error) {
	res, err := s.todoistClient.UpdateTask(ctx, taskID, todoist.UpdateTaskRequest{
		Priority: priority,
		Labels:   labels,
	})
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	return res, nil
}
