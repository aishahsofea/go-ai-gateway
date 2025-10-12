package testutils

import (
	"encoding/json"
	"net/http"
	"strings"
)

func MockUserServiceHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/users/", func(w http.ResponseWriter, r *http.Request) {
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

	return mux
}

func MockProductServiceHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/products/", func(w http.ResponseWriter, r *http.Request) {
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
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	return mux
}
