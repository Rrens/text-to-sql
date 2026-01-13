package service

import (
	"context"

	"github.com/Rrens/text-to-sql/internal/domain"
	"github.com/Rrens/text-to-sql/internal/llm"
	"github.com/Rrens/text-to-sql/internal/mcp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockMessageRepository mocks the MessageRepository interface
type MockMessageRepository struct {
	mock.Mock
}

func (m *MockMessageRepository) Create(ctx context.Context, message *domain.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockMessageRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit int) ([]domain.Message, error) {
	args := m.Called(ctx, workspaceID, limit)
	return args.Get(0).([]domain.Message), args.Error(1)
}

func (m *MockMessageRepository) ListBySession(ctx context.Context, sessionID uuid.UUID, limit int) ([]domain.Message, error) {
	args := m.Called(ctx, sessionID, limit)
	return args.Get(0).([]domain.Message), args.Error(1)
}

func (m *MockMessageRepository) GetMostFrequentQuestions(ctx context.Context, workspaceID uuid.UUID, limit int) ([]string, error) {
	args := m.Called(ctx, workspaceID, limit)
	return args.Get(0).([]string), args.Error(1)
}

// MockSessionRepository mocks the SessionRepository interface
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(ctx context.Context, session *domain.ChatSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) Get(ctx context.Context, id uuid.UUID) (*domain.ChatSession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ChatSession), args.Error(1)
}

func (m *MockSessionRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit int, offset int) ([]domain.ChatSession, error) {
	args := m.Called(ctx, workspaceID, limit, offset)
	return args.Get(0).([]domain.ChatSession), args.Error(1)
}

func (m *MockSessionRepository) Update(ctx context.Context, session *domain.ChatSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockConnectionRepository mocks the ConnectionRepository
type MockConnectionRepository struct {
	mock.Mock
}

func (m *MockConnectionRepository) Create(ctx context.Context, conn *domain.Connection) error {
	args := m.Called(ctx, conn)
	return args.Error(0)
}

func (m *MockConnectionRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Connection, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Connection), args.Error(1)
}

func (m *MockConnectionRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.Connection, error) {
	args := m.Called(ctx, workspaceID)
	return args.Get(0).([]domain.Connection), args.Error(1)
}

func (m *MockConnectionRepository) Update(ctx context.Context, conn *domain.Connection) error {
	args := m.Called(ctx, conn)
	return args.Error(0)
}

func (m *MockConnectionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockWorkspaceRepository mocks WorkspaceRepository
type MockWorkspaceRepository struct {
	mock.Mock
}

func (m *MockWorkspaceRepository) Create(ctx context.Context, workspace *domain.Workspace) error {
	args := m.Called(ctx, workspace)
	return args.Error(0)
}

func (m *MockWorkspaceRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Workspace, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Workspace), args.Error(1)
}

func (m *MockWorkspaceRepository) Update(ctx context.Context, id uuid.UUID, update *domain.WorkspaceUpdate) error {
	args := m.Called(ctx, id, update)
	return args.Error(0)
}

func (m *MockWorkspaceRepository) AddMember(ctx context.Context, member *domain.WorkspaceMember) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockWorkspaceRepository) GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*domain.WorkspaceMember, error) {
	args := m.Called(ctx, workspaceID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkspaceMember), args.Error(1)
}

func (m *MockWorkspaceRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]domain.Workspace), args.Error(1)
}

// MockLLMProvider mocks llm.Provider
type MockLLMProvider struct {
	mock.Mock
}

func (m *MockLLMProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockLLMProvider) AvailableModels() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockLLMProvider) DefaultModel() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockLLMProvider) IsConfigured() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockLLMProvider) GenerateSQL(ctx context.Context, req llm.Request, model string) (*llm.Response, error) {
	args := m.Called(ctx, req, model)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llm.Response), args.Error(1)
}

func (m *MockLLMProvider) GenerateTitle(ctx context.Context, question string, model string) (string, error) {
	args := m.Called(ctx, question, model)
	return args.String(0), args.Error(1)
}

// MockMCPAdapter mocks mcp.Adapter
type MockMCPAdapter struct {
	mock.Mock
}

func (m *MockMCPAdapter) DatabaseType() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMCPAdapter) SQLDialect() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMCPAdapter) Connect(ctx context.Context, config mcp.ConnectionConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockMCPAdapter) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMCPAdapter) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMCPAdapter) ListTables(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockMCPAdapter) DescribeTable(ctx context.Context, tableName string) (*mcp.TableInfo, error) {
	args := m.Called(ctx, tableName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mcp.TableInfo), args.Error(1)
}

func (m *MockMCPAdapter) GetSchemaDDL(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockMCPAdapter) ValidateQuery(sql string) error {
	args := m.Called(sql)
	return args.Error(0)
}

func (m *MockMCPAdapter) ExecuteQuery(ctx context.Context, sql string, opts mcp.QueryOptions) (*mcp.QueryResult, error) {
	args := m.Called(ctx, sql, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mcp.QueryResult), args.Error(1)
}
