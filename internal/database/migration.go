package database

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Migration represents a versioned schema migration file pair or block.
type Migration struct {
	Version  int
	Name     string
	UpSQL    string
	DownSQL  string
	UpFile   string
	DownFile string
	Checksum string
}

// MigrationRecord represents an applied migration from the tracking table.
type MigrationRecord struct {
	ID        int64
	Version   int
	Name      string
	AppliedAt time.Time
	Checksum  string
}

// MigrationStatus describes the current state of a migration.
type MigrationStatus struct {
	Version   int
	Name      string
	Applied   bool
	AppliedAt *time.Time
}

// MigrationEngine orchestrates schema migrations and version tracking.
type MigrationEngine struct {
	db     *sql.DB
	driver DBDriver
}

// NewMigrationEngine creates a migration engine for the given database connection.
func NewMigrationEngine(db *sql.DB, driver DBDriver) *MigrationEngine {
	if driver == nil {
		driver = &SQLiteDriver{}
	}
	return &MigrationEngine{
		db:     db,
		driver: driver,
	}
}

// EnsureTable creates the `_pogn_migrations` metadata table if it does not exist.
func (m *MigrationEngine) EnsureTable() error {
	_, err := m.db.Exec(m.driver.MigrationTableSQL())
	return err
}

// DiscoverMigrations scans the `migrations/` directory and parses `.up.sql` and `.down.sql` files.
func (m *MigrationEngine) DiscoverMigrations(dir string) ([]*Migration, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []*Migration{}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	migrationMap := make(map[int]*Migration)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		// Expected format: 001_initial.up.sql or 001_initial.down.sql
		parts := strings.SplitN(name, "_", 2)
		if len(parts) < 2 {
			continue
		}

		ver, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		mig, exists := migrationMap[ver]
		if !exists {
			mig = &Migration{
				Version: ver,
				Name:    strings.TrimSuffix(strings.TrimSuffix(parts[1], ".up.sql"), ".down.sql"),
			}
			migrationMap[ver] = mig
		}

		filePath := filepath.Join(dir, name)
		contentBytes, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration %s: %w", name, err)
		}
		content := string(contentBytes)

		if strings.HasSuffix(name, ".up.sql") {
			mig.UpSQL = content
			mig.UpFile = filePath
			sum := sha256.Sum256(contentBytes)
			mig.Checksum = hex.EncodeToString(sum[:8])
		} else if strings.HasSuffix(name, ".down.sql") {
			mig.DownSQL = content
			mig.DownFile = filePath
		}
	}

	var list []*Migration
	for _, mig := range migrationMap {
		list = append(list, mig)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Version < list[j].Version
	})

	return list, nil
}

