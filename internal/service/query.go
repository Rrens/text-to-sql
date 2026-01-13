package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Rrens/text-to-sql/internal/domain"
	"github.com/Rrens/text-to-sql/internal/llm"
	"github.com/Rrens/text-to-sql/internal/mcp"
	"github.com/Rrens/text-to-sql/internal/repository/postgres"
	"github.com/Rrens/text-to-sql/internal/repository/redis"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// QueryService handles text-to-SQL query operations
type QueryService struct {
	connectionService *ConnectionService
	mcpRouter         *mcp.Router
	llmRouter         *llm.Router
	schemaCache       *redis.SchemaCache
	messageRepo       domain.MessageRepository
	sessionRepo       domain.SessionRepository
	userRepo          *postgres.UserRepository
}

// NewQueryService creates a new query service
func NewQueryService(
	connectionService *ConnectionService,
	mcpRouter *mcp.Router,
	llmRouter *llm.Router,
	schemaCache *redis.SchemaCache,
	messageRepo domain.MessageRepository,
	sessionRepo domain.SessionRepository,
	userRepo *postgres.UserRepository,
) *QueryService {
	return &QueryService{
		connectionService: connectionService,
		mcpRouter:         mcpRouter,
		llmRouter:         llmRouter,
		schemaCache:       schemaCache,
		messageRepo:       messageRepo,
		sessionRepo:       sessionRepo,
		userRepo:          userRepo,
	}
}

