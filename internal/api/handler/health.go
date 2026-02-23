package handler

import (
	"net/http"

	"github.com/Rrens/text-to-sql/internal/api/response"
	"github.com/Rrens/text-to-sql/internal/config"
	"github.com/Rrens/text-to-sql/internal/repository/postgres"
	"github.com/Rrens/text-to-sql/internal/repository/redis"
)

// HealthCheck returns a simple health check response
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{
		"status": "ok",
	})
}

// ReadyCheck returns readiness status including database connectivity
func ReadyCheck(db *postgres.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			response.Error(w, http.StatusServiceUnavailable, "database not ready")
			return
		}

		response.OK(w, map[string]string{
			"status": "ready",
		})
	}
}

// ListLLMProviders returns available LLM providers
// Always returns all providers since users can store their own API keys in DB
func ListLLMProviders(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providers := []map[string]any{
			{
				"name":       "ollama",
				"models":     []string{"qwen2.5-coder:7b", "qwen2.5-coder:1.5b", "llama3", "codellama", "sqlcoder", "deepseek-coder"},
				"default":    cfg.LLM.DefaultProvider == "ollama",
				"configured": cfg.LLM.Ollama.Host != "",
				"host":       cfg.LLM.Ollama.Host,
			},
			{
				"name":       "gemini",
				"models":     []string{"gemini-2.5-flash", "gemini-1.5-flash", "gemini-1.5-pro", "gemini-1.0-pro"},
				"default":    cfg.LLM.DefaultProvider == "gemini",
				"configured": cfg.LLM.Gemini.APIKey != "",
			},
			{
				"name":       "openai",
				"models":     []string{"gpt-4-turbo", "gpt-4", "gpt-3.5-turbo"},
				"default":    cfg.LLM.DefaultProvider == "openai",
				"configured": cfg.LLM.OpenAI.APIKey != "",
			},
			{
				"name":       "anthropic",
				"models":     []string{"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"},
				"default":    cfg.LLM.DefaultProvider == "anthropic",
				"configured": cfg.LLM.Anthropic.APIKey != "",
			},
			{
				"name":       "deepseek",
				"models":     []string{"deepseek-chat", "deepseek-coder"},
				"default":    cfg.LLM.DefaultProvider == "deepseek",
				"configured": cfg.LLM.DeepSeek.APIKey != "",
			},
		}

		response.OK(w, map[string]any{
			"providers":        providers,
			"default_provider": cfg.LLM.DefaultProvider,
		})
	}
}

// FlushCache clears all schema cache from Redis
func FlushCache(schemaCache *redis.SchemaCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deleted, err := schemaCache.FlushAll(r.Context())
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to flush cache: "+err.Error())
			return
		}

		response.OK(w, map[string]any{
			"message":      "cache flushed successfully",
			"keys_deleted": deleted,
		})
	}
}
