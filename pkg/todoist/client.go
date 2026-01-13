package todoist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const baseURL = "https://api.todoist.com/rest"

const (
	P1 Priority = 4
	P2 Priority = 3
	P3 Priority = 2
	P4 Priority = 1
)

type (
	Logger interface {
		DebugContext(ctx context.Context, msg string, fields ...any)
		InfoContext(ctx context.Context, msg string, fields ...any)
		WarnContext(ctx context.Context, msg string, fields ...any)
	}

	HTTPClient interface {
		Do(req *http.Request) (*http.Response, error)
	}

	Priority int

	Task struct {
		ID        string   `json:"id"`
		ProjectID string   `json:"project_id"`
		Content   string   `json:"content"`
		Priority  Priority `json:"priority"`
		Due       *TaskDue `json:"due"`
		Labels    []string `json:"labels"`
	}

	TaskDue struct {
		Date string `json:"date"`
	}

	Project struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Color          string `json:"color"`
		ParentID       string `json:"parent_id"`
		Order          int    `json:"order"`
		CommentCount   int    `json:"comment_count"`
		IsShared       bool   `json:"is_shared"`
		IsFavorite     bool   `json:"is_favorite"`
		IsInboxProject bool   `json:"is_inbox_project"`
		IsTeamInbox    bool   `json:"is_team_inbox"`
		ViewStyle      string `json:"view_style"`
		URL            string `json:"url"`
	}

	UpdateTaskRequest struct {
		Priority Priority `json:"priority,omitempty,omitzero"`
		Labels   []string `json:"labels,omitempty,omitzero"`
	}

	Client struct {
		token        string
		httpClient   HTTPClient
		retriesCount int
		retriesDelay time.Duration
		log          Logger
	}
)

func NewClient(token string, httpClient HTTPClient, retriesCount int, retriesDelay time.Duration, log Logger) *Client {
	return &Client{
		token:        token,
		httpClient:   httpClient,
		retriesCount: retriesCount,
		retriesDelay: retriesDelay,
		log:          log,
	}
}

func (c *Client) GetTasks(ctx context.Context, isCompleted bool) ([]Task, error) {
	res := make([]Task, 0, 100)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v2/tasks", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	q := req.URL.Query()
	q.Add("is_completed", fmt.Sprintf("%t", isCompleted))
	req.URL.RawQuery = q.Encode()

	c.log.DebugContext(ctx, "sending request",
		"url", req.URL.String(),
		"method", req.Method,
		"token", c.token,
		"is_completed", isCompleted)

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // ignore

	if resp.StatusCode != http.StatusOK {
		c.log.WarnContext(ctx, "unexpected status code", "status_code", resp.StatusCode)

		body := make([]byte, 1024) //nolint:mnd //it's ok
		n, _ := resp.Body.Read(body)
		c.log.DebugContext(ctx, "response payload", "payload", string(body[:n]))

		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return res, nil
}

func (c *Client) GetProjects(ctx context.Context) ([]Project, error) {
	res := make([]Project, 0, 50)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v2/projects", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	c.log.DebugContext(ctx, "sending get projects request",
		"url", req.URL.String(),
		"method", req.Method)

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // ignore

	if resp.StatusCode != http.StatusOK {
		c.log.WarnContext(ctx, "unexpected status code", "status_code", resp.StatusCode)

		body := make([]byte, 1024) //nolint:mnd //it's ok
		n, _ := resp.Body.Read(body)
		c.log.DebugContext(ctx, "response payload", "payload", string(body[:n]))

		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return res, nil
}

func (c *Client) UpdateTask(ctx context.Context, taskID string, updateReq UpdateTaskRequest) (*Task, error) {
	body, err := json.Marshal(updateReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v2/tasks/"+taskID, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	c.log.DebugContext(ctx, "sending update task request",
		"url", req.URL.String(),
		"method", req.Method,
		"task_id", taskID,
		"body", string(body))

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.log.WarnContext(ctx, "unexpected status code", "status_code", resp.StatusCode)

		body := make([]byte, 1024) //nolint:mnd //it's ok
		n, _ := resp.Body.Read(body)
		c.log.DebugContext(ctx, "response payload", "payload", string(body[:n]))

		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var task Task
	if err = json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &task, nil
}

func (c *Client) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var (
		resp *http.Response
		err  error
	)

	for i := 0; i < c.retriesCount; i++ {
		resp, err = c.httpClient.Do(req)
		if err == nil {
			return resp, nil
		}

		c.log.WarnContext(ctx, "request failed", "retry", i+1, "error", err)
		time.Sleep(c.retriesDelay)
	}

	return nil, fmt.Errorf("do request with %d retries: %w", c.retriesCount, err)
}
