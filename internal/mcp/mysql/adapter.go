package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Rrens/text-to-sql/internal/mcp"
	_ "github.com/go-sql-driver/mysql"
)

// Adapter implements mcp.Adapter for MySQL
type Adapter struct {
	db       *sql.DB
	database string
}

// NewAdapter creates a new MySQL adapter
func NewAdapter() mcp.Adapter {
	return &Adapter{}
}

// DatabaseType returns the database type identifier
func (a *Adapter) DatabaseType() string {
	return "mysql"
}

// SQLDialect returns SQL dialect hints for LLM prompting
func (a *Adapter) SQLDialect() string {
	return `MySQL SQL dialect:
- Use backticks for identifiers: ` + "`column_name`" + `
- String concatenation: CONCAT(a, b)
- Case-insensitive matching: LIKE (MySQL is case-insensitive by default)
- Date functions: NOW(), CURDATE(), CURRENT_TIMESTAMP
- Date formatting: DATE_FORMAT(date, '%Y-%m-%d')
- Date extraction: YEAR(date), MONTH(date), DAY(date)
- Pagination: LIMIT n OFFSET m or LIMIT offset, count
- Boolean values: TRUE/FALSE or 1/0
- NULL handling: IFNULL(column, default), NULLIF(a, b), COALESCE()
- String functions: CONCAT(), SUBSTRING(), TRIM(), UPPER(), LOWER()
- Aggregate functions: COUNT(), SUM(), AVG(), MIN(), MAX(), GROUP_CONCAT()
- Use single quotes for strings
- Avoid using reserved words as identifiers
- Use INDEX hints if needed: FORCE INDEX, USE INDEX
- EXPLAIN for query analysis`
}

// Connect establishes connection to MySQL
func (a *Adapter) Connect(ctx context.Context, config mcp.ConnectionConfig) error {
	// Build DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	// Add TLS if required
	if config.SSLMode == "require" || config.SSLMode == "verify-full" {
		dsn += "&tls=true"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping: %w", err)
	}

	a.db = db
	a.database = config.Database
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
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = ? 
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`, a.database)
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
	rows, err := a.db.QueryContext(ctx, `
		SELECT 
			column_name,
			column_type,
			is_nullable = 'YES',
			column_key = 'PRI',
			COALESCE(column_comment, '')
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position
	`, a.database, tableName)
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
	err = a.db.QueryRowContext(ctx, `
		SELECT table_rows 
		FROM information_schema.tables 
		WHERE table_schema = ? AND table_name = ?
	`, a.database, tableName).Scan(&rowCount)

	var rowCountPtr *int64
	if err == nil && rowCount >= 0 {
		rowCountPtr = &rowCount
	}

	return &mcp.TableInfo{
		Name:       tableName,
		SchemaName: a.database,
		Columns:    columns,
		RowCount:   rowCountPtr,
	}, nil
}

// GetSchemaDDL returns full schema as DDL for LLM context
func (a *Adapter) GetSchemaDDL(ctx context.Context) (string, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT 
			table_name,
			column_name,
			column_type,
			is_nullable,
			column_key
		FROM information_schema.columns
		WHERE table_schema = ?
		ORDER BY table_name, ordinal_position
	`, a.database)
	if err != nil {
		return "", fmt.Errorf("failed to get schema: %w", err)
	}
	defer rows.Close()

	var ddl strings.Builder
	currentTable := ""

	for rows.Next() {
		var tableName, columnName, dataType, isNullable, columnKey string

		if err := rows.Scan(&tableName, &columnName, &dataType, &isNullable, &columnKey); err != nil {
			return "", fmt.Errorf("failed to scan: %w", err)
		}

		if tableName != currentTable {
			if currentTable != "" {
				ddl.WriteString("\n);\n\n")
			}
			ddl.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n", tableName))
			currentTable = tableName
		} else {
			ddl.WriteString(",\n")
		}

		nullable := ""
		if isNullable == "NO" {
			nullable = " NOT NULL"
		}

		pk := ""
		if columnKey == "PRI" {
			pk = " PRIMARY KEY"
		}

		ddl.WriteString(fmt.Sprintf("  `%s` %s%s%s", columnName, dataType, nullable, pk))
	}

	if currentTable != "" {
		ddl.WriteString("\n);")
	}

	return ddl.String(), nil
}

// ValidateQuery validates SQL is safe to execute
func (a *Adapter) ValidateQuery(sql string) error {
	return mcp.ValidateSQL(sql, mcp.MysqlBlockedPatterns)
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

	rows, err := a.db.QueryContext(ctx, sql)
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
