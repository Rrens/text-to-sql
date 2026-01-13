package clickhouse

import (
	"context"
	"fmt"
	"strings"

	"github.com/Rrens/text-to-sql/internal/mcp"
)

// Adapter implements mcp.Adapter for ClickHouse using HTTP protocol
type Adapter struct {
	client   *HTTPClient
	database string
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

// Connect establishes connection to ClickHouse using HTTP protocol
func (a *Adapter) Connect(ctx context.Context, config mcp.ConnectionConfig) error {
	a.client = NewHTTPClient(
		config.Host,
		config.Port,
		config.Database,
		config.Username,
		config.Password,
	)
	a.database = config.Database

	// Test connection
	if err := a.client.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping: %w", err)
	}

	return nil
}

// Close closes the connection
func (a *Adapter) Close() error {
	if a.client != nil {
		err := a.client.Close()
		a.client = nil
		return err
	}
	return nil
}

// HealthCheck verifies connection is alive
func (a *Adapter) HealthCheck(ctx context.Context) error {
	if a.client == nil {
		return fmt.Errorf("not connected")
	}
	return a.client.Ping(ctx)
}

// ListTables returns list of table names
func (a *Adapter) ListTables(ctx context.Context) ([]string, error) {
	results, err := a.client.Query(ctx, `
		SELECT name 
		FROM system.tables 
		WHERE database = currentDatabase()
		  AND engine NOT IN ('View', 'MaterializedView')
		  AND name NOT LIKE '.%'
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	var tables []string
	for _, row := range results {
		if name, ok := row["name"].(string); ok {
			tables = append(tables, name)
		}
	}

	return tables, nil
}

// DescribeTable returns detailed table schema
func (a *Adapter) DescribeTable(ctx context.Context, tableName string) (*mcp.TableInfo, error) {
	query := fmt.Sprintf(`
		SELECT 
			name,
			type,
			default_kind != '' as has_default,
			is_in_primary_key,
			comment
		FROM system.columns
		WHERE database = currentDatabase() AND table = '%s'
		ORDER BY position
	`, escapeSQLString(tableName))

	results, err := a.client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to describe table: %w", err)
	}

	var columns []mcp.ColumnInfo
	for _, row := range results {
		name, _ := row["name"].(string)
		dataType, _ := row["type"].(string)
		isPrimaryKey := toBool(row["is_in_primary_key"])
		comment, _ := row["comment"].(string)

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
	countQuery := fmt.Sprintf(`
		SELECT total_rows 
		FROM system.tables 
		WHERE database = currentDatabase() AND name = '%s'
	`, escapeSQLString(tableName))

	countResults, err := a.client.Query(ctx, countQuery)
	var rowCountPtr *int64
	if err == nil && len(countResults) > 0 {
		if count, ok := countResults[0]["total_rows"]; ok {
			var rowCount int64
			switch v := count.(type) {
			case float64:
				rowCount = int64(v)
			case int64:
				rowCount = v
			}
			if rowCount >= 0 {
				rowCountPtr = &rowCount
			}
		}
	}

	return &mcp.TableInfo{
		Name:     tableName,
		Columns:  columns,
		RowCount: rowCountPtr,
	}, nil
}

// GetSchemaDDL returns full schema as DDL for LLM context
func (a *Adapter) GetSchemaDDL(ctx context.Context) (string, error) {
	// 1. Get List of all tables first
	tables, err := a.ListTables(ctx)
	if err != nil {
		return "", err
	}

	// 2. Decide strategy based on table count
	// If too many tables, only include full schema for a subset to save tokens
	const MaxFullTables = 10
	var tablesToDescribe []string
	var tablesToList []string

	if len(tables) > MaxFullTables {
		tablesToDescribe = tables[:MaxFullTables]
		tablesToList = tables[MaxFullTables:]
	} else {
		tablesToDescribe = tables
	}

	var ddl strings.Builder

	// 3. Fetch columns for the selected tables
	if len(tablesToDescribe) > 0 {
		// Build IN clause
		var quotedTables []string
		for _, t := range tablesToDescribe {
			quotedTables = append(quotedTables, fmt.Sprintf("'%s'", escapeSQLString(t)))
		}
		inClause := strings.Join(quotedTables, ",")

		query := fmt.Sprintf(`
			SELECT 
				table,
				name,
				type,
				is_in_primary_key,
				comment
			FROM system.columns
			WHERE database = currentDatabase()
			  AND table IN (%s)
			ORDER BY table, position
		`, inClause)

		results, err := a.client.Query(ctx, query)
		if err != nil {
			return "", fmt.Errorf("failed to get schema details: %w", err)
		}

		currentTable := ""
		for _, row := range results {
			tableName, _ := row["table"].(string)
			columnName, _ := row["name"].(string)
			dataType, _ := row["type"].(string)
			isPrimaryKey := toBool(row["is_in_primary_key"])

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
			ddl.WriteString("\n);\n\n")
		}
	}

	// 4. Append list of other tables (names only)
	if len(tablesToList) > 0 {
		ddl.WriteString(fmt.Sprintf("-- Other tables available (schema truncated for brevity, %d total):\n", len(tables)))
		for _, t := range tablesToList {
			ddl.WriteString(fmt.Sprintf("-- Table: %s\n", t))
		}
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

	results, err := a.client.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Convert results to row format
	var columns []string
	var resultRows [][]any

	if len(results) > 0 {
		// Get column names from first row
		for key := range results[0] {
			columns = append(columns, key)
		}

		// Convert rows
		for i, row := range results {
			if i >= opts.MaxRows {
				break
			}
			values := make([]any, len(columns))
			for j, col := range columns {
				values[j] = row[col]
			}
			resultRows = append(resultRows, values)
		}
	}

	truncated := len(results) > opts.MaxRows

	return &mcp.QueryResult{
		Columns:   columns,
		Rows:      resultRows,
		RowCount:  len(resultRows),
		Truncated: truncated,
	}, nil
}

// Helper functions

func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func toBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val != 0
	case int:
		return val != 0
	case int64:
		return val != 0
	case string:
		return val == "1" || val == "true"
	default:
		return false
	}
}
