package gateway

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type HealthCheckConfig struct {
	Interval       time.Duration
	Timeout        time.Duration
	FailureLimit   int
	HealthEndpoint string
}

func DefaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{
		Interval:       30 * time.Second,
		Timeout:        5 * time.Second,
		FailureLimit:   3,
		HealthEndpoint: "/health",
	}
}

type HealthChecker struct {
	registry *ServiceRegistry
	config   HealthCheckConfig
	client   *http.Client
	mutex    sync.RWMutex
	failures map[string]int // serviceID -> failure count
}

func NewHealthChecker(registry *ServiceRegistry, config HealthCheckConfig) *HealthChecker {
	return &HealthChecker{
		registry: registry,
		config:   config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		failures: make(map[string]int),
	}
}

func (hc *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(hc.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.mutex.RLock()
			for _, instances := range hc.registry.services {
				for _, instance := range instances {
					// go hc.checkService(instance)
					go func(inst *ServiceInstance) {
						isHealthy := hc.checkService(inst)
						hc.updateServiceHealth(inst, isHealthy)
					}(instance)
				}
			}
			hc.mutex.RUnlock()
		}
	}
}

func (hc *HealthChecker) checkService(instance *ServiceInstance) bool {
	req, err := http.NewRequest("GET", instance.URL+hc.config.HealthEndpoint, nil)
	if err != nil {
		return false
	}

	resp, err := hc.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	defer resp.Body.Close()

	return true
}

func (hc *HealthChecker) updateServiceHealth(instance *ServiceInstance, isHealthy bool) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	if isHealthy {
		delete(hc.failures, instance.ID)
		hc.registry.UpdateServiceHealth(instance.Route, instance.ID, "healthy")
	} else {
		hc.failures[instance.ID]++
		if hc.failures[instance.ID] >= hc.config.FailureLimit {
			hc.registry.UpdateServiceHealth(instance.Route, instance.ID, "unhealthy")
		}
	}
}
