package security_test

import (
	"testing"

	"github.com/Rrens/text-to-sql/internal/security"
)

func TestSQLValidator_Validate(t *testing.T) {
	validator := security.NewSQLValidator()

	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		// Valid queries
		{"simple select", "SELECT * FROM users", false},
		{"select with where", "SELECT id, name FROM users WHERE id = 1", false},
		{"select with join", "SELECT u.id, o.total FROM users u JOIN orders o ON u.id = o.user_id", false},
		{"select with limit", "SELECT * FROM users LIMIT 10", false},
		{"select with order", "SELECT * FROM users ORDER BY created_at DESC", false},
		{"select with group", "SELECT status, COUNT(*) FROM orders GROUP BY status", false},
		{"cte query", "WITH active AS (SELECT * FROM users WHERE active = true) SELECT * FROM active", false},
		{"subquery", "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)", false},

		// Invalid queries - empty
		{"empty", "", true},
		{"whitespace only", "   ", true},

		// Invalid queries - not SELECT
		{"insert", "INSERT INTO users (name) VALUES ('test')", true},
		{"update", "UPDATE users SET name = 'test' WHERE id = 1", true},
		{"delete", "DELETE FROM users WHERE id = 1", true},
		{"drop", "DROP TABLE users", true},
		{"truncate", "TRUNCATE TABLE users", true},
		{"alter", "ALTER TABLE users ADD COLUMN email VARCHAR(255)", true},
		{"create", "CREATE TABLE test (id INT)", true},
		{"grant", "GRANT SELECT ON users TO readonly", true},
		{"revoke", "REVOKE SELECT ON users FROM readonly", true},

		// Invalid queries - blocked patterns
		{"exec", "EXEC sp_executesql 'SELECT 1'", true},
		{"execute", "EXECUTE sp_executesql 'SELECT 1'", true},
		{"into outfile", "SELECT * FROM users INTO OUTFILE '/tmp/data.csv'", true},
		{"into dumpfile", "SELECT * FROM users INTO DUMPFILE '/tmp/data.csv'", true},
		{"load_file", "SELECT LOAD_FILE('/etc/passwd')", true},

		// Multiple statements
		{"multiple statements", "SELECT 1; SELECT 2;", true},
		{"statement with drop", "SELECT 1; DROP TABLE users", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSQLValidator_EnforceLimit(t *testing.T) {
	validator := security.NewSQLValidator()

	tests := []struct {
		name     string
		sql      string
		maxRows  int
		expected string
	}{
		{
			"add limit",
			"SELECT * FROM users",
			100,
			"SELECT * FROM users LIMIT 100",
		},
		{
			"already has limit",
			"SELECT * FROM users LIMIT 10",
			100,
			"SELECT * FROM users LIMIT 10",
		},
		{
			"remove trailing semicolon",
			"SELECT * FROM users;",
			100,
			"SELECT * FROM users LIMIT 100",
		},
		{
			"complex query",
			"SELECT u.id, COUNT(o.id) FROM users u JOIN orders o ON u.id = o.user_id GROUP BY u.id ORDER BY COUNT(o.id) DESC",
			50,
			"SELECT u.id, COUNT(o.id) FROM users u JOIN orders o ON u.id = o.user_id GROUP BY u.id ORDER BY COUNT(o.id) DESC LIMIT 50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.EnforceLimit(tt.sql, tt.maxRows)
			if result != tt.expected {
				t.Errorf("EnforceLimit() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSQLValidator_ValidateAndPrepare(t *testing.T) {
	validator := security.NewSQLValidator()

	tests := []struct {
		name    string
		sql     string
		maxRows int
		wantSQL string
		wantErr bool
	}{
		{
			"valid query gets limit",
			"SELECT * FROM users",
			100,
			"SELECT * FROM users LIMIT 100",
			false,
		},
		{
			"invalid query returns error",
			"DELETE FROM users",
			100,
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateAndPrepare(tt.sql, tt.maxRows)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndPrepare() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result != tt.wantSQL {
				t.Errorf("ValidateAndPrepare() = %q, want %q", result, tt.wantSQL)
			}
		})
	}
}
