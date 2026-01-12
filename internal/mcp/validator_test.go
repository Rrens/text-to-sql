package mcp_test

import (
	"testing"

	"github.com/Rrens/text-to-sql/internal/mcp"
)

func TestValidateSQL(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		// Valid SELECT queries
		{"simple select", "SELECT * FROM users", false},
		{"select with where", "SELECT id FROM users WHERE active = true", false},
		{"select with join", "SELECT u.id FROM users u JOIN orders o ON u.id = o.user_id", false},
		{"cte", "WITH cte AS (SELECT * FROM users) SELECT * FROM cte", false},
		{"subquery", "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)", false},

		// Invalid - empty
		{"empty", "", true},
		{"whitespace", "   ", true},

		// Invalid - not SELECT
		{"insert", "INSERT INTO users VALUES (1)", true},
		{"update", "UPDATE users SET name = 'x'", true},
		{"delete", "DELETE FROM users", true},
		{"drop", "DROP TABLE users", true},
		{"truncate", "TRUNCATE users", true},
		{"alter", "ALTER TABLE users ADD col INT", true},
		{"create", "CREATE TABLE t (id INT)", true},
		{"grant", "GRANT SELECT ON users TO x", true},
		{"revoke", "REVOKE SELECT ON users FROM x", true},
		{"exec", "EXEC procedure", true},
		{"execute", "EXECUTE procedure", true},

		// Invalid - multiple statements
		{"multi statement", "SELECT 1; SELECT 2;", true},

		// Invalid - file operations
		{"into outfile", "SELECT * INTO OUTFILE '/tmp/x'", true},
		{"into dumpfile", "SELECT * INTO DUMPFILE '/tmp/x'", true},
		{"load_file", "SELECT LOAD_FILE('/etc/passwd')", true},
		{"load data", "LOAD DATA INFILE '/tmp/x' INTO TABLE t", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mcp.ValidateSQL(tt.sql, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSQL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSQL_PostgresPatterns(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{"pg_read_file", "SELECT pg_read_file('/etc/passwd')", true},
		{"pg_ls_dir", "SELECT pg_ls_dir('/tmp')", true},
		{"lo_import", "SELECT lo_import('/tmp/x')", true},
		{"lo_export", "SELECT lo_export(1234, '/tmp/x')", true},
		{"copy", "COPY users TO '/tmp/x'", true},
		{"dblink", "SELECT * FROM dblink('host=x', 'SELECT 1')", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mcp.ValidateSQL(tt.sql, mcp.PostgresBlockedPatterns)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSQL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSQL_ClickHousePatterns(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{"file function", "SELECT * FROM file('/tmp/x.csv')", true},
		{"url function", "SELECT * FROM url('http://x.com/data')", true},
		{"remote function", "SELECT * FROM remote('host', 'db', 'table')", true},
		{"mysql function", "SELECT * FROM mysql('host', 'db', 'table', 'user', 'pass')", true},
		{"postgresql function", "SELECT * FROM postgresql('host', 'db', 'table', 'user', 'pass')", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mcp.ValidateSQL(tt.sql, mcp.ClickhouseBlockedPatterns)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSQL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnforceLimit(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		maxRows  int
		keyword  string
		expected string
	}{
		{
			"add limit",
			"SELECT * FROM users",
			100,
			"LIMIT",
			"SELECT * FROM users LIMIT 100",
		},
		{
			"already has limit",
			"SELECT * FROM users LIMIT 10",
			100,
			"LIMIT",
			"SELECT * FROM users LIMIT 10",
		},
		{
			"remove semicolon and add limit",
			"SELECT * FROM users;",
			50,
			"LIMIT",
			"SELECT * FROM users LIMIT 50",
		},
		{
			"complex query",
			"SELECT * FROM users WHERE active ORDER BY name",
			25,
			"LIMIT",
			"SELECT * FROM users WHERE active ORDER BY name LIMIT 25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mcp.EnforceLimit(tt.sql, tt.maxRows, tt.keyword)
			if result != tt.expected {
				t.Errorf("EnforceLimit() = %q, want %q", result, tt.expected)
			}
		})
	}
}
