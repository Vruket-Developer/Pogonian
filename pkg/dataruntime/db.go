package dataruntime

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps an active database connection with helper utilities and generic query methods.
type DB struct {
	*sql.DB
	path string
}

// Open initializes and returns a DB instance configured with SQLite WAL pragmas.
func Open(path string) (*DB, error) {
	driver := &SQLiteDriver{Path: path}
	db, err := driver.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return &DB{DB: db, path: path}, nil
}

// Path returns the configured database path.
func (d *DB) Path() string {
	return d.path
}

// Migrate executes an array of SQL migration scripts in a transaction.
func (d *DB) Migrate(queries ...string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, q := range queries {
		if strings.TrimSpace(q) == "" {
			continue
		}
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("migration failed for statement [%s]: %w", q, err)
		}
	}

	return tx.Commit()
}

// IsSafeIdentifier checks if a table or column name is purely alphanumeric and safe.
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

// TableExists checks if a table exists in the database schema.
func (d *DB) TableExists(table string) bool {
	if !IsSafeIdentifier(table) {
		return false
	}
	var exists int
	err := d.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name = ?", table).Scan(&exists)
	return err == nil && exists == 1
}

// GenericInsert safely inserts title and content into a dynamic entity table.
func (d *DB) GenericInsert(table, title, content string) error {
	return d.GenericInsertForm(table, map[string]string{
		"title":   title,
		"content": content,
	})
}

// GenericInsertForm dynamically creates table columns and inserts form key-value pairs.
func (d *DB) GenericInsertForm(table string, formValues map[string]string) error {
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
	if _, err := d.Exec(createStmt); err != nil {
		return err
	}

	// For existing tables, ensure columns exist
	for _, col := range cols {
		if col != "id" && col != "created_at" && IsSafeIdentifier(col) {
			alterStmt := "ALTER TABLE " + table + " ADD COLUMN " + col + " TEXT"
			_, _ = d.Exec(alterStmt)
		}
	}

	queryStmt := "INSERT INTO " + table + " (" + strings.Join(cols, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"
	_, err := d.Exec(queryStmt, vals...)
	return err
}

// GenericSelect safely reads all columns from any table into map entries.
func (d *DB) GenericSelect(table string) ([]map[string]interface{}, error) {
	if !IsSafeIdentifier(table) {
		return nil, fmt.Errorf("invalid table identifier: %s", table)
	}

	selectStmt := "SELECT * FROM " + table + " ORDER BY id DESC"
	rows, err := d.Query(selectStmt)
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
func (d *DB) GenericSelectByID(table string, id int64) (map[string]interface{}, error) {
	if !IsSafeIdentifier(table) {
		return nil, fmt.Errorf("invalid table identifier: %s", table)
	}

	queryStmt := "SELECT * FROM " + table + " WHERE id = ?"
	rows, err := d.Query(queryStmt, id)
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

// GenericDeleteByID deletes a single record from any table by ID.
func (d *DB) GenericDeleteByID(table string, id int64) error {
	if !IsSafeIdentifier(table) {
		return fmt.Errorf("invalid table identifier: %s", table)
	}

	queryStmt := "DELETE FROM " + table + " WHERE id = ?"
	_, err := d.Exec(queryStmt, id)
	return err
}

// GenericUpdateByID updates specified columns for a record by ID.
func (d *DB) GenericUpdateByID(table string, id int64, formValues map[string]string) error {
	if !IsSafeIdentifier(table) {
		return fmt.Errorf("invalid table identifier: %s", table)
	}

	var setClauses []string
	var vals []interface{}

	for k, v := range formValues {
		k = strings.TrimSpace(k)
		if k == "_csrf" || k == "_method" || k == "id" || k == "" || !IsSafeIdentifier(k) {
			continue
		}
		setClauses = append(setClauses, k+" = ?")
		vals = append(vals, v)
	}

	if len(setClauses) == 0 {
		return nil
	}

	vals = append(vals, id)
	queryStmt := "UPDATE " + table + " SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	_, err := d.Exec(queryStmt, vals...)
	return err
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
