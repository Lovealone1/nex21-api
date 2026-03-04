package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load the .env file
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Warning: Error loading .env file (using system env vars if available)")
	}

	// 2. We use DIRECT_URL because pooler urls (6543) often cause issues with raw DDL migrations
	dbURL := os.Getenv("DIRECT_URL")
	if dbURL == "" {
		log.Fatal("DIRECT_URL is not set in environment")
	}

	// 3. Connect to Supabase
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx)

	// 4. Create a tracking table to avoid re-running migrations
	// Drop the existing table to fix data type inconsistencies from golang-migrate (which uses BIGINT)
	_, _ = conn.Exec(ctx, `DROP TABLE IF EXISTS schema_migrations`)
	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create schema_migrations table: %v\n", err)
	}

	// 5. Read all files in the migrations directory
	entries, err := os.ReadDir("migrations")
	if err != nil {
		log.Fatalf("Failed to read migrations directory: %v\n", err)
	}

	// Filter and sort .up.sql files
	var migrationFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			migrationFiles = append(migrationFiles, entry.Name())
		}
	}
	sort.Strings(migrationFiles) // Emulate standard numbering like 000001, 000002...

	if len(migrationFiles) == 0 {
		fmt.Println("No migrations found in the 'migrations' directory.")
		return
	}

	// 6. Execute unapplied migrations
	for _, file := range migrationFiles {
		var exists bool
		err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", file).Scan(&exists)
		if err != nil {
			log.Fatalf("Failed to check migration status for %s: %v\n", file, err)
		}

		if exists {
			fmt.Printf("⏭️  Skipping %s (already applied)\n", file)
			continue
		}

		log.Printf("Executing migration: %s...\n", file)
		data, err := os.ReadFile(filepath.Join("migrations", file))
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v\n", file, err)
		}

		// Run in a transaction to ensure atomic execution per file
		tx, err := conn.Begin(ctx)
		if err != nil {
			log.Fatalf("Failed to begin transaction: %v\n", err)
		}

		_, err = tx.Exec(ctx, string(data))
		if err != nil {
			tx.Rollback(ctx)
			log.Fatalf("Migration failed %s: %v\n", file, err)
		}

		// Record the applied migration
		_, err = tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", file)
		if err != nil {
			tx.Rollback(ctx)
			log.Fatalf("Failed to record migration %s: %v\n", file, err)
		}

		if err := tx.Commit(ctx); err != nil {
			log.Fatalf("Failed to commit migration %s: %v\n", file, err)
		}

		fmt.Printf("Migration %s applied successfully.\n", file)
	}

	fmt.Println("Database is up to date!")
}
