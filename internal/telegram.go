package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gopkg.in/telebot.v3"
)

type (
	HTTPClient interface {
		Do(req *http.Request) (*http.Response, error)
	}

	MessagePublisher struct {
		chatID int64
		bot    *telebot.Bot
		log    Logger
	}

	HTTPMessagePublisher struct {
		httpClient HTTPClient
		botToken   string
	}

	Bot struct {
		bot *telebot.Bot
		log Logger
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

func NewBot(token string, statusJob func(ctx context.Context, notifyIfNoTasks bool) error, log Logger) (*Bot, error) {
	pref := telebot.Settings{
		Token: token,
		Poller: &telebot.LongPoller{
			Timeout: 10 * time.Second,
		},
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}

	initBotHandlers(b, statusJob, log)

	return &Bot{
		bot: b,
		log: log,
	}, nil
}

func (b *Bot) Run(ctx context.Context) {
	go func() {
		select {
		case <-ctx.Done():
			b.log.InfoContext(ctx, "stop bot")
			b.bot.Stop()
		}
	}()

	b.log.InfoContext(ctx, "start bot")
	b.bot.Start()
}

func (p *MessagePublisher) Publish(ctx context.Context, msg string) error {
	p.log.DebugContext(ctx, "publish message", "msg", msg)
	_, err := p.bot.Send(&telebot.Chat{ID: p.chatID}, msg)
	return err
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

func initBotHandlers(b *telebot.Bot, statusJob func(ctx context.Context, notifyIfNoTasks bool) error, log Logger) {
	b.Handle("/start", func(c telebot.Context) error {
		log.Info("start command", "chatID", c.Chat().ID)
		return c.Reply("Hello!")
	})

	b.Handle("/status", func(c telebot.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		log.Info("status command", "chatID", c.Chat().ID)
		if err := statusJob(ctx, true); err != nil {
			log.Error("failed to run status job", "error", err)
			return c.Reply(fmt.Sprintf("Error: %v", err))
		}
		return nil
	})
}
