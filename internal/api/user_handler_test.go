package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aishahsofea/go-ai-gateway/internal/db"
	"github.com/aishahsofea/go-ai-gateway/internal/testutils"
)

func createTestLogger() *log.Logger {
	return log.New(io.Discard, "", 0) // Discards all log output for clean tests
}

func TestUserRegistration(t *testing.T) {
	// setup test DB
	testDB := testutils.SetupTestDB(t)
	defer testutils.CleanupTestDB(t, testDB)

	// create user repository and auth handler
	userRepo := db.NewUserRepository(testDB)
	authHandler := NewAuthHandler(userRepo, nil)

	requestBody := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}

	// convert to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("could not marshal request body: %v", err)
	}

	// create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// create response recorder
	w := httptest.NewRecorder()

	// call the handler
	authHandler.RegisterUser(w, req)

	// check response
	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestUserRegistration_InvalidEmail(t *testing.T) {
	testDB := testutils.SetupTestDB(t)
	defer testutils.CleanupTestDB(t, testDB)

	userRepo := db.NewUserRepository(testDB)
	authHandler := NewAuthHandler(userRepo, createTestLogger())

	requestBody := map[string]string{
		"email":    "invalid-email",
		"password": "password123",
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("could not marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	authHandler.RegisterUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserRegistration_DuplicateEmail(t *testing.T) {
	testDB := testutils.SetupTestDB(t)
	defer testutils.CleanupTestDB(t, testDB)

	userRepo := db.NewUserRepository(testDB)
	authHandler := NewAuthHandler(userRepo, createTestLogger())

	testutils.CreateTestUserWithCredentials(t, userRepo, "duplicate@example.com", "password123")

	requestBody := map[string]string{
		"email":    "duplicate@example.com",
		"password": "differentpassword",
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("could not marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	authHandler.RegisterUser(w, req)

	if w.Code == http.StatusCreated {
		t.Error("expected error for duplicate email, but got success")
	}
}

func TestUserLogin(t *testing.T) {}

func TestUserLogin_InvalidCredentials(t *testing.T) {}
