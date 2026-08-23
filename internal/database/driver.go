package database

import (
	"context"
	"database/sql"
	"fmt"
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

// DefaultPoolConfig returns sensible defaults for web applications.
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

// SQLiteDriver implements DBDriver for SQLite.
type SQLiteDriver struct {
	Path string
}

func (s *SQLiteDriver) Type() DriverType {
	return DriverSQLite
}

func (s *SQLiteDriver) DSN() string {
	return fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)", s.Path)
}

func (s *SQLiteDriver) Open() (*sql.DB, error) {
	conn, err := sql.Open("sqlite", s.DSN())
	if err != nil {
		return nil, err
	}
	ConfigurePool(conn, DefaultPoolConfig())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func (s *SQLiteDriver) CreateTableSQL(table string, columns map[string]string) string {
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id INTEGER PRIMARY KEY AUTOINCREMENT);", table)
}

func (s *SQLiteDriver) MigrationTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS _pogn_migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version INTEGER UNIQUE NOT NULL,
		name TEXT NOT NULL,
		applied_at DATETIME NOT NULL,
		checksum TEXT NOT NULL
	);`
}
