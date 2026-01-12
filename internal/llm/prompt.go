package llm

import "fmt"

// BuildPrompt creates a prompt for SQL generation
func BuildPrompt(req Request) string {
	examplesStr := ""
	if len(req.Examples) > 0 {
		examplesStr = "\n\nExamples:\n"
		for _, ex := range req.Examples {
			examplesStr += fmt.Sprintf("Question: %s\nSQL: %s\n\n", ex.Question, ex.SQL)
		}
	}

	return fmt.Sprintf(`You are an expert SQL query generator for %s databases.

%s

Rules:
1. Generate ONLY the SQL query, no explanations or markdown
2. Use only SELECT statements (no INSERT, UPDATE, DELETE, DROP, etc.)
3. Always include appropriate LIMIT clauses for safety
4. Use only tables and columns from the provided schema
5. Handle NULL values appropriately
6. Use proper date/time functions for the database dialect
7. Prefer explicit column names over SELECT *

Database Schema:
%s
%s
Question: %s

SQL:`, req.DatabaseType, req.SQLDialect, req.SchemaDDL, examplesStr, req.Question)
}

// ExtractSQL extracts SQL from LLM response
func ExtractSQL(content string) string {
	// Try to extract from markdown code blocks
	if sql := extractFromCodeBlock(content, "```sql", "```"); sql != "" {
		return sql
	}
	if sql := extractFromCodeBlock(content, "```", "```"); sql != "" {
		return sql
	}

	// Return trimmed content
	return trimSQL(content)
}

func extractFromCodeBlock(content, startMarker, endMarker string) string {
	startIdx := indexOf(content, startMarker)
	if startIdx == -1 {
		return ""
	}

	contentStart := startIdx + len(startMarker)
	// Skip newline after marker
	if contentStart < len(content) && content[contentStart] == '\n' {
		contentStart++
	}

	endIdx := indexOfFrom(content, endMarker, contentStart)
	if endIdx == -1 {
		return ""
	}

	return trimSQL(content[contentStart:endIdx])
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func indexOfFrom(s, substr string, from int) int {
	for i := from; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func trimSQL(sql string) string {
	// Trim whitespace
	sql = trimWhitespace(sql)

	// Remove trailing semicolon for consistency
	if len(sql) > 0 && sql[len(sql)-1] == ';' {
		sql = sql[:len(sql)-1]
	}

	return trimWhitespace(sql)
}

func trimWhitespace(s string) string {
	start := 0
	for start < len(s) && isWhitespace(s[start]) {
		start++
	}

	end := len(s)
	for end > start && isWhitespace(s[end-1]) {
		end--
	}

	return s[start:end]
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