// ExecuteQuery processes a text-to-SQL query
func (s *QueryService) ExecuteQuery(ctx context.Context, userID, workspaceID uuid.UUID, req domain.QueryRequest) (*domain.QueryResponse, error) {
	requestID := uuid.New().String()
	startTime := time.Now()

	// 1. Handle Session
	// 1. Handle Session
	var sessionID uuid.UUID
	var isNewSession bool
	if req.SessionID != uuid.Nil {
		sessionID = req.SessionID
		// Verify session exists/belongs to user? (Optional but good)
	} else {
		isNewSession = true
		// Create new session
		sessionID = uuid.New()
		newSession := &domain.ChatSession{
			ID:          sessionID,
			WorkspaceID: workspaceID,
			UserID:      &userID,
			Title:       "New Chat", // Will be updated async
			CreatedAt:   startTime,
			UpdatedAt:   startTime,
		}
		if err := s.sessionRepo.Create(ctx, newSession); err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	}

	// 2. Save User Question
	userMsg := &domain.Message{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		UserID:      &userID,
		SessionID:   &sessionID,
		Role:        domain.RoleUser,
		Content:     req.Question,
		CreatedAt:   startTime,
	}
	if err := s.messageRepo.Create(ctx, userMsg); err != nil {
		// Log error but continue execution
		log.Error().Err(err).Msg("failed to save user message")
	}

	// 3. Fetch Chat History (last 10 messages from this session)
	history, err := s.messageRepo.ListBySession(ctx, sessionID, 10)
	if err != nil {
		// log.Error().Err(err).Msg("failed to fetch chat history")
		history = []domain.Message{}
	}

	// Get connection with decrypted credentials
	conn, password, err := s.connectionService.GetFullConnection(ctx, userID, workspaceID, req.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// ... (Get MCP Adapter logic remains same)
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

	// Fetch user config for LLM
	var llmConfig map[string]any
	user, err := s.userRepo.GetByID(ctx, userID)
	if err == nil && user != nil && user.LLMConfig != nil {
		if config, ok := user.LLMConfig[providerName].(map[string]any); ok {
			llmConfig = config
		}
	}

	provider, err := s.llmRouter.GetProviderWithConfig(providerName, llmConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM provider: %w", err)
	}

	// Generate SQL
	llmReq := llm.Request{
		Question:     req.Question,
		SchemaDDL:    schema.DDL,
		SQLDialect:   adapter.SQLDialect(),
		DatabaseType: adapter.DatabaseType(),
		History:      history, // Pass history to LLM
	}

	// DEBUG: Log schema DDL length
	log.Debug().
		Int("schema_ddl_length", len(schema.DDL)).
		Str("question", req.Question).
		Msg("Preparing LLM request")

	modelName := req.LLMModel
	if modelName == "" {
		modelName = provider.DefaultModel()
	}

	// llmStart := time.Now()
	llmResp, err := provider.GenerateSQL(ctx, llmReq, modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQL: %w", err)
	}
	// Calculate total execution time
	// executionTime := time.Since(startTime).Milliseconds()

	// DEBUG: Log LLM response
	log.Debug().
		Str("sql", llmResp.SQL).
		Str("explanation", llmResp.Explanation).
		Int("tokens_used", llmResp.TokensUsed).
		Msg("LLM response received")

	response := &domain.QueryResponse{
		RequestID:   requestID,
		SessionID:   sessionID,
		Question:    req.Question,
		SQL:         llmResp.SQL,
		Explanation: llmResp.Explanation,
		Metadata: &domain.QueryMetadata{
			ConnectionID:    req.ConnectionID,
			DatabaseType:    string(conn.DatabaseType),
			LLMProvider:     providerName,
			LLMModel:        modelName,
			ExecutionTimeMs: time.Since(startTime).Milliseconds(),
			LLMLatencyMs:    llmResp.LatencyMs,
			TokensUsed:      llmResp.TokensUsed,
		},
	}

	// 3. Execute query if requested
	if req.Execute && llmResp.SQL != "" {
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

	// 4. Save Assistant Response (now with full context)
	// Ensure content is not empty
	content := llmResp.Explanation
	if content == "" {
		if response.Error != "" {
			content = fmt.Sprintf("I encountered an error: %s", response.Error)
		} else {
			content = "Here is the result of your query:"
		}
	}

	aiMsg := &domain.Message{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		SessionID:   &sessionID,
		Role:        domain.RoleAssistant,
		Content:     content,
		SQL:         llmResp.SQL,
		Result:      response.Result,
		Metadata:    response.Metadata,
		CreatedAt:   time.Now(),
	}
	if err := s.messageRepo.Create(ctx, aiMsg); err != nil {
		log.Error().Err(err).Msg("failed to save AI message")
	}

	// Update session timestamp
	// Optimization: could be async
	// I'll make a specialized UpdateTimestamp method.
	// Or I'll just ignore for now and let it be created_at based.
	// Actually, having updated_at for sorting sessions is important.
	// Let's quickly fetch and update.
	if sess, err := s.sessionRepo.Get(ctx, sessionID); err == nil {
		sess.UpdatedAt = time.Now()
		// Auto-update title if it's "New Chat" and we have a question
		if sess.Title == "New Chat" {
			if len(req.Question) > 30 {
				sess.Title = req.Question[:30] + "..."
			} else {
				sess.Title = req.Question
			}
		}
		s.sessionRepo.Update(ctx, sess)
	}

	// Trigger async title	// 4. Update session title if needed (async)
	if isNewSession {
		go s.generateSessionTitle(context.Background(), sessionID, req.Question, providerName, modelName)
	}

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

// GetChatHistory returns chat history for a workspace
func (s *QueryService) GetChatHistory(ctx context.Context, workspaceID uuid.UUID) ([]domain.Message, error) {
	// 50 messages limit for now
	return s.messageRepo.ListByWorkspace(ctx, workspaceID, 50)
}

// CreateSession creates a new chat session
func (s *QueryService) CreateSession(ctx context.Context, userID, workspaceID uuid.UUID, title string) (*domain.ChatSession, error) {
	if title == "" {
		title = "New Chat"
	}
	session := &domain.ChatSession{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		UserID:      &userID,
		Title:       title,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	return session, nil
}

// ListSessions lists chat sessions for a workspace
func (s *QueryService) ListSessions(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]domain.ChatSession, error) {
	return s.sessionRepo.ListByWorkspace(ctx, workspaceID, limit, offset)
}

// GetSession retrieves a chat session
func (s *QueryService) GetSession(ctx context.Context, sessionID uuid.UUID) (*domain.ChatSession, error) {
	return s.sessionRepo.Get(ctx, sessionID)
}

// DeleteSession deletes a chat session
func (s *QueryService) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	return s.sessionRepo.Delete(ctx, sessionID)
}

// GetSessionHistory retrieves chat history for a session
func (s *QueryService) GetSessionHistory(ctx context.Context, sessionID uuid.UUID) ([]domain.Message, error) {
	// 50 messages limit for now
	return s.messageRepo.ListBySession(ctx, sessionID, 50)
}

// generateSessionTitle generates and updates the session title using LLM
func (s *QueryService) generateSessionTitle(ctx context.Context, sessionID uuid.UUID, question string, providerName string, modelName string) {
	// 1. Get LLM provider
	if providerName == "" {
		providerName = s.llmRouter.DefaultProvider()
	}

	// Fetch user config for LLM (need userID from session)
	// Since we only have sessionID here, we first get the session to find userID
	session, err := s.sessionRepo.Get(ctx, sessionID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get session for title generation")
		return
	}
	if session.UserID == nil {
		// Anonymous session? fallback to system default
		log.Warn().Msg("session has no user ID, using default config")
	}

	var llmConfig map[string]any
	if session.UserID != nil {
		user, err := s.userRepo.GetByID(ctx, *session.UserID)
		if err == nil && user != nil && user.LLMConfig != nil {
			if config, ok := user.LLMConfig[providerName].(map[string]any); ok {
				llmConfig = config
			}
		}
	}

	provider, err := s.llmRouter.GetProviderWithConfig(providerName, llmConfig)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Msg("failed to get LLM provider for title generation")
		return
	}

	// 2. Generate title
	// Use a reasonable timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if modelName == "" {
		modelName = provider.DefaultModel()
	}
	title, err := provider.GenerateTitle(ctx, question, modelName)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate session title")
		return
	}

	// 3. Update session (we already fetched it)
	session.Title = title
	session.UpdatedAt = time.Now()

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		log.Error().Err(err).Msg("failed to update session title")
	}

	log.Info().Str("session_id", sessionID.String()).Str("title", title).Msg("updated session title")
}

// GetSuggestedQuestions retrieves suggested questions based on frequency
func (s *QueryService) GetSuggestedQuestions(ctx context.Context, workspaceID uuid.UUID) ([]string, error) {
	// Limit to top 5 frequent questions
	return s.messageRepo.GetMostFrequentQuestions(ctx, workspaceID, 5)
}
