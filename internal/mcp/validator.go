package mcp

import (
	"fmt"
	"regexp"
	"strings"
)

// Common blocked SQL patterns across all databases
var blockedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bINSERT\b`),
	regexp.MustCompile(`(?i)\bUPDATE\b`),
	regexp.MustCompile(`(?i)\bDELETE\b`),
	regexp.MustCompile(`(?i)\bDROP\b`),
	regexp.MustCompile(`(?i)\bTRUNCATE\b`),
	regexp.MustCompile(`(?i)\bALTER\b`),
	regexp.MustCompile(`(?i)\bCREATE\b`),
	regexp.MustCompile(`(?i)\bGRANT\b`),
	regexp.MustCompile(`(?i)\bREVOKE\b`),
	regexp.MustCompile(`(?i)\bEXEC\b`),
	regexp.MustCompile(`(?i)\bEXECUTE\b`),
	regexp.MustCompile(`(?i)\bINTO\s+OUTFILE\b`),
	regexp.MustCompile(`(?i)\bINTO\s+DUMPFILE\b`),
	regexp.MustCompile(`(?i)\bLOAD_FILE\b`),
	regexp.MustCompile(`(?i)\bLOAD\s+DATA\b`),
}

// PostgreSQL specific blocked patterns
var PostgresBlockedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)pg_read_file`),
	regexp.MustCompile(`(?i)pg_write_file`),
	regexp.MustCompile(`(?i)pg_ls_dir`),
	regexp.MustCompile(`(?i)lo_import`),
	regexp.MustCompile(`(?i)lo_export`),
	regexp.MustCompile(`(?i)\bCOPY\b`),
	regexp.MustCompile(`(?i)dblink`),
}

// ClickHouse specific blocked patterns
var ClickhouseBlockedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)file\s*\(`),
	regexp.MustCompile(`(?i)url\s*\(`),
	regexp.MustCompile(`(?i)remote\s*\(`),
	regexp.MustCompile(`(?i)mysql\s*\(`),
	regexp.MustCompile(`(?i)postgresql\s*\(`),
}

// MySQL specific blocked patterns
var MysqlBlockedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)LOAD_FILE`),
	regexp.MustCompile(`(?i)INTO\s+OUTFILE`),
	regexp.MustCompile(`(?i)INTO\s+DUMPFILE`),
}

// SQLite specific blocked patterns
var SqliteBlockedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bATTACH\b`),
	regexp.MustCompile(`(?i)\bDETACH\b`),
	regexp.MustCompile(`(?i)load_extension`),
}

// ValidateSQL validates SQL for safety
func ValidateSQL(sql string, additionalPatterns []*regexp.Regexp) error {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return fmt.Errorf("empty SQL query")
	}

	// Check for multiple statements
	if strings.Count(sql, ";") > 1 {
		return fmt.Errorf("multiple statements not allowed")
	}

	// Must start with SELECT or WITH (for CTEs)
	normalized := strings.ToUpper(strings.TrimSpace(sql))
	if !strings.HasPrefix(normalized, "SELECT") && !strings.HasPrefix(normalized, "WITH") {
		return fmt.Errorf("only SELECT statements allowed")
	}

	// Check common blocked patterns
	for _, pattern := range blockedPatterns {
		if pattern.MatchString(sql) {
			return fmt.Errorf("blocked SQL pattern detected")
		}
	}

	// Check database-specific blocked patterns
	for _, pattern := range additionalPatterns {
		if pattern.MatchString(sql) {
			return fmt.Errorf("blocked SQL pattern detected")
		}
	}

	return nil
}

// EnforceLimit ensures the query has a LIMIT clause
func EnforceLimit(sql string, maxRows int, limitKeyword string) string {
	normalized := strings.ToUpper(sql)

	// Check if LIMIT already exists
	if strings.Contains(normalized, "LIMIT") {
		return sql
	}

	// Remove trailing semicolon if present
	sql = strings.TrimSuffix(strings.TrimSpace(sql), ";")

	return fmt.Sprintf("%s %s %d", sql, limitKeyword, maxRows)
}
