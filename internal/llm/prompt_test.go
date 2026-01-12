package llm_test

import (
	"testing"

	"github.com/Rrens/text-to-sql/internal/llm"
)

func TestBuildPrompt(t *testing.T) {
	req := llm.Request{
		Question:     "Show me all active users",
		SchemaDDL:    "CREATE TABLE users (id INT, name VARCHAR, active BOOLEAN);",
		SQLDialect:   "PostgreSQL SQL dialect with ILIKE, LIMIT/OFFSET",
		DatabaseType: "postgres",
	}

	prompt := llm.BuildPrompt(req)

	// Check that prompt contains key elements
	mustContain := []string{
		"postgres",
		"Show me all active users",
		"CREATE TABLE users",
		"SELECT statements",
		"LIMIT",
	}

	for _, s := range mustContain {
		if !contains(prompt, s) {
			t.Errorf("prompt should contain %q", s)
		}
	}
}

func TestBuildPrompt_WithExamples(t *testing.T) {
	req := llm.Request{
		Question:     "Count users by status",
		SchemaDDL:    "CREATE TABLE users (id INT, status VARCHAR);",
		DatabaseType: "postgres",
		Examples: []llm.Example{
			{
				Question: "Get all users",
				SQL:      "SELECT * FROM users",
			},
			{
				Question: "Count total users",
				SQL:      "SELECT COUNT(*) FROM users",
			},
		},
	}

	prompt := llm.BuildPrompt(req)

	// Check examples are included
	mustContain := []string{
		"Get all users",
		"SELECT * FROM users",
		"Count total users",
		"SELECT COUNT(*) FROM users",
	}

	for _, s := range mustContain {
		if !contains(prompt, s) {
			t.Errorf("prompt should contain example %q", s)
		}
	}
}

func TestExtractSQL(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			"plain sql",
			"SELECT * FROM users",
			"SELECT * FROM users",
		},
		{
			"sql with semicolon",
			"SELECT * FROM users;",
			"SELECT * FROM users",
		},
		{
			"sql in code block",
			"```sql\nSELECT * FROM users\n```",
			"SELECT * FROM users",
		},
		{
			"sql in generic code block",
			"```\nSELECT * FROM users\n```",
			"SELECT * FROM users",
		},
		{
			"sql with explanation before",
			"Here is the query:\n```sql\nSELECT * FROM users\n```",
			"SELECT * FROM users",
		},
		{
			"sql with whitespace",
			"  SELECT * FROM users  ",
			"SELECT * FROM users",
		},
		{
			"complex query",
			"```sql\nSELECT u.id, COUNT(o.id) as order_count\nFROM users u\nLEFT JOIN orders o ON u.id = o.user_id\nGROUP BY u.id\nORDER BY order_count DESC\nLIMIT 10\n```",
			"SELECT u.id, COUNT(o.id) as order_count\nFROM users u\nLEFT JOIN orders o ON u.id = o.user_id\nGROUP BY u.id\nORDER BY order_count DESC\nLIMIT 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := llm.ExtractSQL(tt.content)
			if result != tt.expected {
				t.Errorf("ExtractSQL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
