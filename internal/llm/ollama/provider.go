package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Rrens/text-to-sql/internal/llm"
)

// Provider implements llm.Provider for Ollama
type Provider struct {
	host         string
	defaultModel string
	client       *http.Client
}

// NewProvider creates a new Ollama provider
func NewProvider(host, defaultModel string) llm.Provider {
	if defaultModel == "" {
		defaultModel = "llama3"
	}
	return &Provider{
		host:         host,
		defaultModel: defaultModel,
		client:       &http.Client{Timeout: 300 * time.Second},
	}
}

// Name returns the provider identifier
func (p *Provider) Name() string {
	return "ollama"
}

// AvailableModels returns list of supported models
func (p *Provider) AvailableModels() []string {
	return []string{
		"llama3",
		"llama3.1",
		"llama3.2",
		"codellama",
		"sqlcoder",
		"deepseek-coder",
		"mistral",
		"mixtral",
		"phi3",
		"qwen2",
	}
}

// DefaultModel returns the default model
func (p *Provider) DefaultModel() string {
	return p.defaultModel
}

// IsConfigured checks if provider has valid credentials
func (p *Provider) IsConfigured() bool {
	return p.host != ""
}

type ollamaRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stream  bool           `json:"stream"`
	Options map[string]any `json:"options,omitempty"`
}

type ollamaResponse struct {
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	EvalCount int    `json:"eval_count"`
}

// GenerateSQL generates SQL from natural language
func (p *Provider) GenerateSQL(ctx context.Context, req llm.Request, model string) (*llm.Response, error) {
	if model == "" {
		model = p.defaultModel
	}

	prompt := llm.BuildPrompt(req)

	ollamaReq := ollamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
		Options: map[string]any{
			"temperature": 0.0,
			"num_predict": 4096, // Increased for thinking models like Qwen
		},
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	start := time.Now()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.host+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	latencyMs := time.Since(start).Milliseconds()
	sql := llm.ExtractSQL(ollamaResp.Response)

	return &llm.Response{
		SQL:        sql,
		Model:      model,
		TokensUsed: ollamaResp.EvalCount,
		LatencyMs:  latencyMs,
	}, nil
}
