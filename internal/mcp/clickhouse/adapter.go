package clickhouse

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Rrens/text-to-sql/internal/mcp"
)

// Adapter implements mcp.Adapter for ClickHouse
type Adapter struct {
	conn driver.Conn
}

// NewAdapter creates a new ClickHouse adapter
func NewAdapter() mcp.Adapter {
	return &Adapter{}
}

// DatabaseType returns the database type identifier
func (a *Adapter) DatabaseType() string {
	return "clickhouse"
}

// SQLDialect returns SQL dialect hints for LLM prompting
func (a *Adapter) SQLDialect() string {
	return `ClickHouse SQL dialect:
- Use backticks for identifiers: ` + "`column_name`" + `
- String concatenation: concat(a, b) or a || b
- Date functions: today(), now(), toDate(), toDateTime()
- Date truncation: toStartOfMonth(date), toStartOfDay(datetime)
- Date extraction: toYear(date), toMonth(date), toDayOfMonth(date)
- Pagination: LIMIT n OFFSET m (but avoid large offsets)
- Boolean values: 1/0 or true/false
- NULL handling: ifNull(column, default), nullIf(a, b)
- Array functions: arrayJoin(), groupArray(), arrayElement()
- String functions: concat(), substring(), trim(), upper(), lower()
- Aggregate functions: count(), sum(), avg(), min(), max(), groupArray()
- Approximate functions: uniq(), uniqExact(), quantile()
- Use FORMAT JSONEachRow for debugging
- Prefer using MergeTree tables
- Use FINAL for ReplacingMergeTree/CollapsingMergeTree when needed
- Avoid SELECT * on large tables, specify columns`
}

// Connect establishes connection to ClickHouse
func (a *Adapter) Connect(ctx context.Context, config mcp.ConnectionConfig) error {
	options := &clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
		Debug: false,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	}

	// Handle SSL
	if config.SSLMode == "require" || config.SSLMode == "verify-full" {
		options.TLS = &tls.Config{
			InsecureSkipVerify: config.SSLMode != "verify-full",
		}
	}

	conn, err := clickhouse.Open(options)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping: %w", err)
	}

	a.conn = conn
	return nil
}

// Close closes the connection
func (a *Adapter) Close() error {
	if a.conn != nil {
		err := a.conn.Close()
		a.conn = nil
		return err
	}
	return nil
}

// HealthCheck verifies connection is alive
func (a *Adapter) HealthCheck(ctx context.Context) error {
	if a.conn == nil {
		return fmt.Errorf("not connected")
	}
	return a.conn.Ping(ctx)
}

// ListTables returns list of table names
func (a *Adapter) ListTables(ctx context.Context) ([]string, error) {
	rows, err := a.conn.Query(ctx, `
		SELECT name 
		FROM system.tables 
		WHERE database = currentDatabase()
		  AND engine NOT IN ('View', 'MaterializedView')
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
	rows, err := a.conn.Query(ctx, `
		SELECT 
			name,
			type,
			default_kind != '' as has_default,
			is_in_primary_key,
			comment
		FROM system.columns
		WHERE database = currentDatabase() AND table = $1
		ORDER BY position
	`, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to describe table: %w", err)
	}
	defer rows.Close()

	var columns []mcp.ColumnInfo
	for rows.Next() {
		var name, dataType, comment string
		var hasDefault, isPrimaryKey bool

		if err := rows.Scan(&name, &dataType, &hasDefault, &isPrimaryKey, &comment); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		// Check if nullable
		nullable := strings.HasPrefix(dataType, "Nullable(")

		columns = append(columns, mcp.ColumnInfo{
			Name:        name,
			DataType:    dataType,
			Nullable:    nullable,
			PrimaryKey:  isPrimaryKey,
			Description: comment,
		})
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	// Get row count estimate
	var rowCount int64
	err = a.conn.QueryRow(ctx, `
		SELECT total_rows 
		FROM system.tables 
		WHERE database = currentDatabase() AND name = $1
	`, tableName).Scan(&rowCount)

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
	rows, err := a.conn.Query(ctx, `
		SELECT 
			table,
			name,
			type,
			is_in_primary_key,
			comment
		FROM system.columns
		WHERE database = currentDatabase()
		ORDER BY table, position
	`)
	if err != nil {
		return "", fmt.Errorf("failed to get schema: %w", err)
	}
	defer rows.Close()

	var ddl strings.Builder
	currentTable := ""

	for rows.Next() {
		var tableName, columnName, dataType, comment string
		var isPrimaryKey bool

		if err := rows.Scan(&tableName, &columnName, &dataType, &isPrimaryKey, &comment); err != nil {
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

		pk := ""
		if isPrimaryKey {
			pk = " -- PRIMARY KEY"
		}

		ddl.WriteString(fmt.Sprintf("  %s %s%s", columnName, dataType, pk))
	}

	if currentTable != "" {
		ddl.WriteString("\n);")
	}

	return ddl.String(), nil
}

// ValidateQuery validates SQL is safe to execute
func (a *Adapter) ValidateQuery(sql string) error {
	return mcp.ValidateSQL(sql, mcp.ClickhouseBlockedPatterns)
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

	rows, err := a.conn.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	columnTypes := rows.ColumnTypes()
	columns := make([]string, len(columnTypes))
	for i, ct := range columnTypes {
		columns[i] = ct.Name()
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
