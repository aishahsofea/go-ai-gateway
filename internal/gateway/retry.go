package gateway

import (
	"context"
	"errors"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"
)

type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

func (r *RetryConfig) ExecuteWithRetry(ctx context.Context, operation func() (int, error)) error {
	var lastErr error

	for attempt := 0; attempt < r.MaxAttempts; attempt++ {

		statusCode, err := operation()
		if err == nil && !isRetryableStatusCode(statusCode) { // success
			return nil
		}

		// check if error is retryable
		shouldRetry := false
		if err != nil && isRetryableError(err) {
			log.Printf("⚠️ Retryable error encountered: %v", err)
			shouldRetry = true
		}

		if err != nil && isRetryableStatusCode(statusCode) {
			log.Printf("⚠️ Retryable server error: status=%d, error=%v", statusCode, err)
			shouldRetry = true
			lastErr = err
		}

		if attempt == r.MaxAttempts-1 {
			break
		}

		if !shouldRetry {
			break
		}

		delay := r.calculateDelay(attempt)
		time.Sleep(delay)
	}

	return lastErr
}

func (r *RetryConfig) calculateDelay(attempt int) time.Duration {
	delay := time.Duration(float64(r.InitialDelay) * math.Pow(r.Multiplier, float64(attempt)))

	if delay > r.MaxDelay {
		delay = r.MaxDelay
	}

	if r.Jitter {
		jitterRange := float64(delay) * 0.25
		jitter := (rand.Float64()*2 - 1) * jitterRange // random value between -25% and +25%
		delay = time.Duration(float64(delay) + jitter)
	}

	return delay
}

func isRetryableStatusCode(statusCode int) bool {
	return statusCode >= 500
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	errStr := err.Error()

	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"no such host",
		"network is unreachable",
		"i/o timeout",
		"EOF",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}

	return false
}
