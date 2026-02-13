package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Rrens/text-to-sql/internal/mcp"
	_ "modernc.org/sqlite"
)

// Adapter implements mcp.Adapter for SQLite
type Adapter struct {
	db       *sql.DB
	database string
}

// NewAdapter creates a new SQLite adapter
func NewAdapter() mcp.Adapter {
	return &Adapter{}
}

// DatabaseType returns the database type identifier
func (a *Adapter) DatabaseType() string {
	return "sqlite"
}

// SQLDialect returns SQL dialect hints for LLM prompting
func (a *Adapter) SQLDialect() string {
	return `SQLite SQL dialect:
- Use double quotes for identifiers: "column_name"
- String concatenation: || operator (e.g., col1 || ' ' || col2)
- Case-insensitive matching: LIKE (case-insensitive by default for ASCII)
- Date functions: date(), time(), datetime(), julianday(), strftime()
- Current time: datetime('now'), date('now')
- Date formatting: strftime('%Y-%m-%d', date_column)
- Pagination: LIMIT n OFFSET m
- Boolean values: 0 and 1 (no native boolean type)
- NULL handling: IFNULL(column, default), NULLIF(a, b), COALESCE()
- String functions: LENGTH(), SUBSTR(), TRIM(), UPPER(), LOWER(), REPLACE()
- Aggregate functions: COUNT(), SUM(), AVG(), MIN(), MAX(), GROUP_CONCAT()
- Use single quotes for strings
- No native ENUM type - use CHECK constraints
- AUTOINCREMENT with INTEGER PRIMARY KEY
- typeof() function to check value types
- No RIGHT JOIN or FULL OUTER JOIN support (use LEFT JOIN alternatives)
- Use EXPLAIN QUERY PLAN for query analysis`
}

// Connect establishes connection to SQLite database file
func (a *Adapter) Connect(ctx context.Context, config mcp.ConnectionConfig) error {
	// For SQLite, Database field holds the file path
	dbPath := config.Database
	if dbPath == "" {
		return fmt.Errorf("database file path is required")
	}

	// Open with read-only mode and other pragmas via DSN
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite only supports one writer
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	a.db = db
	a.database = dbPath
	return nil
}

// Close closes the connection
func (a *Adapter) Close() error {
	if a.db != nil {
		err := a.db.Close()
		a.db = nil
		return err
	}
	return nil
}

// HealthCheck verifies connection is alive
func (a *Adapter) HealthCheck(ctx context.Context) error {
	if a.db == nil {
		return fmt.Errorf("not connected")
	}
	return a.db.PingContext(ctx)
}

// ListTables returns list of table names
func (a *Adapter) ListTables(ctx context.Context) ([]string, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT name 
		FROM sqlite_master 
		WHERE type = 'table' 
		  AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

// DescribeTable returns detailed table schema
func (a *Adapter) DescribeTable(ctx context.Context, tableName string) (*mcp.TableInfo, error) {
	rows, err := a.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info('%s')", tableName))
	if err != nil {
		return nil, fmt.Errorf("failed to describe table: %w", err)
	}
	defer rows.Close()

	var columns []mcp.ColumnInfo
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var dfltValue sql.NullString

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		columns = append(columns, mcp.ColumnInfo{
			Name:       name,
			DataType:   dataType,
			Nullable:   notNull == 0,
			PrimaryKey: pk > 0,
		})
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	// Get row count
	var rowCount int64
	err = a.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", tableName)).Scan(&rowCount)

	var rowCountPtr *int64
	if err == nil && rowCount >= 0 {
		rowCountPtr = &rowCount
	}

	return &mcp.TableInfo{
		Name:     tableName,
		Columns:  columns,
		RowCount: rowCountPtr,
	}, nil
}

// GetSchemaDDL returns full schema as DDL for LLM context
func (a *Adapter) GetSchemaDDL(ctx context.Context) (string, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT name, sql 
		FROM sqlite_master 
		WHERE type = 'table' 
		  AND name NOT LIKE 'sqlite_%'
		  AND sql IS NOT NULL
		ORDER BY name
	`)
	if err != nil {
		return "", fmt.Errorf("failed to get schema: %w", err)
	}
	defer rows.Close()

	var ddl strings.Builder
	for rows.Next() {
		var name, createSQL string
		if err := rows.Scan(&name, &createSQL); err != nil {
			return "", fmt.Errorf("failed to scan: %w", err)
		}

		ddl.WriteString(createSQL)
		ddl.WriteString(";\n\n")
	}

	return ddl.String(), nil
}

// ValidateQuery validates SQL is safe to execute
func (a *Adapter) ValidateQuery(sql string) error {
	return mcp.ValidateSQL(sql, mcp.SqliteBlockedPatterns)
}

// ExecuteQuery executes read-only SQL query
func (a *Adapter) ExecuteQuery(ctx context.Context, sqlStr string, opts mcp.QueryOptions) (*mcp.QueryResult, error) {
	if err := a.ValidateQuery(sqlStr); err != nil {
		return nil, err
	}

	// Enforce LIMIT
	sqlStr = mcp.EnforceLimit(sqlStr, opts.MaxRows, "LIMIT")

	// Create context with timeout
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	rows, err := a.db.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Collect rows
	var resultRows [][]any
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert []byte to string for better JSON serialization
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				values[i] = string(b)
			}
		}

		resultRows = append(resultRows, values)

		if len(resultRows) > opts.MaxRows {
			break
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	truncated := len(resultRows) > opts.MaxRows
	if truncated {
		resultRows = resultRows[:opts.MaxRows]
	}

	return &mcp.QueryResult{
		Columns:   columns,
		Rows:      resultRows,
		RowCount:  len(resultRows),
		Truncated: truncated,
	}, nil
}
