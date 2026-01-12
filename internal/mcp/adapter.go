package mcp

import (
	"context"
	"time"
)

// TableInfo contains table metadata
type TableInfo struct {
	Name       string       `json:"name"`
	SchemaName string       `json:"schema_name,omitempty"`
	Columns    []ColumnInfo `json:"columns"`
	RowCount   *int64       `json:"row_count,omitempty"`
}

// ColumnInfo contains column metadata
type ColumnInfo struct {
	Name        string `json:"name"`
	DataType    string `json:"data_type"`
	Nullable    bool   `json:"nullable"`
	PrimaryKey  bool   `json:"primary_key"`
	Description string `json:"description,omitempty"`
}

// QueryResult contains query execution result
type QueryResult struct {
	Columns   []string `json:"columns"`
	Rows      [][]any  `json:"rows"`
	RowCount  int      `json:"row_count"`
	Truncated bool     `json:"truncated"`
}

// ConnectionConfig contains database connection parameters
type ConnectionConfig struct {
	Host           string
	Port           int
	Database       string
	Username       string
	Password       string
	SSLMode        string
	MaxRows        int
	TimeoutSeconds int
}

// QueryOptions contains query execution options
type QueryOptions struct {
	MaxRows int
	Timeout time.Duration
}

// Adapter defines the interface for database adapters
type Adapter interface {
	// DatabaseType returns the database type identifier (postgres, clickhouse, mysql)
	DatabaseType() string

	// SQLDialect returns SQL dialect hints for LLM prompting
	SQLDialect() string

	// Connect establishes connection to database
	Connect(ctx context.Context, config ConnectionConfig) error

	// Close closes the connection
	Close() error

	// HealthCheck verifies connection is alive
	HealthCheck(ctx context.Context) error

	// ListTables returns list of table names
	ListTables(ctx context.Context) ([]string, error)

	// DescribeTable returns detailed table schema
	DescribeTable(ctx context.Context, tableName string) (*TableInfo, error)

	// GetSchemaDDL returns full schema as DDL for LLM context
	GetSchemaDDL(ctx context.Context) (string, error)

	// ValidateQuery validates SQL is safe to execute
	ValidateQuery(sql string) error

	// ExecuteQuery executes read-only SQL query
	ExecuteQuery(ctx context.Context, sql string, opts QueryOptions) (*QueryResult, error)
}

// AdapterFactory creates a new adapter instance
type AdapterFactory func() Adapter
