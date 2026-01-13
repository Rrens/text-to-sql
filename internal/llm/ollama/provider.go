package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
			"num_predict": 4096,  // Max output tokens
			"num_ctx":     16384, // Max context window (input + output)
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

	var bodyBytes []byte
	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	// Log the raw body for debugging
	fmt.Printf("DEBUG OLLAMA RAW RESPONSE: %s\n", string(bodyBytes))

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(bodyBytes, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	latencyMs := time.Since(start).Milliseconds()
	sql := llm.ExtractSQL(ollamaResp.Response)

	// If SQL extraction failed, or even if it succeeded, it's useful to have the full text as explanation
	// especially for debugging empty SQL issues.
	explanation := ollamaResp.Response
	if sql != "" {
		// If we successfully extracted SQL, maybe we want to keep explanation cleaner?
		// But for now, let's keep it simple and just return the raw text if needed.
		// Or better: if sql is found, try to remove it from explanation?
		// For debugging "empty sql" issue, raw response is critical.
	}

	return &llm.Response{
		SQL:         sql,
		Explanation: explanation,
		Model:       model,
		TokensUsed:  ollamaResp.EvalCount,
		LatencyMs:   latencyMs,
	}, nil
}

// GenerateTitle generates a short title for the chat session
func (p *Provider) GenerateTitle(ctx context.Context, question string, model string) (string, error) {
	if model == "" {
		model = p.defaultModel
	}

	// Use a simpler model for title generation if possible, or same model
	prompt := fmt.Sprintf("Summarize the following user question into a very short, concise title (max 5 words). Do not use quotes or prefixes. Question: %s", question)

	ollamaReq := ollamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
		Options: map[string]any{
			"temperature": 0.5, // Slightly higher temp for creativity but still focused
			"num_predict": 50,  // Short output
		},
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return "New Chat", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.host+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "New Chat", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "New Chat", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "New Chat", fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "New Chat", fmt.Errorf("failed to decode response: %w", err)
	}

	title := ollamaResp.Response
	// Clean up title (remove quotes, newlines)
	title = string(bytes.TrimSpace([]byte(title)))
	title = string(bytes.Trim([]byte(title), `"'`))

	if title == "" {
		return "New Chat", nil
	}

	return title, nil
}
