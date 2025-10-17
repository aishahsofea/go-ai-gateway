package testutils

import (
	"encoding/json"
	"net/http"
	"strings"
)

var serviceShouldFail = map[string]bool{
	"8001": false,
	"8011": false,
	"8022": false,
	"8002": false,
	"8012": false,
}

func MockUserServiceHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/users/", func(w http.ResponseWriter, r *http.Request) {
		port := "8001" // TODO: figure out port detection
		if serviceShouldFail[port] {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "Simulated service failure",
				"service": "user-service",
				"port":    port,
			})
			return
		}

		userID := strings.TrimPrefix(r.URL.Path, "/api/users/")
		response := map[string]interface{}{
			"service": "user-service",
			"method":  r.Method,
			"path":    r.URL.Path,
			"user_id": userID,
			"message": "Response from User Service",
			"headers": map[string]string{
				"X-Forwarded-By":    r.Header.Get("X-Forwarded-By"),
				"X-Gateway-Version": r.Header.Get("X-Gateway-Version"),
				"X-Backend-URL":     r.Header.Get("X-Backend-URL"),
				"X-Load-Balancer":   r.Header.Get("X-Load-Balancer"),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"service": "user-service",
			"error":   "Route not found in user service",
			"path":    r.URL.Path,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/admin/fail", func(w http.ResponseWriter, r *http.Request) {
		port := "8001" // TODO: figure out port detection
		serviceShouldFail[port] = true

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "service will now fail",
			"port":    port,
			"service": "user-service",
		})
	})

	mux.HandleFunc("/admin/recover", func(w http.ResponseWriter, r *http.Request) {
		port := "8001" // TODO: figure out port detection
		serviceShouldFail[port] = false

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "service recovered",
			"port":    port,
			"service": "user-service",
		})
	})

	mux.HandleFunc("/admin/status", func(w http.ResponseWriter, r *http.Request) {
		port := "8001" // TODO: figure out port detection

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"port":    port,
			"service": "user-service",
			"failing": serviceShouldFail[port],
		})
	})

	return mux
}

func MockProductServiceHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/products/", func(w http.ResponseWriter, r *http.Request) {

		port := "8002" // TODO: figure out port detection
		if serviceShouldFail[port] {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "Simulated service failure",
				"service": "product-service",
				"port":    port,
			})
			return
		}

		productID := strings.TrimPrefix(r.URL.Path, "/api/products/")
		response := map[string]interface{}{
			"service":    "product-service",
			"method":     r.Method,
			"path":       r.URL.Path,
			"product_id": productID,
			"message":    "Response from Product Service",
			"headers": map[string]string{
				"X-Forwarded-By":    r.Header.Get("X-Forwarded-By"),
				"X-Gateway-Version": r.Header.Get("X-Gateway-Version"),
				"X-Backend-URL":     r.Header.Get("X-Backend-URL"),
				"X-Load-Balancer":   r.Header.Get("X-Load-Balancer"),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/admin/fail", func(w http.ResponseWriter, r *http.Request) {
		port := "8002" // TODO: figure out port detection
		serviceShouldFail[port] = true

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "service will now fail",
			"port":    port,
			"service": "product-service",
		})
	})

	mux.HandleFunc("/admin/recover", func(w http.ResponseWriter, r *http.Request) {
		port := "8002" // TODO: figure out port detection
		serviceShouldFail[port] = false

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "service recovered",
			"port":    port,
			"service": "product-service",
		})
	})

	mux.HandleFunc("/admin/status", func(w http.ResponseWriter, r *http.Request) {
		port := "8002" // TODO: figure out port detection

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"port":    port,
			"service": "product-service",
			"failing": serviceShouldFail[port],
		})
	})

	return mux
}
