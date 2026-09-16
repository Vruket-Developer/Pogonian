package database_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	database "pogonian/pkg/dataruntime"
	_ "modernc.org/sqlite"
)

func TestDatabase_MigrationsUpAndDown(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pogn_db_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	driver := &database.SQLiteDriver{Path: dbPath}
	db, err := driver.Open()
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	migDir := filepath.Join(tempDir, "migrations")
	_ = os.MkdirAll(migDir, 0755)

	// Create 2 migration pairs
	up1, down1, err := database.GenerateMigration(migDir, "create_users")
	if err != nil {
		t.Fatalf("GenerateMigration 1 failed: %v", err)
	}
	_ = os.WriteFile(up1, []byte("CREATE TABLE test_users (id INTEGER PRIMARY KEY, email TEXT NOT NULL);"), 0644)
	_ = os.WriteFile(down1, []byte("DROP TABLE IF EXISTS test_users;"), 0644)

	up2, down2, err := database.GenerateMigration(migDir, "create_orders")
	if err != nil {
		t.Fatalf("GenerateMigration 2 failed: %v", err)
	}
	_ = os.WriteFile(up2, []byte("CREATE TABLE test_orders (id INTEGER PRIMARY KEY, user_id INTEGER, amount REAL);"), 0644)
	_ = os.WriteFile(down2, []byte("DROP TABLE IF EXISTS test_orders;"), 0644)

	engine := database.NewMigrationEngine(db, driver)

	// 1. Initial Status (0 applied)
	statuses, err := engine.Status(migDir)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("Expected 2 discovered migrations, got %d", len(statuses))
	}
	if statuses[0].Applied || statuses[1].Applied {
		t.Errorf("Expected all migrations pending initially")
	}

	// 2. Migrate Up
	applied, err := engine.MigrateUp(migDir)
	if err != nil {
		t.Fatalf("MigrateUp failed: %v", err)
	}
	if len(applied) != 2 {
		t.Errorf("Expected 2 migrations applied, got %d", len(applied))
	}

	// Verify tables created in SQLite
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_users'").Scan(&tableName)
	if err != nil || tableName != "test_users" {
		t.Errorf("Expected test_users table to exist in database")
	}

	// 3. Rollback 1 step
	rolledBack, err := engine.Rollback(migDir, 1)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}
	if len(rolledBack) != 1 || rolledBack[0].Version != 2 {
		t.Errorf("Expected migration 2 to be rolled back, got %v", rolledBack)
	}

	// Verify test_orders dropped but test_users remains
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_orders'").Scan(&tableName)
	if err != sql.ErrNoRows {
		t.Errorf("Expected test_orders to be dropped after rollback")
	}

	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_users'").Scan(&tableName)
	if err != nil {
		t.Errorf("Expected test_users to remain after 1-step rollback")
	}
}
