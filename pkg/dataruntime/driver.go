package dataruntime

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DriverType defines supported database dialect types.
type DriverType string

const (
	DriverSQLite   DriverType = "sqlite"
	DriverPostgres DriverType = "postgres"
	DriverMySQL    DriverType = "mysql"
)

// DBDriver defines the standardized dialect interface for Pogonian databases.
type DBDriver interface {
	Type() DriverType
	DSN() string
	Open() (*sql.DB, error)
	CreateTableSQL(table string, columns map[string]string) string
	MigrationTableSQL() string
}

// PoolConfig specifies connection pooling parameters.
type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultPoolConfig returns sensible defaults for high-performance applications.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 15 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}
}

// ConfigurePool configures pool limits on a sql.DB instance.
func ConfigurePool(db *sql.DB, cfg PoolConfig) {
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}
}

// SQLiteDriver implements DBDriver for SQLite WAL mode.
type SQLiteDriver struct {
	Path string
}

func (s *SQLiteDriver) Type() DriverType {
	return DriverSQLite
}

func (s *SQLiteDriver) DSN() string {
	if s.Path == "" {
		s.Path = "data.db"
	}
	return fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)", s.Path)
}

func (s *SQLiteDriver) Open() (*sql.DB, error) {
	if s.Path != ":memory:" && s.Path != "" {
		dir := filepath.Dir(s.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite", s.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
	}
	ConfigurePool(db, DefaultPoolConfig())
	return db, db.Ping()
}

func (s *SQLiteDriver) CreateTableSQL(table string, columns map[string]string) string {
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id INTEGER PRIMARY KEY AUTOINCREMENT", sanitizeIdentifier(table))
	for col, colType := range columns {
		if col == "id" {
			continue
		}
		query += fmt.Sprintf(", %s %s", sanitizeIdentifier(col), colType)
	}
	query += ");"
	return query
}

func (s *SQLiteDriver) MigrationTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS _pogn_migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version INTEGER UNIQUE NOT NULL,
		name TEXT NOT NULL,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		checksum TEXT NOT NULL
	);`
}
