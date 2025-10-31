package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServiceRegistry(t *testing.T) {
	registry := NewServiceRegistry()

	t.Run("RegisterService", func(t *testing.T) {
		instance := &ServiceInstance{
			ID:       "test-service-1",
			URL:      "http://localhost:8001",
			Route:    "/api/users/*",
			Health:   "healthy",
			Metadata: map[string]string{"version": "1.0.0"},
		}

		err := registry.RegisterService(instance)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify service was registered
		services := registry.GetServices("/api/users/*")
		if len(services) != 1 {
			t.Errorf("Expected 1 service, got %d", len(services))
		}

		if services[0].ID != "test-service-1" {
			t.Errorf("Expected service ID 'test-service-1', got %s", services[0].ID)
		}
	})

	t.Run("DeregisterService", func(t *testing.T) {
		// Register first
		instance := &ServiceInstance{
			ID:     "test-service-2",
			URL:    "http://localhost:8002",
			Route:  "/api/products/*",
			Health: "healthy",
		}
		registry.RegisterService(instance)

		// Deregister
		err := registry.DeregisterService("/api/products/*", "test-service-2")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify service was removed
		services := registry.GetServices("/api/products/*")
		if len(services) != 0 {
			t.Errorf("Expected 0 services, got %d", len(services))
		}
	})

	t.Run("GetAllRoutes", func(t *testing.T) {
		// Clear registry
		registry = NewServiceRegistry()

		// Register services on different routes
		registry.RegisterService(&ServiceInstance{
			ID: "svc1", URL: "http://localhost:8001", Route: "/api/users/*", Health: "healthy",
		})
		registry.RegisterService(&ServiceInstance{
			ID: "svc2", URL: "http://localhost:8002", Route: "/api/products/*", Health: "healthy",
		})

		routes := registry.GetAllRoutes()
		if len(routes) != 2 {
			t.Errorf("Expected 2 routes, got %d", len(routes))
		}
	})
}

func TestRegistryHandlers(t *testing.T) {
	registry := NewServiceRegistry()

	t.Run("RegisterHandler", func(t *testing.T) {
		// Create test request
		body := `{
                        "id": "test-svc-1",
                        "url": "http://localhost:8001", 
                        "route": "/api/users/*",
                        "health": "healthy"
                }`

		req := httptest.NewRequest("POST", "/registry/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Call handler
		registry.RegisterHandler(w, req)

		// Assert response
		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		// Verify service was actually registered
		services := registry.GetServices("/api/users/*")
		if len(services) != 1 {
			t.Errorf("Expected 1 service registered, got %d", len(services))
		}
	})

	t.Run("GetAllServicesHandler", func(t *testing.T) {
		// Register a test service first
		registry.RegisterService(&ServiceInstance{
			ID: "test-svc", URL: "http://localhost:8001", Route: "/api/test/*", Health: "healthy",
		})

		req := httptest.NewRequest("GET", "/registry/services", nil)
		w := httptest.NewRecorder()

		registry.GetAllServicesHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Parse response and verify structure
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["data"] == nil {
			t.Error("Expected 'data' field in response")
		}
	})
}

func TestHealthChecker(t *testing.T) {
	registry := NewServiceRegistry()
	config := HealthCheckConfig{
		Interval:       100 * time.Millisecond, // Fast for testing
		Timeout:        1 * time.Second,
		FailureLimit:   2, // Lower for faster testing
		HealthEndpoint: "/health",
	}
	healthChecker := NewHealthChecker(registry, config)

	t.Run("HealthyService", func(t *testing.T) {
		// Create a test HTTP server that returns 200
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Register service with test server URL
		instance := &ServiceInstance{
			ID:     "test-healthy",
			URL:    server.URL,
			Route:  "/api/test/*",
			Health: "unknown",
		}
		registry.RegisterService(instance)

		// Check service health
		isHealthy := healthChecker.checkService(instance)
		if !isHealthy {
			t.Error("Expected service to be healthy")
		}

		// Update health and verify
		healthChecker.updateServiceHealth(instance, isHealthy)
		services := registry.GetServices("/api/test/*")
		if services[0].Health != "healthy" {
			t.Errorf("Expected health 'healthy', got '%s'", services[0].Health)
		}
	})

	t.Run("UnhealthyService", func(t *testing.T) {
		// Register service with non-existent URL
		instance := &ServiceInstance{
			ID:     "test-unhealthy",
			URL:    "http://localhost:99999", // Non-existent port
			Route:  "/api/broken/*",
			Health: "healthy",
		}
		registry.RegisterService(instance)

		// Simulate multiple failures
		for i := 0; i < config.FailureLimit; i++ {
			isHealthy := healthChecker.checkService(instance)
			healthChecker.updateServiceHealth(instance, isHealthy)
		}

		// Verify service marked as unhealthy
		services := registry.GetServices("/api/broken/*")
		if services[0].Health != "unhealthy" {
			t.Errorf("Expected health 'unhealthy', got '%s'", services[0].Health)
		}
	})
}
