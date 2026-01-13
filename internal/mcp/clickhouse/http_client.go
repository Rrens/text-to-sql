package clickhouse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPClient wraps HTTP communication with ClickHouse
type HTTPClient struct {
	baseURL  string
	username string
	password string
	database string
	client   *http.Client
}

// NewHTTPClient creates a new ClickHouse HTTP client
func NewHTTPClient(host string, port int, database, username, password string) *HTTPClient {
	return &HTTPClient{
		baseURL:  fmt.Sprintf("http://%s:%d", host, port),
		username: username,
		password: password,
		database: database,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Ping tests the connection
func (c *HTTPClient) Ping(ctx context.Context) error {
	_, err := c.Query(ctx, "SELECT 1")
	return err
}

// Query executes a query and returns results as JSON
func (c *HTTPClient) Query(ctx context.Context, query string) ([]map[string]interface{}, error) {
	// Add FORMAT JSONEachRow to get JSON output
	if !strings.Contains(strings.ToUpper(query), "FORMAT") {
		query = query + " FORMAT JSONEachRow"
	}

	body, err := c.execute(ctx, query)
	if err != nil {
		return nil, err
	}

	// Parse JSON lines
	var results []map[string]interface{}
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var row map[string]interface{}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
		results = append(results, row)
	}

	return results, nil
}

// QueryRaw executes a query and returns raw response
func (c *HTTPClient) QueryRaw(ctx context.Context, query string) ([]byte, error) {
	return c.execute(ctx, query)
}

// execute sends query to ClickHouse and returns raw response
func (c *HTTPClient) execute(ctx context.Context, query string) ([]byte, error) {
	// Build URL with query parameters
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("database", c.database)
	u.RawQuery = q.Encode()

	// Create request with query in body
	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewBufferString(query))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication headers
	req.Header.Set("X-ClickHouse-User", c.username)
	req.Header.Set("X-ClickHouse-Key", c.password)
	req.Header.Set("Content-Type", "text/plain")

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ClickHouse error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// Close closes the HTTP client
func (c *HTTPClient) Close() error {
	c.client.CloseIdleConnections()
	return nil
}
