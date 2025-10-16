package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/aishahsofea/go-ai-gateway/internal/testutils"
)

func main() {
	log.Println("🚀 Starting Gateway Test Environment")
	log.Println("----------------------------------")

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	// start mock services
	wg.Add(5) // 3 user services, 2 product services

	// start User Services
	go startMockService(&wg, ctx, "8001", "user-service-1", testutils.MockUserServiceHandler())
	go startMockService(&wg, ctx, "8011", "user-service-2", testutils.MockUserServiceHandler())
	go startMockService(&wg, ctx, "8022", "user-service-3", testutils.MockUserServiceHandler())

	// start Product Services
	go startMockService(&wg, ctx, "8002", "product-service-1", testutils.MockProductServiceHandler())
	go startMockService(&wg, ctx, "8012", "product-service-2", testutils.MockProductServiceHandler())

	// wait for services to start
	log.Println("⏳ Waiting for services to start...")
	time.Sleep(3 * time.Second)

	log.Println("Running Gateway Tests...")
	cmd := exec.Command("bash", "./scripts/test-gateway.sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("⚠️ Test script error: %v", err)
	}

	log.Println("✅ Tests completed!")
	log.Println("Mock services still running. Press Ctrl+C to stop")

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	log.Println("🛑 Shutting down test environment...")
	cancel()
	wg.Wait()
	log.Println("Test environment stopped. Goodbye!")
}

func startMockService(wg *sync.WaitGroup, ctx context.Context, port, name string, handler http.Handler) {
	defer wg.Done()

	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	log.Printf("Mock %s starting on :%s", name, port)
	err := server.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Printf("%s error: %v", name, err)
	}
}
