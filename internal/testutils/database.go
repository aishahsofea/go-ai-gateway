package testutils

import (
	"context"
	"testing"

	"github.com/aishahsofea/go-ai-gateway/internal/db"
	"github.com/aishahsofea/go-ai-gateway/migrations"
)

const TestDatabaseURL = "postgres://gateway_user:gateway_password@localhost:5433/gateway?sslmode=disable"

func SetupTestDB(t *testing.T) *db.DB {
	t.Helper()

	testDB, err := db.NewDB(context.Background(), TestDatabaseURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	err = db.MigrateFS(testDB, migrations.FS, ".")
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return testDB
}

func CleanupTestDB(t *testing.T, testDB *db.DB) {
	t.Helper()

	// Clean up test data
	_, err := testDB.Exec(context.Background(), "TRUNCATE TABLE users RESTART IDENTITY CASCADE")
	if err != nil {
		t.Logf("warning: failed to clean up test database: %v", err)
	}

	testDB.Close()
}
