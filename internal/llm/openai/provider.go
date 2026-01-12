package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Rrens/text-to-sql/internal/llm"
)

// Provider implements llm.Provider for OpenAI
type Provider struct {
	apiKey       string
	defaultModel string
	client       *http.Client
	baseURL      string
}

// NewProvider creates a new OpenAI provider
func NewProvider(apiKey, defaultModel string) llm.Provider {
	if defaultModel == "" {
		defaultModel = "gpt-4-turbo"
	}
	return &Provider{
		apiKey:       apiKey,
		defaultModel: defaultModel,
		client:       &http.Client{Timeout: 120 * time.Second},
		baseURL:      "https://api.openai.com/v1",
	}
}

// Name returns the provider identifier
func (p *Provider) Name() string {
	return "openai"
}

// AvailableModels returns list of supported models
func (p *Provider) AvailableModels() []string {
	return []string{
		"gpt-4-turbo",
		"gpt-4",
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-3.5-turbo",
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

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// GenerateSQL generates SQL from natural language
func (p *Provider) GenerateSQL(ctx context.Context, req llm.Request, model string) (*llm.Response, error) {
	if model == "" {
		model = p.defaultModel
	}

	prompt := llm.BuildPrompt(req)

	chatReq := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: "You are an expert SQL query generator. Respond with ONLY the SQL query, no explanations or markdown formatting.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0,
		MaxTokens:   2048,
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	start := time.Now()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai returned status %d", resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	latencyMs := time.Since(start).Milliseconds()
	sql := llm.ExtractSQL(chatResp.Choices[0].Message.Content)

	return &llm.Response{
		SQL:        sql,
		Model:      model,
		TokensUsed: chatResp.Usage.TotalTokens,
		LatencyMs:  latencyMs,
	}, nil
}
