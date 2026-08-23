package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	return db.GenericInsertForm(table, map[string]string{
		"title":   title,
		"content": content,
	})
}

// GenericInsertForm dynamically creates table columns and inserts any form key-value pairs.
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
		if k == "_csrf" || k == "" || !IsSafeIdentifier(k) {
			continue
		}
		cols = append(cols, k)
		colDefs = append(colDefs, k+" TEXT")
		placeholders = append(placeholders, "?")
		vals = append(vals, v)
	}

	cols = append(cols, "created_at")
	colDefs = append(colDefs, "created_at DATETIME DEFAULT CURRENT_TIMESTAMP")
	placeholders = append(placeholders, "?")
	vals = append(vals, time.Now().UTC().Format(time.RFC3339))

	// Ensure table exists
	createStmt := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id INTEGER PRIMARY KEY AUTOINCREMENT, %s);",
		table, strings.Join(colDefs, ", "))
	if _, err := db.Exec(createStmt); err != nil {
		return err
	}

	// For existing tables, ensure columns exist
	for _, col := range cols {
		if col != "id" && col != "created_at" && IsSafeIdentifier(col) {
			alterStmt := "ALTER TABLE " + table + " ADD COLUMN " + col + " TEXT"
			_, _ = db.Exec(alterStmt)
		}
	}

	queryStmt := "INSERT INTO " + table + " (" + strings.Join(cols, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"
	_, err := db.Exec(queryStmt, vals...)
	return err
}

// GenericSelect safely reads all columns from any table into map entries.
func (db *DB) GenericSelect(table string) ([]map[string]interface{}, error) {
	if !IsSafeIdentifier(table) {
		return nil, fmt.Errorf("invalid table identifier: %s", table)
	}

	selectStmt := "SELECT * FROM " + table + " ORDER BY id DESC"
	rows, err := db.Query(selectStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		entry := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			if strVal, ok := val.(string); ok {
				if strings.Contains(strings.ToLower(col), "price") || strings.Contains(strings.ToLower(col), "amount") {
					if f, err := strconv.ParseFloat(strVal, 64); err == nil {
						val = f
					}
				}
			}
			entry[col] = val

			pascal := ToPascalCase(col)
			entry[pascal] = val
			if strings.EqualFold(col, "id") {
				entry["ID"] = val
			}
		}
		results = append(results, entry)
	}
	return results, nil
}

// GenericSelectByID reads a single record from any table by ID.
func (db *DB) GenericSelectByID(table string, id int64) (map[string]interface{}, error) {
	if !IsSafeIdentifier(table) {
		return nil, fmt.Errorf("invalid table identifier: %s", table)
	}

	queryStmt := "SELECT * FROM " + table + " WHERE id = ?"
	rows, err := db.Query(queryStmt, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		entry := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			if strVal, ok := val.(string); ok {
				if strings.Contains(strings.ToLower(col), "price") || strings.Contains(strings.ToLower(col), "amount") {
					if f, err := strconv.ParseFloat(strVal, 64); err == nil {
						val = f
					}
				}
			}
			entry[col] = val
			pascal := ToPascalCase(col)
			entry[pascal] = val
			if strings.EqualFold(col, "id") {
				entry["ID"] = val
			}
		}
		return entry, nil
	}
	return nil, nil
}

// ToPascalCase converts snake_case or lowercase string to PascalCase (e.g. item_name -> ItemName, created_at -> CreatedAt).
func ToPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	res := strings.Join(parts, "")
	if strings.EqualFold(res, "id") {
		return "ID"
	}
	return res
}