// GetAppliedMigrations queries `_pogn_migrations` and returns a map of applied versions.
func (m *MigrationEngine) GetAppliedMigrations() (map[int]*MigrationRecord, error) {
	if err := m.EnsureTable(); err != nil {
		return nil, err
	}

	rows, err := m.db.Query("SELECT id, version, name, applied_at, checksum FROM _pogn_migrations ORDER BY version ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]*MigrationRecord)
	for rows.Next() {
		var rec MigrationRecord
		var appliedAtStr string
		if err := rows.Scan(&rec.ID, &rec.Version, &rec.Name, &appliedAtStr, &rec.Checksum); err != nil {
			return nil, err
		}
		rec.AppliedAt, _ = time.Parse(time.RFC3339, appliedAtStr)
		applied[rec.Version] = &rec
	}

	return applied, nil
}

// MigrateUp executes all pending migrations in ascending order within individual transactions.
func (m *MigrationEngine) MigrateUp(dir string) ([]*Migration, error) {
	if err := m.EnsureTable(); err != nil {
		return nil, err
	}

	discovered, err := m.DiscoverMigrations(dir)
	if err != nil {
		return nil, err
	}

	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	var appliedMigrations []*Migration

	for _, mig := range discovered {
		if _, ok := applied[mig.Version]; ok {
			continue // Already applied
		}

		if strings.TrimSpace(mig.UpSQL) == "" {
			continue
		}

		tx, err := m.db.Begin()
		if err != nil {
			return appliedMigrations, fmt.Errorf("failed to begin transaction for migration %03d: %w", mig.Version, err)
		}

		// Execute up SQL statements
		statements := strings.Split(mig.UpSQL, ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := tx.Exec(stmt); err != nil {
				_ = tx.Rollback()
				return appliedMigrations, fmt.Errorf("migration %03d_%s failed at statement [%s]: %w", mig.Version, mig.Name, stmt, err)
			}
		}

		// Record migration in _pogn_migrations
		insertQuery := "INSERT INTO _pogn_migrations (version, name, applied_at, checksum) VALUES (?, ?, ?, ?)"
		nowStr := time.Now().UTC().Format(time.RFC3339)
		if _, err := tx.Exec(insertQuery, mig.Version, mig.Name, nowStr, mig.Checksum); err != nil {
			_ = tx.Rollback()
			return appliedMigrations, fmt.Errorf("failed to record migration %03d: %w", mig.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return appliedMigrations, fmt.Errorf("failed to commit migration %03d: %w", mig.Version, err)
		}

		appliedMigrations = append(appliedMigrations, mig)
	}

	return appliedMigrations, nil
}

// Rollback rolls back the last `steps` applied migrations using their `.down.sql`.
func (m *MigrationEngine) Rollback(dir string, steps int) ([]*Migration, error) {
	if err := m.EnsureTable(); err != nil {
		return nil, err
	}

	discovered, err := m.DiscoverMigrations(dir)
	if err != nil {
		return nil, err
	}

	migLookup := make(map[int]*Migration)
	for _, d := range discovered {
		migLookup[d.Version] = d
	}

	rows, err := m.db.Query("SELECT version, name FROM _pogn_migrations ORDER BY version DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type appEntry struct {
		version int
		name    string
	}
	var appliedList []appEntry
	for rows.Next() {
		var e appEntry
		if err := rows.Scan(&e.version, &e.name); err == nil {
			appliedList = append(appliedList, e)
		}
	}

	if len(appliedList) == 0 {
		return []*Migration{}, nil
	}

	if steps <= 0 {
		steps = 1
	}
	if steps > len(appliedList) {
		steps = len(appliedList)
	}

	var rolledBack []*Migration

	for i := 0; i < steps; i++ {
		entry := appliedList[i]
		mig, exists := migLookup[entry.version]
		if !exists || strings.TrimSpace(mig.DownSQL) == "" {
			return rolledBack, fmt.Errorf("cannot rollback migration %03d: down script missing or empty", entry.version)
		}

		tx, err := m.db.Begin()
		if err != nil {
			return rolledBack, err
		}

		statements := strings.Split(mig.DownSQL, ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := tx.Exec(stmt); err != nil {
				_ = tx.Rollback()
				return rolledBack, fmt.Errorf("rollback of migration %03d failed at [%s]: %w", entry.version, stmt, err)
			}
		}

		if _, err := tx.Exec("DELETE FROM _pogn_migrations WHERE version = ?", entry.version); err != nil {
			_ = tx.Rollback()
			return rolledBack, err
		}

		if err := tx.Commit(); err != nil {
			return rolledBack, err
		}

		rolledBack = append(rolledBack, mig)
	}

	return rolledBack, nil
}

// Status returns a list of all migrations and whether they have been applied.
func (m *MigrationEngine) Status(dir string) ([]MigrationStatus, error) {
	discovered, err := m.DiscoverMigrations(dir)
	if err != nil {
		return nil, err
	}

	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	var statuses []MigrationStatus
	for _, d := range discovered {
		st := MigrationStatus{
			Version: d.Version,
			Name:    d.Name,
		}
		if rec, ok := applied[d.Version]; ok {
			st.Applied = true
			st.AppliedAt = &rec.AppliedAt
		}
		statuses = append(statuses, st)
	}

	return statuses, nil
}

// GenerateMigration creates a new numbered `.up.sql` and `.down.sql` template in the migrations folder.
func GenerateMigration(dir, name string) (string, string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", err
	}

	entries, _ := os.ReadDir(dir)
	highestVer := 0

	for _, e := range entries {
		parts := strings.SplitN(e.Name(), "_", 2)
		if len(parts) >= 2 {
			if ver, err := strconv.Atoi(parts[0]); err == nil && ver > highestVer {
				highestVer = ver
			}
		}
	}

	nextVer := highestVer + 1
	cleanName := strings.ToLower(strings.ReplaceAll(name, " ", "_"))

	upFileName := fmt.Sprintf("%03d_%s.up.sql", nextVer, cleanName)
	downFileName := fmt.Sprintf("%03d_%s.down.sql", nextVer, cleanName)

	upPath := filepath.Join(dir, upFileName)
	downPath := filepath.Join(dir, downFileName)

	upContent := fmt.Sprintf("-- Migration Up: %03d_%s\n-- Write your schema changes (CREATE, ALTER, INSERT) below:\n\n", nextVer, cleanName)
	downContent := fmt.Sprintf("-- Migration Down: %03d_%s\n-- Write instructions to reverse the up migration (DROP TABLE, etc.) below:\n\n", nextVer, cleanName)

	if err := os.WriteFile(upPath, []byte(upContent), 0644); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(downPath, []byte(downContent), 0644); err != nil {
		return "", "", err
	}

	return upPath, downPath, nil
}
