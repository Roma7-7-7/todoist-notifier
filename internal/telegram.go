package internal

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/telebot.v3"
)

type (
	MessagePublisher struct {
		chatID int64
		bot    *telebot.Bot
		log    Logger
	}

	Bot struct {
		bot *telebot.Bot
		log Logger
	}
)

func NewBot(token string, log Logger) (*Bot, error) {
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

	initBotHandlers(b, log)

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

func initBotHandlers(b *telebot.Bot, log Logger) {
	b.Handle("/start", func(c telebot.Context) error {
		log.Info("start command", "chatID", c.Chat().ID)
		return c.Reply("Hello!")
	})
}
