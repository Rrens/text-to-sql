package llm

import "context"

// Request contains text-to-SQL generation parameters
type Request struct {
	Question     string
	SchemaDDL    string
	SQLDialect   string
	DatabaseType string
	Examples     []Example
}

// Example represents a question-SQL pair for few-shot learning
type Example struct {
	Question string
	SQL      string
}

// Response contains LLM generation result
type Response struct {
	SQL         string
	Explanation string
	Model       string
	TokensUsed  int
	LatencyMs   int64
}

// Provider defines the interface for LLM providers
type Provider interface {
	// Name returns the provider identifier
	Name() string

	// AvailableModels returns list of supported models
	AvailableModels() []string

	// DefaultModel returns the default model
	DefaultModel() string

	// IsConfigured checks if provider has valid credentials
	IsConfigured() bool

	// GenerateSQL generates SQL from natural language
	GenerateSQL(ctx context.Context, req Request, model string) (*Response, error)
}

// ProviderFactory creates a new provider instance
type ProviderFactory func() Provider
