package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Rrens/text-to-sql/internal/api/handler"
	customMiddleware "github.com/Rrens/text-to-sql/internal/api/middleware"
	"github.com/Rrens/text-to-sql/internal/config"
	"github.com/Rrens/text-to-sql/internal/llm"
	"github.com/Rrens/text-to-sql/internal/llm/anthropic"
	"github.com/Rrens/text-to-sql/internal/llm/deepseek"
	"github.com/Rrens/text-to-sql/internal/llm/gemini"
	"github.com/Rrens/text-to-sql/internal/llm/ollama"
	"github.com/Rrens/text-to-sql/internal/llm/openai"
	"github.com/Rrens/text-to-sql/internal/mcp"
	mcpClickhouse "github.com/Rrens/text-to-sql/internal/mcp/clickhouse"
	mcpMongo "github.com/Rrens/text-to-sql/internal/mcp/mongo"
	mcpMySQL "github.com/Rrens/text-to-sql/internal/mcp/mysql"
	mcpPostgres "github.com/Rrens/text-to-sql/internal/mcp/postgres"
	mcpSQLite "github.com/Rrens/text-to-sql/internal/mcp/sqlite"
	"github.com/Rrens/text-to-sql/internal/repository/postgres"
	"github.com/Rrens/text-to-sql/internal/repository/redis"
	"github.com/Rrens/text-to-sql/internal/security"
	"github.com/Rrens/text-to-sql/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/log"
)

// NewRouter creates and configures the HTTP router

