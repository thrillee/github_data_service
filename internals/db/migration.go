// internal/db/db.go
package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/pressly/goose/v3"
)

// RunMigrations runs all database migrations
func RunMigrations(migrationsDir, dbString string) error {
	db, err := sql.Open("sqlite3", dbString)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Set up goose
	goose.SetBaseFS(nil) // Use the OS filesystem
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// Run migrations
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Migrations completed successfully")
	return nil
}
