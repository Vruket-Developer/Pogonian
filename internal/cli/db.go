package cli

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"pogonian/internal/config"
	"pogonian/internal/database"
)

// RunDB handles `pogn db <subcommand>` or `pogn db:migrate`, `pogn db:rollback`, `pogn db:status`, `pogn db:new`.
func RunDB(args []string) error {
	if len(args) == 0 {
		PrintDBHelp()
		return nil
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "migrate", "up":
		return RunDBMigrate(subArgs)
	case "rollback", "down":
		return RunDBRollback(subArgs)
	case "status":
		return RunDBStatus(subArgs)
	case "new", "create":
		return RunDBNew(subArgs)
	default:
		return fmt.Errorf("unknown db command '%s'. Run 'pogn db' for available commands", sub)
	}
}

func getDBConn(projectDir string) (*sql.DB, *config.Config, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, nil, err
	}

	cfg, err := config.Load(absDir)
	if err != nil {
		cfg = config.DefaultConfig(filepath.Base(absDir))
	}

	dbPath := cfg.Database.Path
	if !filepath.IsAbs(dbPath) && dbPath != ":memory:" {
		dbPath = filepath.Join(absDir, dbPath)
	}

	driver := &database.SQLiteDriver{Path: dbPath}
	conn, err := driver.Open()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database (%s): %w", dbPath, err)
	}

	return conn, cfg, nil
}

// RunDBMigrate executes `pogn db:migrate`.
func RunDBMigrate(args []string) error {
	projectDir := "."
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		projectDir = args[0]
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	conn, _, err := getDBConn(absDir)
	if err != nil {
		return err
	}
	defer conn.Close()

	migrationsDir := filepath.Join(absDir, "migrations")
	engine := database.NewMigrationEngine(conn, &database.SQLiteDriver{})

	fmt.Println()
	fmt.Printf("🗄️  \x1b[1;36mRunning Database Migrations:\x1b[0m %s\n\n", migrationsDir)

	applied, err := engine.MigrateUp(migrationsDir)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	if len(applied) == 0 {
		fmt.Printf("✨ \x1b[1;32mDatabase schema is up to date!\x1b[0m 0 pending migrations.\n\n")
		return nil
	}

	for _, mig := range applied {
		fmt.Printf("  \x1b[1;32m✓ Applied:\x1b[0m %03d_%s.up.sql \x1b[1;30m(checksum: %s)\x1b[0m\n", mig.Version, mig.Name, mig.Checksum)
	}

	fmt.Println()
	fmt.Printf("✨ \x1b[1;32mSuccessfully applied %d migration(s)!\x1b[0m\n\n", len(applied))
	return nil
}

// RunDBRollback executes `pogn db:rollback [steps]`.
func RunDBRollback(args []string) error {
	projectDir := "."
	steps := 1

	for _, arg := range args {
		if s, err := strconv.Atoi(arg); err == nil && s > 0 {
			steps = s
		} else if !strings.HasPrefix(arg, "-") {
			projectDir = arg
		}
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	conn, _, err := getDBConn(absDir)
	if err != nil {
		return err
	}
	defer conn.Close()

	migrationsDir := filepath.Join(absDir, "migrations")
	engine := database.NewMigrationEngine(conn, &database.SQLiteDriver{})

	fmt.Println()
	fmt.Printf("↩️  \x1b[1;33mRolling Back Database (%d step(s)):\x1b[0m %s\n\n", steps, migrationsDir)

	rolledBack, err := engine.Rollback(migrationsDir, steps)
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	if len(rolledBack) == 0 {
		fmt.Printf("✨ No migrations to roll back.\n\n")
		return nil
	}

	for _, mig := range rolledBack {
		fmt.Printf("  \x1b[1;33m✓ Reverted:\x1b[0m %03d_%s.down.sql\n", mig.Version, mig.Name)
	}

	fmt.Println()
	fmt.Printf("✨ \x1b[1;32mSuccessfully rolled back %d migration(s)!\x1b[0m\n\n", len(rolledBack))
	return nil
}

// RunDBStatus executes `pogn db:status`.
func RunDBStatus(args []string) error {
	projectDir := "."
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		projectDir = args[0]
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	conn, _, err := getDBConn(absDir)
	if err != nil {
		return err
	}
	defer conn.Close()

	migrationsDir := filepath.Join(absDir, "migrations")
	engine := database.NewMigrationEngine(conn, &database.SQLiteDriver{})

	statuses, err := engine.Status(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to fetch migration status: %w", err)
	}

	fmt.Println()
	fmt.Printf("📊 \x1b[1;36mDatabase Migration Status:\x1b[0m %s\n\n", migrationsDir)
	fmt.Printf("  %-8s | %-32s | %s\n", "VERSION", "MIGRATION NAME", "STATUS")
	fmt.Printf("  %s\n", strings.Repeat("─", 65))

	if len(statuses) == 0 {
		fmt.Println("  (No migration files found in migrations/ directory)")
	}

	for _, st := range statuses {
		statusStr := "\x1b[1;33mPending\x1b[0m"
		if st.Applied {
			statusStr = fmt.Sprintf("\x1b[1;32mApplied\x1b[0m (%s)", st.AppliedAt.Format("Jan 02 15:04"))
		}
		fmt.Printf("  %03d      | %-32s | %s\n", st.Version, st.Name, statusStr)
	}
	fmt.Println()

	return nil
}

// RunDBNew executes `pogn db:new <name>`.
func RunDBNew(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("migration name is required: pogn db:new <migration_name>")
	}

	name := args[0]
	projectDir := "."
	if len(args) > 1 {
		projectDir = args[1]
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	migrationsDir := filepath.Join(absDir, "migrations")
	upPath, downPath, err := database.GenerateMigration(migrationsDir, name)
	if err != nil {
		return fmt.Errorf("failed to create migration files: %w", err)
	}

	relUp, _ := filepath.Rel(absDir, upPath)
	relDown, _ := filepath.Rel(absDir, downPath)

	fmt.Println()
	fmt.Printf("✨ \x1b[1;32mCreated new database migration pair:\x1b[0m\n")
	fmt.Printf("  • \x1b[1;36m%s\x1b[0m\n", relUp)
	fmt.Printf("  • \x1b[1;36m%s\x1b[0m\n\n", relDown)
	fmt.Printf("To apply your migration, run:\n")
	fmt.Printf("  \x1b[1;34mpogn db:migrate\x1b[0m\n\n")

	return nil
}

// PrintDBHelp prints help for db subcommands.
func PrintDBHelp() {
	fmt.Println("\n\x1b[1mDATABASE COMMANDS:\x1b[0m")
	fmt.Println("  pogn db:migrate       Apply all pending schema migrations")
	fmt.Println("  pogn db:rollback      Revert the last batch of applied migrations")
	fmt.Println("  pogn db:status        Display migration status and version history")
	fmt.Println("  pogn db:new <name>    Create a new versioned .up.sql and .down.sql migration pair")
	fmt.Println()
}
