package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Rrens/text-to-sql/internal/domain"
	"github.com/Rrens/text-to-sql/internal/llm"
	"github.com/Rrens/text-to-sql/internal/mcp"
	"github.com/Rrens/text-to-sql/internal/repository/redis"
	"github.com/google/uuid"
)

// QueryService handles text-to-SQL query operations
type QueryService struct {
	connectionService *ConnectionService
	mcpRouter         *mcp.Router
	llmRouter         *llm.Router
	schemaCache       *redis.SchemaCache
}

// NewQueryService creates a new query service
func NewQueryService(
	connectionService *ConnectionService,
	mcpRouter *mcp.Router,
	llmRouter *llm.Router,
	schemaCache *redis.SchemaCache,
) *QueryService {
	return &QueryService{
		connectionService: connectionService,
		mcpRouter:         mcpRouter,
		llmRouter:         llmRouter,
		schemaCache:       schemaCache,
	}
}

// ExecuteQuery processes a text-to-SQL query
func (s *QueryService) ExecuteQuery(ctx context.Context, userID, workspaceID uuid.UUID, req domain.QueryRequest) (*domain.QueryResponse, error) {
	requestID := uuid.New().String()
	startTime := time.Now()

	// Get connection with decrypted credentials
	conn, password, err := s.connectionService.GetFullConnection(ctx, userID, workspaceID, req.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// Get or create MCP adapter
	mcpConfig := mcp.ConnectionConfig{
		Host:           conn.Host,
		Port:           conn.Port,
		Database:       conn.Database,
		Username:       conn.Username,
		Password:       password,
		SSLMode:        conn.SSLMode,
		MaxRows:        conn.MaxRows,
		TimeoutSeconds: conn.TimeoutSeconds,
	}

	adapter, err := s.mcpRouter.GetAdapter(ctx, conn.ID, string(conn.DatabaseType), mcpConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get database adapter: %w", err)
	}

	// Get schema (from cache or refresh)
	schema, err := s.getSchema(ctx, conn.ID, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	// Get LLM provider
	providerName := req.LLMProvider
	if providerName == "" {
		providerName = s.llmRouter.DefaultProvider()
	}

	provider, err := s.llmRouter.GetProvider(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM provider: %w", err)
	}

	// Generate SQL
	llmReq := llm.Request{
		Question:     req.Question,
		SchemaDDL:    schema.DDL,
		SQLDialect:   adapter.SQLDialect(),
		DatabaseType: adapter.DatabaseType(),
	}

	modelName := req.LLMModel
	if modelName == "" {
		modelName = provider.DefaultModel()
	}

	llmStart := time.Now()
	llmResp, err := provider.GenerateSQL(ctx, llmReq, modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQL: %w", err)
	}
	llmLatency := time.Since(llmStart).Milliseconds()

	response := &domain.QueryResponse{
		RequestID: requestID,
		Question:  req.Question,
		SQL:       llmResp.SQL,
		Metadata: &domain.QueryMetadata{
			ConnectionID: conn.ID,
			DatabaseType: string(conn.DatabaseType),
			LLMProvider:  providerName,
			LLMModel:     modelName,
			LLMLatencyMs: llmLatency,
			TokensUsed:   llmResp.TokensUsed,
		},
	}

	// Execute query if requested
	if req.Execute {
		maxRows := conn.MaxRows
		timeout := time.Duration(conn.TimeoutSeconds) * time.Second

		if req.Options != nil {
			if req.Options.MaxRows > 0 && req.Options.MaxRows < maxRows {
				maxRows = req.Options.MaxRows
			}
			if req.Options.TimeoutSeconds > 0 {
				timeout = time.Duration(req.Options.TimeoutSeconds) * time.Second
			}
		}

		queryOpts := mcp.QueryOptions{
			MaxRows: maxRows,
			Timeout: timeout,
		}

		result, err := adapter.ExecuteQuery(ctx, llmResp.SQL, queryOpts)
		if err != nil {
			response.Error = err.Error()
		} else {
			response.Result = &domain.QueryResult{
				Columns:   result.Columns,
				Rows:      result.Rows,
				RowCount:  result.RowCount,
				Truncated: result.Truncated,
			}
		}
	}

	response.Metadata.ExecutionTimeMs = time.Since(startTime).Milliseconds()

	return response, nil
}

// getSchema retrieves schema from cache or database
func (s *QueryService) getSchema(ctx context.Context, connectionID uuid.UUID, adapter mcp.Adapter) (*domain.SchemaInfo, error) {
	// Try cache first
	if s.schemaCache != nil {
		cached, err := s.schemaCache.Get(ctx, connectionID)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	// Get from database
	tables, err := adapter.ListTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	var tableInfos []domain.TableInfo
	for _, tableName := range tables {
		tableInfo, err := adapter.DescribeTable(ctx, tableName)
		if err != nil {
			continue // Skip tables we can't describe
		}

		columns := make([]domain.ColumnInfo, len(tableInfo.Columns))
		for i, col := range tableInfo.Columns {
			columns[i] = domain.ColumnInfo{
				Name:        col.Name,
				DataType:    col.DataType,
				Nullable:    col.Nullable,
				PrimaryKey:  col.PrimaryKey,
				Description: col.Description,
			}
		}

		tableInfos = append(tableInfos, domain.TableInfo{
			Name:       tableInfo.Name,
			SchemaName: tableInfo.SchemaName,
			Columns:    columns,
			RowCount:   tableInfo.RowCount,
		})
	}

	ddl, err := adapter.GetSchemaDDL(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DDL: %w", err)
	}

	schema := &domain.SchemaInfo{
		DatabaseType: adapter.DatabaseType(),
		Tables:       tableInfos,
		DDL:          ddl,
		CachedAt:     time.Now(),
	}

	// Cache the schema
	if s.schemaCache != nil {
		s.schemaCache.Set(ctx, connectionID, schema)
	}

	return schema, nil
}

// RefreshSchema forces a schema refresh for a connection
func (s *QueryService) RefreshSchema(ctx context.Context, userID, workspaceID, connectionID uuid.UUID) (*domain.SchemaInfo, error) {
	// Invalidate cache
	if s.schemaCache != nil {
		s.schemaCache.Invalidate(ctx, connectionID)
	}

	// Get connection
	conn, password, err := s.connectionService.GetFullConnection(ctx, userID, workspaceID, connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// Get adapter
	mcpConfig := mcp.ConnectionConfig{
		Host:     conn.Host,
		Port:     conn.Port,
		Database: conn.Database,
		Username: conn.Username,
		Password: password,
		SSLMode:  conn.SSLMode,
	}

	adapter, err := s.mcpRouter.GetAdapter(ctx, conn.ID, string(conn.DatabaseType), mcpConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get adapter: %w", err)
	}

	return s.getSchema(ctx, connectionID, adapter)
}

// GetSchema returns cached or fresh schema for a connection
func (s *QueryService) GetSchema(ctx context.Context, userID, workspaceID, connectionID uuid.UUID) (*domain.SchemaInfo, error) {
	// Try cache first
	if s.schemaCache != nil {
		cached, err := s.schemaCache.Get(ctx, connectionID)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	// Refresh if not cached
	return s.RefreshSchema(ctx, userID, workspaceID, connectionID)
}
