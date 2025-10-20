package gateway

import (
	"context"
	"time"
)

type TimeoutConfig struct {
	RequestTimeout time.Duration // overall request timeout (client -> gateway -> backend -> client)
	BackendTimeout time.Duration // per-backend timeout (gateway -> backend)
	ConnectTimeout time.Duration // TCP connection timeout
}

func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		RequestTimeout: 30 * time.Second,
		BackendTimeout: 5 * time.Second,
		ConnectTimeout: 2 * time.Second,
	}
}

func (tc *TimeoutConfig) WithBackendTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, tc.BackendTimeout)
}

func (tc *TimeoutConfig) WithRequestTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, tc.RequestTimeout)
}

func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	return err == context.DeadlineExceeded
}
