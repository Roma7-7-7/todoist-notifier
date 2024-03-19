package internal

import "context"

type StubPublisher struct {
	log Logger
}

func (p *StubPublisher) Publish(ctx context.Context, msg string) error {
	p.log.InfoContext(ctx, "publish message", "msg", msg)
	return nil
}