func NewRouter(cfg *config.Config, db *postgres.DB, redisClient *redis.Client) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customMiddleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.Server.MiddlewareTimeout))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Workspace-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Initialize security components
	jwtManager := security.NewJWTManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.RefreshTokenTTL,
	)

	// Initialize encryptor
	encryptionKey := []byte(cfg.Auth.JWTSecret)
	if len(encryptionKey) > 32 {
		encryptionKey = encryptionKey[:32]
	} else if len(encryptionKey) < 32 {
		padded := make([]byte, 32)
		copy(padded, encryptionKey)
		encryptionKey = padded
	}
	encryptor, _ := security.NewEncryptor(encryptionKey)

	// Initialize repositories
	userRepo := postgres.NewUserRepository(db)
	workspaceRepo := postgres.NewWorkspaceRepository(db)
	connectionRepo := postgres.NewConnectionRepository(db)
	messageRepo := postgres.NewMessageRepository(db.Pool)
	sessionRepo := postgres.NewSessionRepository(db.Pool)

	// Initialize rate limiter and schema cache
	rateLimiter := redis.NewRateLimiter(
		redisClient,
		cfg.Security.RateLimit.RequestsPerMinute,
		cfg.Security.RateLimit.Burst,
	)
	schemaCache := redis.NewSchemaCache(redisClient)

	// Initialize MCP Router with database adapters
	mcpRouter := mcp.NewRouter()
	mcpRouter.RegisterAdapter("postgres", mcpPostgres.NewAdapter)
	mcpRouter.RegisterAdapter("clickhouse", mcpClickhouse.NewAdapter)
	mcpRouter.RegisterAdapter("mysql", mcpMySQL.NewAdapter)
	mcpRouter.RegisterAdapter("mongodb", mcpMongo.NewAdapter)
	mcpRouter.RegisterAdapter("sqlite", mcpSQLite.NewAdapter)

	// Initialize LLM Router with providers
	llmRouter := llm.NewRouter(cfg.LLM.DefaultProvider)

	// Register LLM providers and factories
	log.Info().Msgf("Initializing LLM providers. Default: %s", cfg.LLM.DefaultProvider)

	// Ollama Factory
	llmRouter.RegisterFactory("ollama", func(config map[string]any) (llm.Provider, error) {
		host, _ := config["host"].(string)
		model, _ := config["model"].(string)
		if host == "" {
			host = cfg.LLM.Ollama.Host
		}
		if model == "" {
			model = cfg.LLM.Ollama.DefaultModel
		}
		return ollama.NewProvider(host, model), nil
	})

	// OpenAI Factory
	llmRouter.RegisterFactory("openai", func(config map[string]any) (llm.Provider, error) {
		apiKey, _ := config["api_key"].(string)
		model, _ := config["model"].(string)
		if apiKey == "" {
			apiKey = cfg.LLM.OpenAI.APIKey
		}
		if model == "" {
			model = cfg.LLM.OpenAI.Model
		}
		return openai.NewProvider(apiKey, model), nil
	})

	// Anthropic Factory
	llmRouter.RegisterFactory("anthropic", func(config map[string]any) (llm.Provider, error) {
		apiKey, _ := config["api_key"].(string)
		model, _ := config["model"].(string)
		if apiKey == "" {
			apiKey = cfg.LLM.Anthropic.APIKey
		}
		if model == "" {
			model = cfg.LLM.Anthropic.Model
		}
		return anthropic.NewProvider(apiKey, model), nil
	})

	// DeepSeek Factory
	llmRouter.RegisterFactory("deepseek", func(config map[string]any) (llm.Provider, error) {
		apiKey, _ := config["api_key"].(string)
		model, _ := config["model"].(string)
		if apiKey == "" {
			apiKey = cfg.LLM.DeepSeek.APIKey
		}
		if model == "" {
			model = cfg.LLM.DeepSeek.Model
		}
		return deepseek.NewProvider(apiKey, model), nil
	})

	// Register default/system instances
	if cfg.LLM.Ollama.Host != "" {
		log.Info().Str("host", cfg.LLM.Ollama.Host).Msg("Registering Ollama provider")
		llmRouter.RegisterProvider(ollama.NewProvider(cfg.LLM.Ollama.Host, cfg.LLM.Ollama.DefaultModel))
	}
	if cfg.LLM.OpenAI.APIKey != "" {
		llmRouter.RegisterProvider(openai.NewProvider(cfg.LLM.OpenAI.APIKey, cfg.LLM.OpenAI.Model))
	}
	if cfg.LLM.Anthropic.APIKey != "" {
		llmRouter.RegisterProvider(anthropic.NewProvider(cfg.LLM.Anthropic.APIKey, cfg.LLM.Anthropic.Model))
	}
	if cfg.LLM.DeepSeek.APIKey != "" {
		llmRouter.RegisterProvider(deepseek.NewProvider(cfg.LLM.DeepSeek.APIKey, cfg.LLM.DeepSeek.Model))
	}
	if cfg.LLM.Gemini.APIKey != "" {
		log.Info().Str("key_len", fmt.Sprintf("%d", len(cfg.LLM.Gemini.APIKey))).Msg("Registering Gemini provider")
		llmRouter.RegisterProvider(gemini.NewProvider(cfg.LLM.Gemini))
	} else {
		log.Warn().Msg("Gemini API Key is empty, skipping registration")
	}

	// Initialize services
	authService := service.NewAuthService(userRepo, workspaceRepo, jwtManager)
	workspaceService := service.NewWorkspaceService(workspaceRepo)
	connectionService := service.NewConnectionService(
		connectionRepo,
		workspaceRepo,
		encryptor,
		mcpRouter,
		cfg.Security.MaxRows,
		int(cfg.Security.QueryTimeout.Seconds()),
	)
	queryService := service.NewQueryService(
		connectionService,
		mcpRouter,
		llmRouter,
		schemaCache,
		messageRepo,
		sessionRepo,
		userRepo,
	)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	workspaceHandler := handler.NewWorkspaceHandler(workspaceService)
	connectionHandler := handler.NewConnectionHandler(connectionService)
	queryHandler := handler.NewQueryHandler(queryService)
	uploadHandler := handler.NewUploadHandler("data/sqlite")

	// Auth middleware
	authMiddleware := customMiddleware.NewAuthMiddleware(jwtManager)
	rateLimitMiddleware := customMiddleware.NewRateLimitMiddleware(rateLimiter)

	// Public routes
	r.Route("/api/v1", func(r chi.Router) {
		// Health check
		r.Get("/health", handler.HealthCheck)
		r.Get("/ready", handler.ReadyCheck(db))

		// Auth routes (public)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Use(rateLimitMiddleware.Limit)

			// Auth check
			r.Get("/auth/me", authHandler.Me)
			r.Patch("/auth/me/llm-config", authHandler.UpdateLLMConfig)

			// LLM providers
			r.Get("/llm-providers", handler.ListLLMProviders(cfg))

			// Cache management
			r.Post("/cache/flush", handler.FlushCache(schemaCache))

			// Workspace routes
			r.Route("/workspaces", func(r chi.Router) {
				r.Get("/", workspaceHandler.List)
				r.Post("/", workspaceHandler.Create)

				r.Route("/{workspaceID}", func(r chi.Router) {
					r.Use(customMiddleware.WorkspaceContext)

					r.Get("/", workspaceHandler.Get)
					r.Patch("/", workspaceHandler.Update)
					r.Delete("/", workspaceHandler.Delete)

					// Query endpoints
					r.Post("/query", queryHandler.Execute)
					r.Post("/generate", queryHandler.Generate)

					// Session Management
					sessionHandler := handler.NewSessionHandler(queryService)
					r.Route("/sessions", func(r chi.Router) {
						r.Get("/", sessionHandler.List)
						r.Post("/", sessionHandler.Create)
						r.Route("/{sessionID}", func(r chi.Router) {
							r.Get("/", sessionHandler.GetHistory) // Get history for session
							r.Delete("/", sessionHandler.Delete)
						})
					})

					// Suggested Questions
					suggestionHandler := handler.NewSuggestionHandler(queryService)
					r.Get("/suggestions", suggestionHandler.GetSuggestions)

					r.Get("/chat", queryHandler.GetHistory) // Legacy endpoint (optional)

					// Connection routes
					r.Route("/connections", func(r chi.Router) {
						r.Get("/", connectionHandler.List)
						r.Post("/", connectionHandler.Create)

						r.Route("/{connectionID}", func(r chi.Router) {
							r.Get("/", connectionHandler.Get)
							r.Patch("/", connectionHandler.Update)
							r.Delete("/", connectionHandler.Delete)
							r.Post("/test", connectionHandler.Test)
							r.Get("/schema", queryHandler.GetSchema)
							r.Post("/schema/refresh", queryHandler.RefreshSchema)
						})
					})

					// Upload routes
					r.Post("/upload-sqlite", uploadHandler.UploadSQLite)
				})
			})
		})
	})

	// Serve Frontend (SPA)
	workDir, _ := os.Getwd()
	frontendDir := filepath.Join(workDir, "frontend")
	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		// Try /app/frontend (docker default)
		frontendDir = "/app/frontend"
	}

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		fs := http.FileServer(http.Dir(frontendDir))

		// Check if file exists
		path := filepath.Join(frontendDir, r.URL.Path)
		_, err := os.Stat(path)

		if os.IsNotExist(err) {
			// If file not found (and not /api), serve index.html
			http.ServeFile(w, r, filepath.Join(frontendDir, "index.html"))
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Verify it's not a directory without index (though FileServer handles it, we want SPA)
		// Usually FileServer handles directories by looking for index.html or listing.
		// We rely on standard FileServer for assets.
		fs.ServeHTTP(w, r)
	})

	return r
}
