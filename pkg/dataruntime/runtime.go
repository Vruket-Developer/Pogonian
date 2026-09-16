package dataruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Record represents an entity row/document.
type Record map[string]any

// Operator represents query comparison operators.
type Operator string

const (
	OpEq   Operator = "="
	OpNeq  Operator = "!="
	OpGt   Operator = ">"
	OpGte  Operator = ">="
	OpLt   Operator = "<"
	OpLte  Operator = "<="
	OpLike Operator = "LIKE"
)

// Filter specifies a condition for queries.
type Filter struct {
	Field    string
	Operator Operator
	Value    any
}

// Query defines structured data retrieval parameters.
type Query struct {
	Entity    string
	Filters   []Filter
	OrderBy   string
	OrderDesc bool
	Limit     int
	Offset    int
}

// StorageEngine defines the low-level persistence driver interface.
type StorageEngine interface {
	Name() string
	Find(ctx context.Context, q Query) ([]Record, error)
	FindByID(ctx context.Context, entity string, id any) (Record, error)
	Insert(ctx context.Context, entity string, record Record) (any, error)
	Update(ctx context.Context, entity string, id any, record Record) error
	Delete(ctx context.Context, entity string, id any) error
	Count(ctx context.Context, entity string, filters []Filter) (int64, error)
	EnsureSchema(ctx context.Context, entity string, schema map[string]string) error
	Close() error
}

// SQLiteEngine implements the native local storage engine using SQLite WAL mode.
type SQLiteEngine struct {
	db     *sql.DB
	path   string
	mu     sync.RWMutex
	tables map[string]bool
}

// NewSQLiteEngine initializes SQLite with production concurrency pragmas.
func NewSQLiteEngine(dbPath string) (*SQLiteEngine, error) {
	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	return &SQLiteEngine{
		db:     db,
		path:   dbPath,
		tables: make(map[string]bool),
	}, nil
}

func (s *SQLiteEngine) Name() string {
	return "sqlite_wal"
}

func (s *SQLiteEngine) Close() error {
	return s.db.Close()
}

// DB returns the underlying sql.DB instance.
func (s *SQLiteEngine) DB() *sql.DB {
	return s.db
}

