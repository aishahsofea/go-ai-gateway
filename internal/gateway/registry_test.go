package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
