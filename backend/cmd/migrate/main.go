package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/indraprhmbd/AwoChat/backend/internal/config"
	"github.com/indraprhmbd/AwoChat/backend/internal/database"
)

func main() {
	command := flag.String("command", "up", "Migration command: up, down, create")
	version := flag.Int("version", 0, "Target version for migration (0 for all)")
	name := flag.String("name", "", "Name for new migration (required for create command)")
	flag.Parse()

	cfg := config.Load()

	db, err := database.New(cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Ensure migrations table exists
	if err := ensureMigrationsTable(db); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create migrations table: %v\n", err)
		os.Exit(1)
	}

	switch *command {
	case "up":
		if err := runUp(db, *version); err != nil {
			fmt.Fprintf(os.Stderr, "Migration up failed: %v\n", err)
			os.Exit(1)
		}
	case "down":
		if err := runDown(db, *version); err != nil {
			fmt.Fprintf(os.Stderr, "Migration down failed: %v\n", err)
			os.Exit(1)
		}
	case "create":
		if *name == "" {
			fmt.Fprintln(os.Stderr, "Error: -name is required for create command")
			os.Exit(1)
		}
		if err := createMigration(*name); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create migration: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		os.Exit(1)
	}
}

func ensureMigrationsTable(db *database.DB) error {
	ctx := context.Background()
	_, err := db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func runUp(db *database.DB, targetVersion int) error {
	ctx := context.Background()

	// Get current version
	currentVersion, err := getCurrentVersion(db)
	if err != nil {
		return err
	}

	// Get all migration files
	migrations, err := getMigrationFiles()
	if err != nil {
		return err
	}

	// Filter and sort migrations to apply
	var toApply []int
	for v := range migrations {
		if v > currentVersion && (targetVersion == 0 || v <= targetVersion) {
			toApply = append(toApply, v)
		}
	}

	if len(toApply) == 0 {
		fmt.Println("No migrations to apply")
		return nil
	}

	// Sort ascending
	for i := 0; i < len(toApply)-1; i++ {
		for j := i + 1; j < len(toApply); j++ {
			if toApply[i] > toApply[j] {
				toApply[i], toApply[j] = toApply[j], toApply[i]
			}
		}
	}

	// Apply migrations
	for _, v := range toApply {
		fmt.Printf("Applying migration %d...\n", v)
		if err := applyMigration(ctx, db, v, migrations[v]); err != nil {
			return fmt.Errorf("migration %d failed: %w", v, err)
		}
		fmt.Printf("Migration %d applied successfully\n", v)
	}

	return nil
}

func runDown(db *database.DB, targetVersion int) error {
	ctx := context.Background()

	// Get current version
	currentVersion, err := getCurrentVersion(db)
	if err != nil {
		return err
	}

	if currentVersion == 0 {
		fmt.Println("No migrations to revert")
		return nil
	}

	// Determine target
	if targetVersion == 0 {
		targetVersion = currentVersion - 1
	}

	if targetVersion >= currentVersion {
		return fmt.Errorf("target version %d must be less than current version %d", targetVersion, currentVersion)
	}

	// Revert migrations from current down to target
	for v := currentVersion; v > targetVersion; v-- {
		migrationFile := getMigrationDownFile(v)
		if migrationFile == "" {
			return fmt.Errorf("no down migration found for version %d", v)
		}

		fmt.Printf("Reverting migration %d...\n", v)
		if err := revertMigration(ctx, db, v, migrationFile); err != nil {
			return fmt.Errorf("reverting migration %d failed: %w", v, err)
		}
		fmt.Printf("Migration %d reverted successfully\n", v)
	}

	return nil
}

func getCurrentVersion(db *database.DB) (int, error) {
	ctx := context.Background()
	var version int
	err := db.Pool.QueryRow(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	return version, err
}

func getMigrationFiles() (map[int]string, error) {
	migrations := make(map[int]string)

	dir := "migrations"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}

		// Extract version from filename: NNN_name.up.sql
		parts := strings.SplitN(name, "_", 2)
		if len(parts) != 2 {
			continue
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		migrations[version] = filepath.Join(dir, name)
	}

	return migrations, nil
}

func getMigrationDownFile(version int) string {
	dir := "migrations"
	pattern := fmt.Sprintf("%03d_*.down.sql", version)

	entries, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil || len(entries) == 0 {
		return ""
	}

	return entries[0]
}

func applyMigration(ctx context.Context, db *database.DB, version int, file string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	// Run in transaction
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, string(content)); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func revertMigration(ctx context.Context, db *database.DB, version int, file string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	// Run in transaction
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, string(content)); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", version); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func createMigration(name string) error {
	// Get next version number
	migrations, _ := getMigrationFiles()
	nextVersion := 1
	for v := range migrations {
		if v >= nextVersion {
			nextVersion = v + 1
		}
	}

	dir := "migrations"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	upFile := filepath.Join(dir, fmt.Sprintf("%03d_%s.up.sql", nextVersion, name))
	downFile := filepath.Join(dir, fmt.Sprintf("%03d_%s.down.sql", nextVersion, name))

	upContent := fmt.Sprintf(`-- migrations/%03d_%s.up.sql
-- TODO: Add your migration SQL here

`, nextVersion, name)

	downContent := fmt.Sprintf(`-- migrations/%03d_%s.down.sql
-- TODO: Add your rollback SQL here

`, nextVersion, name)

	if err := os.WriteFile(upFile, []byte(upContent), 0644); err != nil {
		return err
	}

	if err := os.WriteFile(downFile, []byte(downContent), 0644); err != nil {
		return err
	}

	fmt.Printf("Created migration %03d_%s\n", nextVersion, name)
	return nil
}
