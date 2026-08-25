package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// GenericInsertForm dynamically creates table columns and inserts form values.
func (db *DB) GenericInsertForm(table string, formValues map[string]string) error {
	if !IsSafeIdentifier(table) {
		return fmt.Errorf("invalid table identifier: %s", table)
	}

	var cols []string
	var colDefs []string
	var placeholders []string
	var vals []interface{}

	for k, v := range formValues {
		k = strings.TrimSpace(k)
		if k == "" || !IsSafeIdentifier(k) {
			continue
		}
		cols = append(cols, k)
		colDefs = append(colDefs, fmt.Sprintf("%s TEXT", k))
		placeholders = append(placeholders, "?")
		vals = append(vals, v)
	}

	if len(cols) == 0 {
		return nil
	}

	createTableQuery := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			%s,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`, table, strings.Join(colDefs, ", "))

	if _, err := db.Exec(createTableQuery); err != nil {
		return fmt.Errorf("failed to ensure table %s: %w", table, err)
	}

	insertQuery := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`,
		table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))

	_, err := db.Exec(insertQuery, vals...)
	return err
}

// GenericSelect retrieves all rows from the specified table as key-value maps.
func (db *DB) GenericSelect(table string) ([]map[string]interface{}, error) {
	if !IsSafeIdentifier(table) {
		return nil, fmt.Errorf("invalid table identifier: %s", table)
	}

	query := fmt.Sprintf("SELECT * FROM %s ORDER BY id DESC", table)
	rows, err := db.Query(query)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		colVals := make([]interface{}, len(cols))
		colPointers := make([]interface{}, len(cols))
		for i := range colVals {
			colPointers[i] = &colVals[i]
		}

		if err := rows.Scan(colPointers...); err != nil {
			continue
		}

		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := colVals[i]
			if b, ok := val.([]byte); ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		results = append(results, rowMap)
	}

	return results, nil
}

// GenericSelectByID retrieves a single row by ID from the specified table.
func (db *DB) GenericSelectByID(table string, id int64) (map[string]interface{}, error) {
	if !IsSafeIdentifier(table) {
		return nil, fmt.Errorf("invalid table identifier: %s", table)
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE id = ? LIMIT 1", table)
	rows, err := db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		colVals := make([]interface{}, len(cols))
		colPointers := make([]interface{}, len(cols))
		for i := range colVals {
			colPointers[i] = &colVals[i]
		}

		if err := rows.Scan(colPointers...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := colVals[i]
			if b, ok := val.([]byte); ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		return rowMap, nil
	}

	return nil, sql.ErrNoRows
}
