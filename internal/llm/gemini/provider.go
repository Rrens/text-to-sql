package gemini

import (
	"context"
	"fmt"
	"time"

	"github.com/Rrens/text-to-sql/internal/config"
	"github.com/Rrens/text-to-sql/internal/llm"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type Provider struct {
	apiKey string
	model  string
}

func NewProvider(cfg config.GeminiConfig) *Provider {
	return &Provider{
		apiKey: cfg.APIKey,
		model:  cfg.Model,
	}
}

func (p *Provider) Name() string {
	return "gemini"
}

func (p *Provider) AvailableModels() []string {
	return []string{
		"gemini-2.5-flash",
		"gemini-1.5-flash",
		"gemini-1.5-pro",
		"gemini-1.0-pro",
	}
}

func (p *Provider) DefaultModel() string {
	if p.model != "" {
		return p.model
	}
	return "gemini-2.5-flash"
}

func (p *Provider) IsConfigured() bool {
	return p.apiKey != ""
}

func (p *Provider) GenerateSQL(ctx context.Context, req llm.Request, model string) (*llm.Response, error) {
	if !p.IsConfigured() {
		return nil, fmt.Errorf("gemini provider is not configured (missing API key)")
	}

	if model == "" {
		model = p.DefaultModel()
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(p.apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}
	defer client.Close()

	generativeModel := client.GenerativeModel(model)
	// Set temperature to 0 for deterministic SQL generation
	var temperature float32 = 0.0
	generativeModel.Temperature = &temperature

	prompt := llm.BuildPrompt(req)

	start := time.Now()
	resp, err := generativeModel.GenerateContent(ctx, genai.Text(prompt))
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return nil, fmt.Errorf("gemini generation error: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from gemini")
	}

	var output string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			output += string(text)
		}
	}

	sql := llm.ExtractSQL(output)

	tokensUsed := 0
	if resp.UsageMetadata != nil {
		tokensUsed = int(resp.UsageMetadata.TotalTokenCount)
	}

	return &llm.Response{
		SQL:         sql,
		Explanation: output,
		Model:       model,
		TokensUsed:  tokensUsed,
		LatencyMs:   latency,
	}, nil
}
