package sqlserver

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Rrens/text-to-sql/internal/mcp"
	_ "github.com/microsoft/go-mssqldb"
)

// Adapter implements mcp.Adapter for SQL Server
type Adapter struct {
	db       *sql.DB
	database string
}

// NewAdapter creates a new SQL Server adapter
func NewAdapter() mcp.Adapter {
	return &Adapter{}
}

// DatabaseType returns the database type identifier
func (a *Adapter) DatabaseType() string {
	return "sqlserver"
}

// SQLDialect returns SQL dialect hints for LLM prompting
func (a *Adapter) SQLDialect() string {
	return `T-SQL (SQL Server) dialect:
- Use square brackets for identifiers: [column_name]
- String concatenation: CONCAT(a, b) or a + b
- Case-insensitive matching: LIKE (SQL Server is case-insensitive by default with most collations)
- Date functions: GETDATE(), SYSDATETIME(), CURRENT_TIMESTAMP
- Date formatting: FORMAT(date, 'yyyy-MM-dd') or CONVERT(VARCHAR, date, 23)
- Date extraction: YEAR(date), MONTH(date), DAY(date), DATEPART(part, date)
- Pagination: OFFSET m ROWS FETCH NEXT n ROWS ONLY (SQL Server 2012+) or TOP n
- Boolean values: 1/0 (no native BOOLEAN type, use BIT)
- NULL handling: ISNULL(column, default), NULLIF(a, b), COALESCE()
- String functions: CONCAT(), SUBSTRING(), TRIM(), UPPER(), LOWER(), LEN(), CHARINDEX()
- Aggregate functions: COUNT(), SUM(), AVG(), MIN(), MAX(), STRING_AGG()
- Use single quotes for strings
- Use TOP N instead of LIMIT N for simple row limiting
- Use OFFSET/FETCH for pagination with ORDER BY
- Common Table Expressions (WITH) are supported
- Use SET NOCOUNT ON to suppress row count messages
- Use EXPLAIN → SET SHOWPLAN_TEXT ON for query analysis`
}

