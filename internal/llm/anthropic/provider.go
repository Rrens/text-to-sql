package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Rrens/text-to-sql/internal/llm"
)

// Provider implements llm.Provider for Anthropic
type Provider struct {
	apiKey       string
	defaultModel string
	client       *http.Client
	baseURL      string
}

// NewProvider creates a new Anthropic provider
func NewProvider(apiKey, defaultModel string) llm.Provider {
	if defaultModel == "" {
		defaultModel = "claude-3-sonnet-20240229"
	}
	return &Provider{
		apiKey:       apiKey,
		defaultModel: defaultModel,
		client:       &http.Client{Timeout: 120 * time.Second},
		baseURL:      "https://api.anthropic.com/v1",
	}
}

// Name returns the provider identifier
func (p *Provider) Name() string {
	return "anthropic"
}

// AvailableModels returns list of supported models
func (p *Provider) AvailableModels() []string {
	return []string{
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
		"claude-3-5-sonnet-20241022",
	}
}

// DefaultModel returns the default model
func (p *Provider) DefaultModel() string {
	return p.defaultModel
}

// IsConfigured checks if provider has valid credentials
func (p *Provider) IsConfigured() bool {
	return p.apiKey != ""
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// GenerateSQL generates SQL from natural language
func (p *Provider) GenerateSQL(ctx context.Context, req llm.Request, model string) (*llm.Response, error) {
	if model == "" {
		model = p.defaultModel
	}

	prompt := llm.BuildPrompt(req)

	anthropicReq := anthropicRequest{
		Model:     model,
		MaxTokens: 2048,
		System:    "You are an expert SQL query generator. Respond with ONLY the SQL query, no explanations or markdown formatting.",
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	start := time.Now()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic returned status %d", resp.StatusCode)
	}

	var anthropicResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return nil, fmt.Errorf("no response from Anthropic")
	}

	latencyMs := time.Since(start).Milliseconds()
	sql := llm.ExtractSQL(anthropicResp.Content[0].Text)
	totalTokens := anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens

	return &llm.Response{
		SQL:        sql,
		Model:      model,
		TokensUsed: totalTokens,
		LatencyMs:  latencyMs,
	}, nil
}

func (p *Provider) GenerateTitle(ctx context.Context, question string, model string) (string, error) {
	return "New Chat", nil // Stub
}
