package gateway

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
)

type LoadBalancer interface {
	SelectBackend(backends []Backend) (*Backend, error)
	String() string
}

type RoundRobinBalancer struct {
	counter uint64
}

func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{}
}

func (rb *RoundRobinBalancer) SelectBackend(backends []Backend) (*Backend, error) {
	healthy := getHealthyBackends(backends)
	if len(healthy) == 0 {
		return nil, fmt.Errorf("no healthy backends available")
	}

	idx := atomic.AddUint64(&rb.counter, 1) % uint64(len(healthy))
	return &healthy[idx], nil
}

func (rb *RoundRobinBalancer) String() string {
	return "RoundRobin"
}

type RandomBalancer struct{}

func NewRandomBalancer() *RandomBalancer {
	return &RandomBalancer{}
}

func (r *RandomBalancer) SelectBackend(backends []Backend) (*Backend, error) {
	healthy := getHealthyBackends(backends)
	if len(healthy) == 0 {
		return nil, fmt.Errorf("no healthy backends available")
	}

	idx := rand.Intn(len(healthy))
	return &healthy[idx], nil
}

func (r *RandomBalancer) String() string {
	return "Random"
}

type LeastConnectionsBalancer struct {
	connections map[string]int64
	mutex       sync.RWMutex
}

func NewLeastConnectionsBalancer() *LeastConnectionsBalancer {
	return &LeastConnectionsBalancer{
		connections: make(map[string]int64),
	}
}

func (lc *LeastConnectionsBalancer) SelectBackend(backends []Backend) (*Backend, error) {
	healthy := getHealthyBackends(backends)
	if len(healthy) == 0 {
		return nil, fmt.Errorf("no healthy backends available")
	}

	lc.mutex.RLock()
	defer lc.mutex.RUnlock()

	var selected *Backend
	minConnections := int64(-1)

	for i := range healthy {
		backend := &healthy[i]
		connections := lc.connections[backend.URL]

		if minConnections == -1 || connections < minConnections {
			minConnections = connections
			selected = backend
		}
	}

	return selected, nil
}

func (lc *LeastConnectionsBalancer) String() string {
	return "LeastConnections"
}

func (lc *LeastConnectionsBalancer) IncrementConnections(url string) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	lc.connections[url]++
}

func (lc *LeastConnectionsBalancer) DecrementConnections(url string) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	if lc.connections[url] > 0 {
		lc.connections[url]--
	}
}

func getHealthyBackends(backends []Backend) []Backend {
	var healthy []Backend
	for _, backend := range backends {
		canRequest := backend.CircuitBreaker.CanRequest()
		log.Printf("🔍 Backend %s: healthy=%v, canRequest=%v, state=%v",
			backend.URL, backend.Healthy, canRequest, backend.CircuitBreaker.GetState())
		if backend.Healthy && canRequest {
			healthy = append(healthy, backend)
		}
	}
	return healthy
}

func NewLoadBalancer(strategy LoadBalancerStrategy) LoadBalancer {
	switch strategy {
	case RoundRobin:
		return NewRoundRobinBalancer()
	case Random:
		return NewRandomBalancer()
	case LeastConnections:
		return NewLeastConnectionsBalancer()
	default:
		return NewRoundRobinBalancer() // Default to RoundRobin
	}
}
