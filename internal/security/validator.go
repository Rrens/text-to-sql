package security

import (
	"fmt"
	"regexp"
	"strings"
)

// SQLValidator validates SQL queries for safety
type SQLValidator struct {
	blockedPatterns []*regexp.Regexp
}

// NewSQLValidator creates a new SQL validator
func NewSQLValidator() *SQLValidator {
	patterns := []string{
		`(?i)\bINSERT\b`,
		`(?i)\bUPDATE\b`,
		`(?i)\bDELETE\b`,
		`(?i)\bDROP\b`,
		`(?i)\bTRUNCATE\b`,
		`(?i)\bALTER\b`,
		`(?i)\bCREATE\b`,
		`(?i)\bGRANT\b`,
		`(?i)\bREVOKE\b`,
		`(?i)\bEXEC\b`,
		`(?i)\bEXECUTE\b`,
		`(?i)\bCOPY\b`,
		`(?i)\bINTO\s+OUTFILE\b`,
		`(?i)\bINTO\s+DUMPFILE\b`,
		`(?i)\bLOAD_FILE\b`,
		`(?i)pg_read_file`,
		`(?i)pg_write_file`,
		`(?i)pg_ls_dir`,
		`(?i)lo_import`,
		`(?i)lo_export`,
		`(?i)dblink`,
		`(?i);\s*--`,                        // Comment after semicolon
		`(?i);\s*/\*`,                       // Block comment after semicolon
		`(?i)\bUNION\s+ALL\s+SELECT\s+NULL`, // Common SQL injection pattern
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		compiled = append(compiled, regexp.MustCompile(p))
	}

	return &SQLValidator{blockedPatterns: compiled}
}

// ValidationError represents a SQL validation error
type ValidationError struct {
	Message string
	Pattern string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Validate checks if a SQL query is safe to execute
func (v *SQLValidator) Validate(sql string) error {
	// Trim and normalize
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return &ValidationError{Message: "empty SQL query"}
	}

	// Check for multiple statements (basic check)
	if strings.Count(sql, ";") > 1 {
		return &ValidationError{Message: "multiple statements not allowed"}
	}

	// Must start with SELECT (or WITH for CTEs)
	normalized := strings.ToUpper(strings.TrimSpace(sql))
	if !strings.HasPrefix(normalized, "SELECT") && !strings.HasPrefix(normalized, "WITH") {
		return &ValidationError{Message: "only SELECT statements allowed"}
	}

	// Check blocked patterns
	for _, pattern := range v.blockedPatterns {
		if pattern.MatchString(sql) {
			return &ValidationError{
				Message: "blocked SQL pattern detected",
				Pattern: pattern.String(),
			}
		}
	}

	return nil
}

// EnforceLimit ensures the query has a LIMIT clause
func (v *SQLValidator) EnforceLimit(sql string, maxRows int) string {
	normalized := strings.ToUpper(sql)

	// Check if LIMIT already exists
	if strings.Contains(normalized, "LIMIT") {
		return sql
	}

	// Remove trailing semicolon if present
	sql = strings.TrimSuffix(strings.TrimSpace(sql), ";")

	return fmt.Sprintf("%s LIMIT %d", sql, maxRows)
}

// ValidateAndPrepare validates and prepares a SQL query for execution
func (v *SQLValidator) ValidateAndPrepare(sql string, maxRows int) (string, error) {
	if err := v.Validate(sql); err != nil {
		return "", err
	}
	return v.EnforceLimit(sql, maxRows), nil
}
