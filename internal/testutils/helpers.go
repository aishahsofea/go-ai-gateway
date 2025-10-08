package testutils

import (
	"context"
	"testing"
	"time"

	"github.com/aishahsofea/go-ai-gateway/internal/db"
	"github.com/aishahsofea/go-ai-gateway/internal/models"
	"github.com/google/uuid"
)

func CreateTestUser(t *testing.T, userRepo *db.UserRepository) *models.User {
	t.Helper()

	// Create user with default email and password
	user := &models.User{
		ID:        uuid.New(),
		Email:     "testuser@example.com",
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := user.Password.Set("password123")
	if err != nil {
		t.Fatalf("failed to set password: %v", err)
	}

	err = userRepo.CreateUser(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to create test user in database: %v", err)
	}

	return user
}

func CreateTestUserWithCredentials(t *testing.T, userRepo *db.UserRepository, email, password string) *models.User {
	t.Helper()

	// Create user with custom credentials
	user := &models.User{
		ID:        uuid.New(),
		Email:     email,
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := user.Password.Set(password)
	if err != nil {
		t.Fatalf("failed to set password: %v", err)
	}

	err = userRepo.CreateUser(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to create test user in database: %v", err)
	}

	return user
}

func CreateTestUserWithRole(t *testing.T, userRepo *db.UserRepository, email, role string) *models.User {
	t.Helper()

	// Create user with custom role (e.g., "admin", "user")
	user := &models.User{
		Email: email,
		Role:  role,
	}

	err := user.Password.Set("password123")
	if err != nil {
		t.Fatalf("failed to set password: %v", err)
	}

	err = userRepo.CreateUser(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to create test user in database: %v", err)
	}

	return user
}
