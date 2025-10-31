package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aishahsofea/go-ai-gateway/internal/api"
	"github.com/aishahsofea/go-ai-gateway/internal/db"
	"github.com/aishahsofea/go-ai-gateway/internal/gateway"
	"github.com/aishahsofea/go-ai-gateway/internal/middleware"
	"github.com/aishahsofea/go-ai-gateway/migrations"
)

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	newDB, err := db.NewDB(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("could not create new db: %v", err)
	}

	err = db.MigrateFS(newDB, migrations.FS, ".")
	if err != nil {
		log.Fatalf("could not run migrations: %v", err)
		panic(err)
	}

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	userRepo := db.NewUserRepository(newDB)

	authHandler := api.NewAuthHandler(userRepo, logger)

	requestTimeout := flag.Duration("request-timeout", 30*time.Second, "Overall request timeout")
	backendTimeout := flag.Duration("backend-timeout", 5*time.Second, "Backend request timeout")
	connectTimeout := flag.Duration("connect-timeout", 2*time.Second, "Connection timeout")
	useLocalhost := flag.Bool("use-localhost", false, "Use localhost instead of host.docker.internal for backends")
	flag.Parse()

	timeoutConfig := gateway.TimeoutConfig{
		RequestTimeout: *requestTimeout,
		BackendTimeout: *backendTimeout,
		ConnectTimeout: *connectTimeout,
	}

	gatewayConfig := gateway.DefaultConfig()

	if *useLocalhost {
		gatewayConfig = gateway.LocalhostConfig()
	}

	proxy := gateway.NewProxy(gatewayConfig, timeoutConfig)
	registry := gateway.NewServiceRegistry()
	healthConfig := gateway.DefaultHealthCheckConfig()
	healthChecker := gateway.NewHealthChecker(registry, healthConfig)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthCheck)
	mux.HandleFunc("POST /users", authHandler.RegisterUser)
	mux.HandleFunc("POST /users/login", authHandler.Login)

	// Registry endpoints
	mux.HandleFunc("POST /registry/register", registry.RegisterHandler)
	mux.HandleFunc("DELETE /registry/deregister/{id}", registry.DeregisterHandler)
	mux.HandleFunc("GET /registry/services", registry.GetAllServicesHandler)
	mux.HandleFunc("GET /registry/services/{route}", registry.GetServicesByRouteHandler)

	mux.Handle("GET /protected", middleware.Authenticate(http.HandlerFunc(protectedHandler)))
	mux.Handle("/", proxy)

	go healthChecker.Start(context.Background())

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go startServer(server, port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give server 30 seconds to complete ongoing requests
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)

	if err != nil {
		log.Fatalf("Could not gracefully shut down the server: %v", err)
	}

	log.Println("Server gracefully stopped")

}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy", "service": "go-ai-gateway"}`))
}

func startServer(server *http.Server, port string) {
	log.Printf("Starting server on port %s", port)

	err := server.ListenAndServe()

	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not start server: %s", err)
	}
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Access granted!",
		"user_id": user.ID,
	})
}