// Connect establishes connection to SQL Server
func (a *Adapter) Connect(ctx context.Context, config mcp.ConnectionConfig) error {
	// Build DSN: sqlserver://user:pass@host:port?database=dbname
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	// Add encryption setting based on SSL mode
	switch config.SSLMode {
	case "disable":
		dsn += "&encrypt=disable"
	case "require":
		dsn += "&encrypt=true&TrustServerCertificate=true"
	case "verify-full":
		dsn += "&encrypt=true&TrustServerCertificate=false"
	default:
		dsn += "&encrypt=disable"
	}

	db, err := sql.Open("sqlserver", dsn)
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
		SELECT TABLE_NAME 
		FROM INFORMATION_SCHEMA.TABLES 
		WHERE TABLE_CATALOG = @p1 
		  AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME
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
			c.COLUMN_NAME,
			c.DATA_TYPE,
			CASE WHEN c.IS_NULLABLE = 'YES' THEN 1 ELSE 0 END AS is_nullable,
			CASE WHEN kcu.COLUMN_NAME IS NOT NULL THEN 1 ELSE 0 END AS is_primary_key,
			ISNULL(sep.value, '') AS description
		FROM INFORMATION_SCHEMA.COLUMNS c
		LEFT JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
			ON tc.TABLE_CATALOG = c.TABLE_CATALOG
			AND tc.TABLE_SCHEMA = c.TABLE_SCHEMA
			AND tc.TABLE_NAME = c.TABLE_NAME
			AND tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
		LEFT JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
			ON kcu.CONSTRAINT_NAME = tc.CONSTRAINT_NAME
			AND kcu.TABLE_CATALOG = tc.TABLE_CATALOG
			AND kcu.TABLE_SCHEMA = tc.TABLE_SCHEMA
			AND kcu.TABLE_NAME = tc.TABLE_NAME
			AND kcu.COLUMN_NAME = c.COLUMN_NAME
		LEFT JOIN sys.extended_properties sep
			ON sep.major_id = OBJECT_ID(c.TABLE_SCHEMA + '.' + c.TABLE_NAME)
			AND sep.minor_id = c.ORDINAL_POSITION
			AND sep.name = 'MS_Description'
		WHERE c.TABLE_CATALOG = @p1 AND c.TABLE_NAME = @p2
		ORDER BY c.ORDINAL_POSITION
	`, a.database, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to describe table: %w", err)
	}
	defer rows.Close()

	var columns []mcp.ColumnInfo
	for rows.Next() {
		var col mcp.ColumnInfo
		var desc string
		if err := rows.Scan(&col.Name, &col.DataType, &col.Nullable, &col.PrimaryKey, &desc); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}
		col.Description = desc
		columns = append(columns, col)
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	// Get row count estimate
	var rowCount int64
	err = a.db.QueryRowContext(ctx, `
		SELECT SUM(p.rows)
		FROM sys.partitions p
		JOIN sys.tables t ON p.object_id = t.object_id
		WHERE t.name = @p1
		  AND p.index_id IN (0, 1)
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
	rows, err := a.db.QueryContext(ctx, `
		SELECT 
			c.TABLE_NAME,
			c.COLUMN_NAME,
			c.DATA_TYPE,
			c.IS_NULLABLE,
			CASE WHEN kcu.COLUMN_NAME IS NOT NULL THEN 'PRI' ELSE '' END AS column_key
		FROM INFORMATION_SCHEMA.COLUMNS c
		LEFT JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
			ON tc.TABLE_CATALOG = c.TABLE_CATALOG
			AND tc.TABLE_SCHEMA = c.TABLE_SCHEMA
			AND tc.TABLE_NAME = c.TABLE_NAME
			AND tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
		LEFT JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
			ON kcu.CONSTRAINT_NAME = tc.CONSTRAINT_NAME
			AND kcu.TABLE_CATALOG = tc.TABLE_CATALOG
			AND kcu.TABLE_SCHEMA = tc.TABLE_SCHEMA
			AND kcu.TABLE_NAME = tc.TABLE_NAME
			AND kcu.COLUMN_NAME = c.COLUMN_NAME
		WHERE c.TABLE_CATALOG = @p1
		ORDER BY c.TABLE_NAME, c.ORDINAL_POSITION
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
			ddl.WriteString(fmt.Sprintf("CREATE TABLE [%s] (\n", tableName))
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

		ddl.WriteString(fmt.Sprintf("  [%s] %s%s%s", columnName, dataType, nullable, pk))
	}

	if currentTable != "" {
		ddl.WriteString("\n);")
	}

	return ddl.String(), nil
}

// ValidateQuery validates SQL is safe to execute
func (a *Adapter) ValidateQuery(sql string) error {
	return mcp.ValidateSQL(sql, mcp.SqlserverBlockedPatterns)
}

// ExecuteQuery executes read-only SQL query
func (a *Adapter) ExecuteQuery(ctx context.Context, sqlQuery string, opts mcp.QueryOptions) (*mcp.QueryResult, error) {
	if err := a.ValidateQuery(sqlQuery); err != nil {
		return nil, err
	}

	// SQL Server uses TOP instead of LIMIT
	sqlQuery = enforceSQLServerLimit(sqlQuery, opts.MaxRows)

	// Create context with timeout
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	rows, err := a.db.QueryContext(ctx, sqlQuery)
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

// enforceSQLServerLimit ensures the query has a TOP clause if no OFFSET/FETCH or TOP is present
func enforceSQLServerLimit(sqlQuery string, maxRows int) string {
	normalized := strings.ToUpper(sqlQuery)

	// Check if TOP, OFFSET, or FETCH already exists
	if strings.Contains(normalized, "TOP") ||
		strings.Contains(normalized, "OFFSET") ||
		strings.Contains(normalized, "FETCH") {
		return sqlQuery
	}

	// Remove trailing semicolon
	sqlQuery = strings.TrimSuffix(strings.TrimSpace(sqlQuery), ";")

	// Wrap in SELECT TOP N * FROM (original query)
	return fmt.Sprintf("SELECT TOP %d * FROM (%s) AS __limited", maxRows, sqlQuery)
}
