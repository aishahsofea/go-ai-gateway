package gateway

import (
	"fmt"
	"strings"
)

// Gateway route configuration
type Route struct {
	Pattern     string `json:"pattern"`
	Target      string `json:"target"`
	StripPrefix string `json:"stripPrefix"`
}

type GatewayConfig struct {
	Routes []Route `json:"routes"`
}

// Create a basic gateway configuration
func DefaultConfig() *GatewayConfig {
	return &GatewayConfig{
		Routes: []Route{
			{
				Pattern: "/api/users/*",
				Target:  "http://host.docker.internal:8001", // TODO: create mock service
			},
			{
				Pattern: "/api/products/*",
				Target:  "http://host.docker.internal:8002",
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
