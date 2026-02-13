package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"time"

	"github.com/Rrens/text-to-sql/internal/api"
	"github.com/Rrens/text-to-sql/internal/config"
	"github.com/Rrens/text-to-sql/internal/repository/postgres"
	"github.com/Rrens/text-to-sql/internal/repository/redis"
	"github.com/joho/godotenv"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// ... existing env loading code ...
	// Determine environment (default: development)
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	// Load .env.{APP_ENV} first, then fallback to .env
	envFile := fmt.Sprintf(".env.%s", appEnv)
	envPaths := []string{envFile, "../" + envFile, "../../" + envFile}
	// Also try generic .env as fallback
	envFallback := []string{".env", "../.env", "../../.env"}

	envLoaded := false
	for _, p := range envPaths {
		if err := godotenv.Load(p); err == nil {
			fmt.Printf("Loaded env from: %s (APP_ENV=%s)\n", p, appEnv)
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		for _, p := range envFallback {
			if err := godotenv.Load(p); err == nil {
				fmt.Printf("Loaded env from: %s (fallback, APP_ENV=%s)\n", p, appEnv)
				envLoaded = true
				break
			}
		}
	}
	if !envLoaded {
		fmt.Println("Warning: .env file not found in any standard location")
	}

	// Debug: print key env vars to verify loading
	geminiKey := os.Getenv("GEMINI_API_KEY")
	keyPreview := ""
	if len(geminiKey) >= 10 {
		keyPreview = geminiKey[:10]
	}
	fmt.Printf("DEBUG ENV: SERVER_PORT=%s, POSTGRES_HOST=%s, GEMINI_API_KEY=%s..., OLLAMA_HOST=%s\n",
		os.Getenv("SERVER_PORT"),
		os.Getenv("POSTGRES_HOST"),
		keyPreview,
		os.Getenv("OLLAMA_HOST"),
	)

	// Setup logger with rotation
	zerolog.TimeFieldFormat = time.RFC3339

	// Ensure logs directory exists
	if err := os.MkdirAll("logs", 0755); err != nil {
		fmt.Printf("Failed to create logs directory: %v\n", err)
	}

	// Configure log rotation
	logFile := "logs/app-%Y-%m-%d-%H.log"
	rotator, err := rotatelogs.New(
		logFile,
		rotatelogs.WithRotationTime(time.Hour),
		rotatelogs.WithMaxAge(7*24*time.Hour), // Keep logs for 7 days
	)
	if err != nil {
		fmt.Printf("Failed to initialize log rotation: %v\n", err)
	}

	// Multi-writer: Console + File
	// Console writer (pretty print for dev)
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr}

	if rotator != nil {
		multi := zerolog.MultiLevelWriter(consoleWriter, rotator)
		log.Logger = zerolog.New(multi).With().Timestamp().Logger()
	} else {
		log.Logger = log.Output(consoleWriter)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	log.Info().
		Str("host", cfg.Server.Host).
		Int("port", cfg.Server.Port).
		Msg("Starting Text-to-SQL API server")

	log.Info().Msg("Made by Rendy Yusuf (https://www.linkedin.com/in/rendy-yusuf)")

	// Initialize database
	db, err := postgres.NewDB(context.Background(), cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Run database migrations
	migrationSource := "file://./migrations" // Default relative path
	if os.Getenv("MIGRATION_SOURCE") != "" {
		migrationSource = os.Getenv("MIGRATION_SOURCE")
	}
	// In Docker, we copy migrations to /app/migrations
	if _, err := os.Stat("/app/migrations"); err == nil {
		migrationSource = "file:///app/migrations"
	}

	log.Info().Msgf("Running migrations from %s", migrationSource)
	if err := postgres.RunMigrations(cfg.Database.DSN(), migrationSource); err != nil {
		log.Fatal().Err(err).Msg("Failed to run database migrations")
	}

	// Initialize Redis
	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer redisClient.Close()

	// Initialize router
	router := api.NewRouter(cfg, db, redisClient)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Info().Msgf("Server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped")
}
