package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps an active SQLite database connection with helper utilities.
type DB struct {
	*sql.DB
	Path string
}

// Open initializes and configures a SQLite connection with production pragmas.
func Open(dbPath string) (*DB, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)", dbPath)
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Set connection pool limits
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite database: %w", err)
	}

	return &DB{
		DB:   conn,
		Path: dbPath,
	}, nil
}

// Migrate executes an array of SQL migration scripts in order.
func (db *DB) Migrate(queries ...string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, q := range queries {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("migration failed for statement [%s]: %w", q, err)
		}
	}

	return tx.Commit()
}

// IsSafeIdentifier checks if a table name is purely alphanumeric.
func IsSafeIdentifier(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// GenericInsert safely inserts title and content into a dynamic entity table.
func (db *DB) GenericInsert(table, title, content string) error {
	if !IsSafeIdentifier(table) {
		return fmt.Errorf("invalid table identifier: %s", table)
	}

	createStmt := "CREATE TABLE IF NOT EXISTS " + table + ` (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		content TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(createStmt); err != nil {
		return err
	}

	insertStmt := "INSERT INTO " + table + " (title, content, created_at) VALUES (?, ?, ?)"
	_, err := db.Exec(insertStmt, title, content, time.Now().UTC().Format(time.RFC3339))
	return err
}

// GenericRecord represents a standard dynamic row.
type GenericRecord struct {
	ID      int64
	Title   string
	Content string
}

// GenericSelect safely reads rows from a dynamic entity table.
func (db *DB) GenericSelect(table string) ([]*GenericRecord, error) {
	if !IsSafeIdentifier(table) {
		return nil, fmt.Errorf("invalid table identifier: %s", table)
	}

	selectStmt := "SELECT id, title, content FROM " + table + " ORDER BY id DESC"
	rows, err := db.Query(selectStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*GenericRecord
	for rows.Next() {
		r := &GenericRecord{}
		if err := rows.Scan(&r.ID, &r.Title, &r.Content); err == nil {
			items = append(items, r)
		}
	}
	return items, nil
}
