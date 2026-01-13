package llm

import (
	"fmt"
	"strings"

	"github.com/Rrens/text-to-sql/internal/domain"
)

// BuildPrompt creates a prompt for SQL generation
func BuildPrompt(req Request) string {
	examplesStr := ""
	if len(req.Examples) > 0 {
		examplesStr = "\n\nExamples:\n"
		for _, ex := range req.Examples {
			examplesStr += fmt.Sprintf("Question: %s\nSQL: %s\n\n", ex.Question, ex.SQL)
		}
	}

	historyStr := ""
	if len(req.History) > 0 {
		var sb strings.Builder
		sb.WriteString("\n\nChat History:\n")
		for _, msg := range req.History {
			role := "User"
			if msg.Role == domain.RoleAssistant {
				role = "Assistant"
			}
			content := msg.Content
			if msg.Role == domain.RoleAssistant && msg.SQL != "" {
				content = fmt.Sprintf("```sql\n%s\n```", msg.SQL)
			}
			sb.WriteString(fmt.Sprintf("%s: %s\n", role, content))
		}
		historyStr = sb.String()
	}

	return fmt.Sprintf(`You are an expert SQL query generator for %s databases, but you are also a helpful assistant.
	
%s

Rules:
1. If the user asks a question that requires data from the database, generate ONLY the SQL query.
2. If the user sends a greeting, asks a clarification question, or says something that doesn't require a database query, respond naturally in plain text.
3. For SQL queries:
   - Use only SELECT statements (no INSERT, UPDATE, DELETE, DROP, etc.)
   - Always include appropriate LIMIT clauses for safety
   - Use only tables and columns from the provided schema
   - Handle NULL values appropriately
   - Use proper date/time functions for the database dialect
   - Prefer explicit column names over SELECT *
4. If you generate SQL, wrap it in a markdown code block like this:
   `+"```sql"+`
   SELECT ...
   `+"```"+`
5. If you cannot answer the question based on the schema, explain why.

Database Schema:
%s
%s
%s
Question: %s

Response:`, req.DatabaseType, req.SQLDialect, req.SchemaDDL, examplesStr, historyStr, req.Question)
}

// ExtractSQL extracts SQL from LLM response
func ExtractSQL(content string) string {
	// First, remove any <think>...</think> sections (used by Qwen and similar models)
	content = removeThinkingTags(content)

	// Try to extract from markdown code blocks
	if sql := extractFromCodeBlock(content, "```sql", "```"); sql != "" {
		return sql
	}
	if sql := extractFromCodeBlock(content, "```", "```"); sql != "" {
		return sql
	}

	// Try to find SQL starting with SELECT
	if sql := extractSelectStatement(content); sql != "" {
		return sql
	}

	// If no code block and no SELECT statement found, valid SQL is unlikely.
	// We check for common SQL keywords just in case, otherwise return empty.
	trimmed := trimSQL(content)
	upper := toUpper(trimmed)
	if startsWithAny(upper, []string{"SELECT", "WITH", "VALUES", "SHOW", "DESCRIBE", "EXPLAIN"}) {
		return trimmed
	}

	return ""
}

func startsWithAny(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if len(s) >= len(p) && s[:len(p)] == p {
			return true
		}
	}
	return false
}

// removeThinkingTags removes <think>...</think> sections from content
func removeThinkingTags(content string) string {
	for {
		startIdx := indexOf(content, "<think>")
		if startIdx == -1 {
			break
		}
		endIdx := indexOf(content, "</think>")
		if endIdx == -1 {
			// If no closing tag, remove everything from <think> onwards
			content = content[:startIdx]
			break
		}
		// Remove the entire <think>...</think> block
		content = content[:startIdx] + content[endIdx+len("</think>"):]
	}
	return trimWhitespace(content)
}

// extractSelectStatement finds and extracts a SELECT statement from content
func extractSelectStatement(content string) string {
	// Look for SELECT (case-insensitive)
	upperContent := toUpper(content)
	selectIdx := indexOf(upperContent, "SELECT")
	if selectIdx == -1 {
		return ""
	}

	// Extract from SELECT to the end or until we hit a stopping point
	sql := content[selectIdx:]

	// Find the end of the SQL (newline followed by non-SQL text, or end of string)
	// Look for double newlines as statement end
	endIdx := indexOf(sql, "\n\n")
	if endIdx != -1 {
		sql = sql[:endIdx]
	}

	return trimSQL(sql)
}

func toUpper(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			result[i] = c - 32
		} else {
			result[i] = c
		}
	}
	return string(result)
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
