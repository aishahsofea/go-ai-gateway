package gateway

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type BulkheadConfig struct {
	MaxConcurrentRequests int
	QueueSize             int
	QueueTimeout          time.Duration
}

func DefaultBulkheadConfig() BulkheadConfig {
	return BulkheadConfig{
		MaxConcurrentRequests: 10,
		QueueSize:             5,
		QueueTimeout:          2 * time.Second,
	}
}

type Request struct {
	ctx      context.Context
	response chan error
}

type ServiceBulkhead struct {
	config      BulkheadConfig
	semaphore   chan struct{}
	queue       chan *Request
	activeCount int64
	queuedCount int64
	mutex       sync.RWMutex
}

func NewServiceBulkhead(config BulkheadConfig) *ServiceBulkhead {
	return &ServiceBulkhead{
		config:    config,
		semaphore: make(chan struct{}, config.MaxConcurrentRequests),
		queue:     make(chan *Request, config.QueueSize),
	}
}

func (sb *ServiceBulkhead) TryAcquire(ctx context.Context) error {
	select {
	case sb.semaphore <- struct{}{}:
		// Acquired a slot in semaphore channel
		sb.mutex.Lock()
		sb.activeCount++
		sb.mutex.Unlock()
		return nil
	default:
		return sb.tryQueue(ctx)
	}
}

func (sb *ServiceBulkhead) Release() {
	<-sb.semaphore
	sb.mutex.Lock()
	sb.activeCount--
	sb.mutex.Unlock()

	// Process queued requests if any
	select {
	case req := <-sb.queue:
		sb.semaphore <- struct{}{}
		sb.mutex.Lock()
		sb.activeCount++
		sb.queuedCount--
		sb.mutex.Unlock()
		req.response <- nil
	default:
		// no queued requests
	}
}

func (sb *ServiceBulkhead) tryQueue(ctx context.Context) error {
	req := &Request{
		ctx:      ctx,
		response: make(chan error, 1),
	}

	select {
	case sb.queue <- req:
		sb.mutex.Lock()
		sb.queuedCount++
		sb.mutex.Unlock()

		// Wait for either being granted a slot or timeout/cancellation
		select {
		case result := <-req.response:
			return result
		case <-time.After(sb.config.QueueTimeout):
			return fmt.Errorf("bulkhead queue timeout")
		case <-ctx.Done():
			return ctx.Err()
		}

	default:
		return fmt.Errorf("bulkhead queue is full")
	}
}

func (sb *ServiceBulkhead) GetStats() map[string]any {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	return map[string]any{
		"max_concurrent": sb.config.MaxConcurrentRequests,
		"active_count":   sb.activeCount,
		"queued_count":   sb.queuedCount,
		"queue_size":     sb.config.QueueSize,
	}
}
