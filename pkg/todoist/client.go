package todoist

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const baseURL = "https://api.todoist.com/rest"

type (
	Logger interface {
		DebugContext(ctx context.Context, msg string, fields ...any)
		WarnContext(ctx context.Context, msg string, fields ...any)
	}

	HTTPClient interface {
		Do(req *http.Request) (*http.Response, error)
	}

	Task struct {
		ID        string   `json:"id"`
		ProjectID string   `json:"project_id"`
		Content   string   `json:"content"`
		Priority  int      `json:"priority"`
		Due       *TaskDue `json:"due"`
	}

	TaskDue struct {
		Date string `json:"date"`
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

func (c *Client) GetTasksV2(ctx context.Context, isCompleted bool) ([]Task, error) {
	res := make([]Task, 0, 100)

	req, err := http.NewRequest("GET", baseURL+"/v2/tasks", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	q := req.URL.Query()
	q.Add("is_completed", fmt.Sprintf("%t", isCompleted))
	req.URL.RawQuery = q.Encode()

	c.log.DebugContext(ctx, "sending request", slog.String("url", req.URL.String()),
		slog.String("method", req.Method),
		slog.String("token", c.token),
		slog.Bool("is_completed", isCompleted))

	req = req.WithContext(ctx)
	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		c.log.WarnContext(ctx, "unexpected status code", slog.Int("status_code", resp.StatusCode), slog.String("body", string(body[:n])))

		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return res, nil
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
