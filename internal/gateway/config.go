package gateway

import (
	"fmt"
	"strings"
)

type LoadBalancerStrategy string

const (
	RoundRobin       LoadBalancerStrategy = "round_robin"
	LeastConnections LoadBalancerStrategy = "least_connections"
	Random           LoadBalancerStrategy = "random"
)

type Backend struct {
	URL            string          `json:"url"`
	Healthy        bool            `json:"healthy"`
	Weight         int             `json:"weight"`
	CircuitBreaker *CircuitBreaker `json:"_"`
}

type Route struct {
	Pattern      string               `json:"pattern"`
	Target       string               `json:"target"`
	Backends     []Backend            `json:"backends"`
	LoadBalancer LoadBalancerStrategy `json:"load_balancer"`
	StripPrefix  string               `json:"strip_prefix"`
}

type GatewayConfig struct {
	Routes []Route `json:"routes"`
}

func DefaultConfig() *GatewayConfig {
	cbConfig := DefaultCircuitBreakerConfig()

	return &GatewayConfig{
		Routes: []Route{
			{
				Pattern: "/api/users/*",
				Backends: []Backend{
					{
						URL:            "http://host.docker.internal:8001",
						Healthy:        true,
						Weight:         1,
						CircuitBreaker: NewCircuitBreaker(cbConfig),
					},
					{
						URL:            "http://host.docker.internal:8011",
						Healthy:        true,
						Weight:         1,
						CircuitBreaker: NewCircuitBreaker(cbConfig),
					},
					{
						URL:            "http://host.docker.internal:8022",
						Healthy:        true,
						Weight:         1,
						CircuitBreaker: NewCircuitBreaker(cbConfig),
					},
				},
				LoadBalancer: RoundRobin,
			},
			{
				Pattern: "/api/products/*",
				Backends: []Backend{
					{
						URL:            "http://host.docker.internal:8002",
						Healthy:        true,
						Weight:         1,
						CircuitBreaker: NewCircuitBreaker(cbConfig),
					},
					{
						URL:            "http://host.docker.internal:8012",
						Healthy:        true,
						Weight:         1,
						CircuitBreaker: NewCircuitBreaker(cbConfig),
					},
				},
				LoadBalancer: RoundRobin,
			},
			// Existing routes stay on this service
			{
				Pattern: "/users",
				Target:  "http://localhost:8080",
			},
			{
				Pattern: "/auth/*",
				Target:  "http://localhost:8080",
			},
		},
	}
}

func LocalhostConfig() *GatewayConfig {
	config := DefaultConfig()

	// Replace host.docker.internal with localhost
	for i := range config.Routes {
		for j := range config.Routes[i].Backends {
			url := config.Routes[i].Backends[j].URL
			config.Routes[i].Backends[j].URL = strings.Replace(url, "host.docker.internal", "localhost", 1)
		}
	}

	return config
}

// Finds matching route for a given path
func (gc *GatewayConfig) MatchRoute(path string) (*Route, error) {
	for _, route := range gc.Routes {
		if matchesPattern(route.Pattern, path) {
			return &route, nil
		}
	}
	return nil, fmt.Errorf("no matching route for path: %s", path)
}

func matchesPattern(pattern, path string) bool {
	// TODO: improve pattern matching (e.g., support more complex patterns)
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix)
	}
	return pattern == path
}

func (r *Route) GetBackends() []Backend {
	if len(r.Backends) > 0 {
		return r.Backends
	}

	// fallback to single target as backend (backward compatibility)
	if r.Target != "" {
		return []Backend{
			{URL: r.Target, Healthy: true, Weight: 1},
		}
	}

	return []Backend{}
}
