package handler

import (
	"net/http"

	"github.com/rensmac/text-to-sql/internal/api/response"
	"github.com/rensmac/text-to-sql/internal/config"
	"github.com/rensmac/text-to-sql/internal/repository/postgres"
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
func ListLLMProviders(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providers := []map[string]any{}

		if cfg.LLM.OpenAI.APIKey != "" {
			providers = append(providers, map[string]any{
				"name":    "openai",
				"models":  []string{"gpt-4-turbo", "gpt-4", "gpt-3.5-turbo"},
				"default": cfg.LLM.DefaultProvider == "openai",
			})
		}

		if cfg.LLM.Anthropic.APIKey != "" {
			providers = append(providers, map[string]any{
				"name":    "anthropic",
				"models":  []string{"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"},
				"default": cfg.LLM.DefaultProvider == "anthropic",
			})
		}

		if cfg.LLM.Ollama.Host != "" {
			providers = append(providers, map[string]any{
				"name":    "ollama",
				"models":  []string{"llama3", "codellama", "sqlcoder", "deepseek-coder"},
				"default": cfg.LLM.DefaultProvider == "ollama",
				"host":    cfg.LLM.Ollama.Host,
			})
		}

		if cfg.LLM.DeepSeek.APIKey != "" {
			providers = append(providers, map[string]any{
				"name":    "deepseek",
				"models":  []string{"deepseek-chat", "deepseek-coder"},
				"default": cfg.LLM.DefaultProvider == "deepseek",
			})
		}

		response.OK(w, map[string]any{
			"providers":        providers,
			"default_provider": cfg.LLM.DefaultProvider,
		})
	}
}