func (s *SQLiteEngine) EnsureSchema(ctx context.Context, entity string, schema map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tableName := sanitizeIdentifier(entity)
	var colDefs []string
	colDefs = append(colDefs, "id INTEGER PRIMARY KEY AUTOINCREMENT")

	for col, colType := range schema {
		if strings.EqualFold(col, "id") {
			continue
		}
		colDefs = append(colDefs, fmt.Sprintf("%s %s", sanitizeIdentifier(col), colType))
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", tableName, strings.Join(colDefs, ", "))
	_, err := s.db.ExecContext(ctx, query)
	if err == nil {
		s.tables[tableName] = true
	}
	return err
}

func (s *SQLiteEngine) Find(ctx context.Context, q Query) ([]Record, error) {
	tableName := sanitizeIdentifier(q.Entity)
	whereClauses, args := buildWhereClause(q.Filters)

	sqlStr := fmt.Sprintf("SELECT * FROM %s", tableName)
	if len(whereClauses) > 0 {
		sqlStr += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	if q.OrderBy != "" {
		direction := "ASC"
		if q.OrderDesc {
			direction = "DESC"
		}
		sqlStr += fmt.Sprintf(" ORDER BY %s %s", sanitizeIdentifier(q.OrderBy), direction)
	}

	if q.Limit > 0 {
		sqlStr += fmt.Sprintf(" LIMIT %d", q.Limit)
		if q.Offset > 0 {
			sqlStr += fmt.Sprintf(" OFFSET %d", q.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []Record
	for rows.Next() {
		colVals := make([]any, len(cols))
		colPointers := make([]any, len(cols))
		for i := range colVals {
			colPointers[i] = &colVals[i]
		}

		if err := rows.Scan(colPointers...); err != nil {
			return nil, err
		}

		rec := make(Record)
		for i, colName := range cols {
			val := colVals[i]
			if b, ok := val.([]byte); ok {
				rec[colName] = string(b)
			} else {
				rec[colName] = val
			}
		}
		results = append(results, rec)
	}

	return results, rows.Err()
}

func (s *SQLiteEngine) FindByID(ctx context.Context, entity string, id any) (Record, error) {
	records, err := s.Find(ctx, Query{
		Entity: entity,
		Filters: []Filter{
			{Field: "id", Operator: OpEq, Value: id},
		},
		Limit: 1,
	})
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("entity '%s' with id '%v' not found", entity, id)
	}
	return records[0], nil
}

func (s *SQLiteEngine) Insert(ctx context.Context, entity string, record Record) (any, error) {
	tableName := sanitizeIdentifier(entity)

	var cols []string
	var placeholders []string
	var args []any

	for col, val := range record {
		if strings.EqualFold(col, "id") && val == nil {
			continue
		}
		cols = append(cols, sanitizeIdentifier(col))
		placeholders = append(placeholders, "?")
		args = append(args, val)
	}

	if len(cols) == 0 {
		return nil, fmt.Errorf("cannot insert empty record into '%s'", entity)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName, strings.Join(cols, ", "), strings.Join(placeholders, ", "))

	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return res.LastInsertId()
}

func (s *SQLiteEngine) Update(ctx context.Context, entity string, id any, record Record) error {
	tableName := sanitizeIdentifier(entity)

	var setClauses []string
	var args []any

	for col, val := range record {
		if strings.EqualFold(col, "id") {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", sanitizeIdentifier(col)))
		args = append(args, val)
	}

	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", tableName, strings.Join(setClauses, ", "))

	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("entity '%s' with id '%v' not found for update", entity, id)
	}
	return nil
}

func (s *SQLiteEngine) Delete(ctx context.Context, entity string, id any) error {
	tableName := sanitizeIdentifier(entity)
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", tableName)

	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("entity '%s' with id '%v' not found for deletion", entity, id)
	}
	return nil
}

func (s *SQLiteEngine) Count(ctx context.Context, entity string, filters []Filter) (int64, error) {
	tableName := sanitizeIdentifier(entity)
	whereClauses, args := buildWhereClause(filters)

	sqlStr := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	if len(whereClauses) > 0 {
		sqlStr += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	var count int64
	err := s.db.QueryRowContext(ctx, sqlStr, args...).Scan(&count)
	return count, err
}

func buildWhereClause(filters []Filter) ([]string, []any) {
	var clauses []string
	var args []any

	for _, f := range filters {
		op := f.Operator
		if op == "" {
			op = OpEq
		}
		clauses = append(clauses, fmt.Sprintf("%s %s ?", sanitizeIdentifier(f.Field), op))
		args = append(args, f.Value)
	}

	return clauses, args
}

func sanitizeIdentifier(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// DataRuntime provides the platform-independent data access layer for Pogonian applications.
type DataRuntime struct {
	engine StorageEngine
}

// New creates a new Pogonian Data Runtime with the given storage engine.
func New(engine StorageEngine) *DataRuntime {
	return &DataRuntime{engine: engine}
}

// Default creates a Data Runtime backed by local SQLite.
func Default(sqlitePath string) (*DataRuntime, error) {
	engine, err := NewSQLiteEngine(sqlitePath)
	if err != nil {
		return nil, err
	}
	return New(engine), nil
}

// Engine returns the underlying storage engine.
func (dr *DataRuntime) Engine() StorageEngine {
	return dr.engine
}

// DB returns the underlying *sql.DB if supported by the storage engine.
func (dr *DataRuntime) DB() *sql.DB {
	if se, ok := dr.engine.(*SQLiteEngine); ok {
		return se.DB()
	}
	return nil
}

// EnsureEntity registers an entity schema in the active storage engine.
func (dr *DataRuntime) EnsureEntity(ctx context.Context, entity string, schema map[string]string) error {
	return dr.engine.EnsureSchema(ctx, entity, schema)
}

// Find executes a query against an entity.
func (dr *DataRuntime) Find(ctx context.Context, entity string, filters ...Filter) ([]Record, error) {
	return dr.engine.Find(ctx, Query{
		Entity:  entity,
		Filters: filters,
	})
}

// FindByID retrieves a single entity record by ID.
func (dr *DataRuntime) FindByID(ctx context.Context, entity string, id any) (Record, error) {
	return dr.engine.FindByID(ctx, entity, id)
}

// Insert inserts a new record into the entity.
func (dr *DataRuntime) Insert(ctx context.Context, entity string, record Record) (any, error) {
	return dr.engine.Insert(ctx, entity, record)
}

// Update updates an existing record in the entity.
func (dr *DataRuntime) Update(ctx context.Context, entity string, id any, record Record) error {
	return dr.engine.Update(ctx, entity, id, record)
}

// Delete removes an entity record by ID.
func (dr *DataRuntime) Delete(ctx context.Context, entity string, id any) error {
	return dr.engine.Delete(ctx, entity, id)
}

// Count returns the number of records matching filters.
func (dr *DataRuntime) Count(ctx context.Context, entity string, filters ...Filter) (int64, error) {
	return dr.engine.Count(ctx, entity, filters)
}

// Close closes the underlying storage engine.
func (dr *DataRuntime) Close() error {
	return dr.engine.Close()
}
