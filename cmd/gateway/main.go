package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aishahsofea/go-ai-gateway/internal/db"
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

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthCheck)

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
