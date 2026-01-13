package gemini

import (
	"context"
	"fmt"
	"time"

	"github.com/Rrens/text-to-sql/internal/config"
	"github.com/Rrens/text-to-sql/internal/domain"
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

	// Convert history to Gemini format
	var history []*genai.Content
	for _, msg := range req.History {
		role := "user"
		if msg.Role == domain.RoleAssistant {
			role = "model"
		}
		history = append(history, &genai.Content{
			Role:  role,
			Parts: []genai.Part{genai.Text(msg.Content)},
		})
	}

	// Create chat session with history
	cs := generativeModel.StartChat()
	cs.History = history

	start := time.Now()
	// Use SendMessage instead of GenerateContent for chat
	resp, err := cs.SendMessage(ctx, genai.Text(prompt))
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

// GenerateTitle generates a short title for the chat session
func (p *Provider) GenerateTitle(ctx context.Context, question string, model string) (string, error) {
	if model == "" {
		model = p.DefaultModel()
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(p.apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create gemini client: %w", err)
	}
	defer client.Close()

	genModel := client.GenerativeModel(model)

	prompt := fmt.Sprintf("Summarize the following user question into a very short, concise title (max 5 words). Do not use quotes or prefixes. Question: %s", question)
	resp, err := genModel.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate title: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "New Chat", nil
	}

	var title string
	if text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		title = string(text)
	} else {
		return "New Chat", nil
	}

	return title, nil
}
