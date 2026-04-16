package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Initialize(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign keys
	if _, err := DB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return runMigrations()
}

func runMigrations() error {
	// Try to find the migration file in multiple locations
	possiblePaths := []string{
		"migrations/001_initial_schema.sql",       // When running from backend root
		"../../migrations/001_initial_schema.sql", // When running tests from internal/database
		"../migrations/001_initial_schema.sql",    // Fallback
	}

	var migrationPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			migrationPath = path
			break
		}
	}

	if migrationPath == "" {
		return fmt.Errorf("migration file not found. Checked paths: %v", possiblePaths)
	}

	content, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("failed to read migration file %s: %w", migrationPath, err)
	}

	if _, err := DB.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", migrationPath, err)
	}

	return nil
}

func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

func GetDB() *sql.DB {
	return DB
}
