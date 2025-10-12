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
	wg.Add(2)

	// start User Service
	go func() {
		defer wg.Done()
		server := &http.Server{
			Addr:    ":8001",
			Handler: testutils.MockUserServiceHandler(),
		}

		go func() {
			<-ctx.Done()
			server.Shutdown(context.Background())
		}()

		log.Println("Mock User Service starting on :8001")
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Printf("User Service error: %v", err)
		}
	}()

	// start Product Service
	go func() {
		defer wg.Done()
		server := &http.Server{
			Addr:    ":8002",
			Handler: testutils.MockProductServiceHandler(),
		}

		go func() {
			<-ctx.Done()
			server.Shutdown(context.Background())
		}()

		log.Println("Mock Product Service starting on :8002")
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Printf("Product Service error: %v", err)
		}
	}()

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
