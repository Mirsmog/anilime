package ratelimit

import (
	"context"
	"time"
)

type Limiter struct {
	t *time.Ticker
}

// NewRPS creates a simple token ticker allowing up to rps operations per second.
func NewRPS(rps int) *Limiter {
	if rps <= 0 {
		rps = 1
	}
	interval := time.Second / time.Duration(rps)
	if interval <= 0 {
		interval = time.Second
	}
	return &Limiter{t: time.NewTicker(interval)}
}

func (l *Limiter) Stop() {
	if l != nil && l.t != nil {
		l.t.Stop()
	}
}

func (l *Limiter) Wait(ctx context.Context) error {
	if l == nil || l.t == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-l.t.C:
		return nil
	}
}
