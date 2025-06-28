package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type (
	HTTPClient interface {
		Do(req *http.Request) (*http.Response, error)
	}

	HTTPMessagePublisher struct {
		httpClient HTTPClient
		botToken   string
	}

	sendMessageRequest struct {
		ChatID string `json:"chat_id"`
		Text   string `json:"text"`
	}
)

func NewHTTPMessagePublisher(httpClient HTTPClient, botToken string) *HTTPMessagePublisher {
	return &HTTPMessagePublisher{
		httpClient: httpClient,
		botToken:   botToken,
	}
}

func (p *HTTPMessagePublisher) SendMessage(ctx context.Context, chatID, text string) error {
	body, err := json.Marshal(sendMessageRequest{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", p.botToken), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
