package domain

import (
	"time"

	"github.com/google/uuid"
)

// QueryRequest represents a text-to-SQL query request
type QueryRequest struct {
	ConnectionID uuid.UUID     `json:"connection_id" validate:"required"`
	Question     string        `json:"question" validate:"required,max=2000"`
	LLMProvider  string        `json:"llm_provider" validate:"omitempty,oneof=openai anthropic ollama deepseek gemini"`
	LLMModel     string        `json:"llm_model,omitempty"`
	Execute      bool          `json:"execute"`
	Options      *QueryOptions `json:"options,omitempty"`
}

// QueryOptions represents optional query parameters
type QueryOptions struct {
	MaxRows        int `json:"max_rows" validate:"omitempty,min=1,max=10000"`
	TimeoutSeconds int `json:"timeout_seconds" validate:"omitempty,min=1,max=300"`
}

// QueryResponse represents query execution result
type QueryResponse struct {
	RequestID string         `json:"request_id"`
	Question  string         `json:"question"`
	SQL       string         `json:"sql"`
	Result    *QueryResult   `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
	Metadata  *QueryMetadata `json:"metadata"`
}

// QueryResult contains query execution data
type QueryResult struct {
	Columns   []string `json:"columns"`
	Rows      [][]any  `json:"rows"`
	RowCount  int      `json:"row_count"`
	Truncated bool     `json:"truncated"`
}

// QueryMetadata contains query execution metadata
type QueryMetadata struct {
	ConnectionID    uuid.UUID `json:"connection_id"`
	DatabaseType    string    `json:"database_type"`
	LLMProvider     string    `json:"llm_provider"`
	LLMModel        string    `json:"llm_model"`
	ExecutionTimeMs int64     `json:"execution_time_ms"`
	LLMLatencyMs    int64     `json:"llm_latency_ms"`
	TokensUsed      int       `json:"tokens_used"`
}

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

// SchemaInfo contains database schema information
type SchemaInfo struct {
	DatabaseType string      `json:"database_type"`
	Tables       []TableInfo `json:"tables"`
	DDL          string      `json:"ddl"`
	CachedAt     time.Time   `json:"cached_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           uuid.UUID      `json:"id"`
	WorkspaceID  uuid.UUID      `json:"workspace_id"`
	UserID       uuid.UUID      `json:"user_id"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type,omitempty"`
	ResourceID   *uuid.UUID     `json:"resource_id,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	IPAddress    string         `json:"ip_address,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// Audit actions
const (
	AuditActionLogin            = "login"
	AuditActionLogout           = "logout"
	AuditActionConnectionCreate = "connection.create"
	AuditActionConnectionDelete = "connection.delete"
	AuditActionQueryExecute     = "query.execute"
	AuditActionSchemaRefresh    = "schema.refresh"
)
