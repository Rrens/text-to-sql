package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rensmac/text-to-sql/internal/mcp"
)

// Adapter implements mcp.Adapter for PostgreSQL
type Adapter struct {
	pool *pgxpool.Pool
}

// NewAdapter creates a new PostgreSQL adapter
func NewAdapter() mcp.Adapter {
	return &Adapter{}
}

// DatabaseType returns the database type identifier
func (a *Adapter) DatabaseType() string {
	return "postgres"
}

// SQLDialect returns SQL dialect hints for LLM prompting
func (a *Adapter) SQLDialect() string {
	return `PostgreSQL SQL dialect:
- Use double quotes for identifiers with special characters: "column name"
- String concatenation: column1 || column2
- Case-insensitive matching: ILIKE instead of LIKE
- Date/time functions: NOW(), CURRENT_DATE, CURRENT_TIMESTAMP
- Date truncation: DATE_TRUNC('month', date_column)
- Date extraction: EXTRACT(YEAR FROM date_column)
- Pagination: LIMIT n OFFSET m
- Boolean values: TRUE, FALSE
- NULL handling: COALESCE(column, default_value), NULLIF(a, b)
- Array functions: ANY(), ALL(), array_agg()
- JSON functions: jsonb_extract_path(), ->, ->>
- String functions: CONCAT(), SUBSTRING(), TRIM(), UPPER(), LOWER()
- Aggregate functions: COUNT(), SUM(), AVG(), MIN(), MAX(), STRING_AGG()
- Window functions: ROW_NUMBER(), RANK(), DENSE_RANK(), LAG(), LEAD()
- Common table expressions (CTEs): WITH cte AS (SELECT ...)`
}

// Connect establishes connection to PostgreSQL
func (a *Adapter) Connect(ctx context.Context, config mcp.ConnectionConfig) error {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	poolConfig.MaxConns = 5
	poolConfig.MinConns = 1

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping: %w", err)
	}

	a.pool = pool
	return nil
}

// Close closes the connection
func (a *Adapter) Close() error {
	if a.pool != nil {
		a.pool.Close()
		a.pool = nil
	}
	return nil
}

// HealthCheck verifies connection is alive
func (a *Adapter) HealthCheck(ctx context.Context) error {
	if a.pool == nil {
		return fmt.Errorf("not connected")
	}
	return a.pool.Ping(ctx)
}

// ListTables returns list of table names
func (a *Adapter) ListTables(ctx context.Context) ([]string, error) {
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := a.pool.Query(ctx, query)
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
	query := `
		SELECT 
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' as nullable,
			COALESCE(
				(SELECT true FROM information_schema.key_column_usage kcu
				 JOIN information_schema.table_constraints tc 
				   ON kcu.constraint_name = tc.constraint_name
				 WHERE tc.constraint_type = 'PRIMARY KEY'
				   AND kcu.table_name = c.table_name
				   AND kcu.column_name = c.column_name
				 LIMIT 1), false
			) as primary_key,
			COALESCE(col_description(
				(SELECT oid FROM pg_class WHERE relname = c.table_name LIMIT 1),
				c.ordinal_position
			), '') as description
		FROM information_schema.columns c
		WHERE c.table_schema = 'public' AND c.table_name = $1
		ORDER BY c.ordinal_position
	`

	rows, err := a.pool.Query(ctx, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to describe table: %w", err)
	}
	defer rows.Close()

	var columns []mcp.ColumnInfo
	for rows.Next() {
		var col mcp.ColumnInfo
		if err := rows.Scan(&col.Name, &col.DataType, &col.Nullable, &col.PrimaryKey, &col.Description); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}
		columns = append(columns, col)
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	// Get row count estimate
	var rowCount int64
	err = a.pool.QueryRow(ctx, `
		SELECT reltuples::bigint 
		FROM pg_class 
		WHERE relname = $1
	`, tableName).Scan(&rowCount)

	var rowCountPtr *int64
	if err == nil && rowCount >= 0 {
		rowCountPtr = &rowCount
	}

	return &mcp.TableInfo{
		Name:       tableName,
		SchemaName: "public",
		Columns:    columns,
		RowCount:   rowCountPtr,
	}, nil
}

// GetSchemaDDL returns full schema as DDL for LLM context
func (a *Adapter) GetSchemaDDL(ctx context.Context) (string, error) {
	query := `
		SELECT 
			c.table_name,
			c.column_name,
			c.data_type,
			c.is_nullable,
			c.column_default,
			COALESCE(
				(SELECT 'PRIMARY KEY' FROM information_schema.key_column_usage kcu
				 JOIN information_schema.table_constraints tc 
				   ON kcu.constraint_name = tc.constraint_name
				 WHERE tc.constraint_type = 'PRIMARY KEY'
				   AND kcu.table_name = c.table_name
				   AND kcu.column_name = c.column_name
				 LIMIT 1), ''
			) as constraint_type
		FROM information_schema.columns c
		WHERE c.table_schema = 'public'
		ORDER BY c.table_name, c.ordinal_position
	`

	rows, err := a.pool.Query(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to get schema: %w", err)
	}
	defer rows.Close()

	var ddl strings.Builder
	currentTable := ""

	for rows.Next() {
		var tableName, columnName, dataType, isNullable, constraintType string
		var columnDefault *string

		if err := rows.Scan(&tableName, &columnName, &dataType, &isNullable, &columnDefault, &constraintType); err != nil {
			return "", fmt.Errorf("failed to scan: %w", err)
		}

		if tableName != currentTable {
			if currentTable != "" {
				ddl.WriteString("\n);\n\n")
			}
			ddl.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", tableName))
			currentTable = tableName
		} else {
			ddl.WriteString(",\n")
		}

		nullable := ""
		if isNullable == "NO" {
			nullable = " NOT NULL"
		}

		pk := ""
		if constraintType == "PRIMARY KEY" {
			pk = " PRIMARY KEY"
		}

		ddl.WriteString(fmt.Sprintf("  %s %s%s%s", columnName, dataType, nullable, pk))
	}

	if currentTable != "" {
		ddl.WriteString("\n);")
	}

	return ddl.String(), nil
}

// ValidateQuery validates SQL is safe to execute
func (a *Adapter) ValidateQuery(sql string) error {
	return mcp.ValidateSQL(sql, mcp.PostgresBlockedPatterns)
}

// ExecuteQuery executes read-only SQL query
func (a *Adapter) ExecuteQuery(ctx context.Context, sql string, opts mcp.QueryOptions) (*mcp.QueryResult, error) {
	if err := a.ValidateQuery(sql); err != nil {
		return nil, err
	}

	// Enforce LIMIT
	sql = mcp.EnforceLimit(sql, opts.MaxRows, "LIMIT")

	// Create context with timeout
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	rows, err := a.pool.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	fieldDescs := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescs))
	for i, fd := range fieldDescs {
		columns[i] = string(fd.Name)
	}

	// Collect rows
	var resultRows [][]any
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to get row values: %w", err)
		}
		resultRows = append(resultRows, values)

		// Stop if we've exceeded max rows
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
