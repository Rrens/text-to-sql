package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Rrens/text-to-sql/internal/api"
	"github.com/Rrens/text-to-sql/internal/config"
	"github.com/Rrens/text-to-sql/internal/repository/postgres"
	"github.com/Rrens/text-to-sql/internal/repository/redis"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load .env file - try multiple locations
	envPaths := []string{".env", "../.env", "../../.env"}
	envLoaded := false
	for _, p := range envPaths {
		if err := godotenv.Load(p); err == nil {
			fmt.Printf("Loaded .env from: %s\n", p)
			envLoaded = true
			break
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

	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("ENV") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
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

	// Initialize database
	db, err := postgres.NewDB(context.Background(), cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

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
