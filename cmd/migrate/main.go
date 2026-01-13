package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Rrens/text-to-sql/internal/config"
	"github.com/Rrens/text-to-sql/internal/repository/postgres"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	fmt.Printf("Connecting to database at %s:%d...\n", cfg.Database.Host, cfg.Database.Port)

	// Connect to database
	db, err := postgres.NewDB(context.Background(), cfg.Database)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}
	defer db.Close()

	// Get migration files
	files, err := filepath.Glob("migrations/*.up.sql")
	if err != nil {
		panic(err)
	}
	sort.Strings(files)

	for _, file := range files {
		fmt.Printf("Applying migration: %s\n", file)
		content, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}

		// Split by semicolon to handle multiple statements if needed,
		// but pgx pool.Exec might handle it. Let's try executing the whole block.
		_, err = db.Pool.Exec(context.Background(), string(content))
		if err != nil {
			fmt.Printf("⚠️  Error applying %s: %v\n", file, err)
			// Don't exit, just continue. This mimics the "NOTICE: relation exists, skipping" behavior
			// if the SQL uses "IF NOT EXISTS", or just fails if it doesn't.
			// Ideally we should have a schema_migrations table, but for now this is the fix.
		} else {
			fmt.Printf("✅ %s applied successfully\n", file)
		}
	}
}
