package internal

import (
	"context"
	"fmt"
	"log/slog"
)

type LambdaHandler struct {
	notifier *Notifier
}

func NewLambdaHandler(conf *Config, todoistClient TodoistClient, msgPublisher HTTPMessagePublisher, log *slog.Logger) (*LambdaHandler, error) {
	notifier, err := NewNotifier(conf, todoistClient, msgPublisher, log)
	if err != nil {
		return nil, fmt.Errorf("create notifier: %w", err)
	}

	return &LambdaHandler{
		notifier: notifier,
	}, nil
}

func (h *LambdaHandler) HandleRequest(ctx context.Context) error {
	return h.notifier.SendNotification(ctx)
}
