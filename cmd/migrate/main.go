package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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

	// 3. Parse subcommand: up (default) | down [N]
	command := "up"
	downSteps := 1
	if len(os.Args) > 1 {
		command = os.Args[1]
		if command == "down" && len(os.Args) > 2 {
			n, err := strconv.Atoi(os.Args[2])
			if err != nil || n < 1 {
				log.Fatalf("Invalid number of steps for down: %q (must be a positive integer)", os.Args[2])
			}
			downSteps = n
		}
	}
	if command != "up" && command != "down" {
		log.Fatalf("Unknown command %q. Usage: go run ./cmd/migrate [up|down [N]]", command)
	}

	// 4. Connect to Supabase
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx)

	// 5. Create tracking table if it doesn't exist
	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create schema_migrations table: %v\n", err)
	}

	// 6. Read all files in the migrations directory
	entries, err := os.ReadDir("migrations")
	if err != nil {
		log.Fatalf("Failed to read migrations directory: %v\n", err)
	}

	if command == "up" {
		runUp(ctx, conn, entries)
	} else {
		runDown(ctx, conn, downSteps)
	}
}

// runUp applies all pending .up.sql migrations in ascending order.
func runUp(ctx context.Context, conn *pgx.Conn, entries []os.DirEntry) {
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	if len(files) == 0 {
		fmt.Println("No .up.sql migrations found.")
		return
	}

	applied := 0
	for _, file := range files {
		var exists bool
		if err := conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", file).Scan(&exists); err != nil {
			log.Fatalf("Failed to check migration status for %s: %v\n", file, err)
		}
		if exists {
			fmt.Printf("Skipping %s (already applied)\n", file)
			continue
		}

		execMigration(ctx, conn, file, file)
		applied++
	}

	if applied == 0 {
		fmt.Println("Database is already up to date!")
	} else {
		fmt.Printf("Done! %d migration(s) applied.\n", applied)
	}
}

// runDown rolls back the last N applied migrations in reverse order using .down.sql files.
func runDown(ctx context.Context, conn *pgx.Conn, steps int) {
	rows, err := conn.Query(ctx, `
		SELECT version FROM schema_migrations
		ORDER BY applied_at DESC, version DESC
		LIMIT $1
	`, steps)
	if err != nil {
		log.Fatalf("Failed to query applied migrations: %v\n", err)
	}
	defer rows.Close()

	var toRollback []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			log.Fatalf("Failed to scan migration version: %v\n", err)
		}
		toRollback = append(toRollback, v)
	}

	if len(toRollback) == 0 {
		fmt.Println("Nothing to roll back.")
		return
	}

	for _, upFile := range toRollback {
		// Derive the .down.sql filename from the .up.sql version key
		downFile := strings.TrimSuffix(upFile, ".up.sql") + ".down.sql"

		if _, err := os.Stat(filepath.Join("migrations", downFile)); os.IsNotExist(err) {
			log.Fatalf("Down file not found for %s: expected %s\n", upFile, downFile)
		}

		// Run the down script (no tracking insert — we DELETE instead)
		execMigration(ctx, conn, downFile, "")

		if _, err := conn.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", upFile); err != nil {
			log.Fatalf("Failed to remove migration record for %s: %v\n", upFile, err)
		}
		fmt.Printf("Rolled back: %s\n", upFile)
	}

	fmt.Printf("Done! %d migration(s) rolled back.\n", len(toRollback))
}

// execMigration reads a SQL file and executes it inside a transaction.
// If trackAs is non-empty, it records the version in schema_migrations.
func execMigration(ctx context.Context, conn *pgx.Conn, filename string, trackAs string) {
	log.Printf("Executing %s...\n", filename)

	data, err := os.ReadFile(filepath.Join("migrations", filename))
	if err != nil {
		log.Fatalf("Failed to read %s: %v\n", filename, err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v\n", err)
	}

	if _, err = tx.Exec(ctx, string(data)); err != nil {
		tx.Rollback(ctx)
		log.Fatalf("Migration failed (%s): %v\n", filename, err)
	}

	if trackAs != "" {
		if _, err = tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", trackAs); err != nil {
			tx.Rollback(ctx)
			log.Fatalf("Failed to record migration %s: %v\n", trackAs, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("Failed to commit %s: %v\n", filename, err)
	}

	fmt.Printf("✓ %s\n", filename)
}
