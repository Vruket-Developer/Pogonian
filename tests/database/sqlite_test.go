package database_test

import (
	"path/filepath"
	"testing"

	"pogonian/internal/database"
)

func TestDatabase_SQLite(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	// 1. Run migrations
	err = db.Migrate(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			role TEXT NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// 2. Insert row
	res, err := db.Exec("INSERT INTO users (email, role) VALUES (?, ?)", "alice@example.com", "admin")
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	id, _ := res.LastInsertId()
	if id != 1 {
		t.Errorf("Expected last insert id 1, got %d", id)
	}

	// 3. Query row
	var email, role string
	err = db.QueryRow("SELECT email, role FROM users WHERE id = ?", id).Scan(&email, &role)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if email != "alice@example.com" || role != "admin" {
		t.Errorf("Unexpected user data: email=%s, role=%s", email, role)
	}
}
