package cache

import (
	"context"
	"sync"
	"time"
)

type Fetcher[T any] func(ctx context.Context) (T, error)

type TTL[T any] struct {
	fetcher Fetcher[T]
	ttl     time.Duration

	mu          sync.RWMutex
	value       T
	lastFetched time.Time
}

func NewTTL[T any](fetcher Fetcher[T], ttl time.Duration) *TTL[T] {
	return &TTL[T]{
		fetcher: fetcher,
		ttl:     ttl,
	}
}

func (c *TTL[T]) Get(ctx context.Context) (T, error) {
	c.mu.RLock()
	if c.isValid() {
		value := c.value
		c.mu.RUnlock()
		return value, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isValid() {
		return c.value, nil
	}

	value, err := c.fetcher(ctx)
	if err != nil {
		var zero T
		return zero, err
	}

	c.value = value
	c.lastFetched = time.Now()

	return value, nil
}

func (c *TTL[T]) isValid() bool {
	return !c.lastFetched.IsZero() && time.Since(c.lastFetched) < c.ttl
}

func (c *TTL[T]) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastFetched = time.Time{}
}
