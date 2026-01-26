package router

import (
	"context"
	"time"
)

type TimeoutConfig struct {
	Index time.Duration
	LSP   time.Duration
	Regex time.Duration
	Total time.Duration
}

func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Index: 50 * time.Millisecond,
		LSP:   2 * time.Second,
		Regex: 5 * time.Second,
		Total: 10 * time.Second,
	}
}

func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}
