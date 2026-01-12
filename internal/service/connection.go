package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rensmac/text-to-sql/internal/domain"
	"github.com/rensmac/text-to-sql/internal/repository/postgres"
	"github.com/rensmac/text-to-sql/internal/security"
)

// ConnectionService handles database connection operations
type ConnectionService struct {
	connectionRepo *postgres.ConnectionRepository
	workspaceRepo  *postgres.WorkspaceRepository
	encryptor      *security.Encryptor
	defaultMaxRows int
	defaultTimeout int
}

// NewConnectionService creates a new connection service
func NewConnectionService(
	connectionRepo *postgres.ConnectionRepository,
	workspaceRepo *postgres.WorkspaceRepository,
	encryptor *security.Encryptor,
	defaultMaxRows int,
	defaultTimeout int,
) *ConnectionService {
	return &ConnectionService{
		connectionRepo: connectionRepo,
		workspaceRepo:  workspaceRepo,
		encryptor:      encryptor,
		defaultMaxRows: defaultMaxRows,
		defaultTimeout: defaultTimeout,
	}
}

// Create creates a new database connection
func (s *ConnectionService) Create(ctx context.Context, userID, workspaceID uuid.UUID, input domain.ConnectionCreate) (*domain.ConnectionInfo, error) {
	// Check workspace access
	isMember, err := s.workspaceRepo.IsMember(ctx, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return nil, errors.New("access denied")
	}

	// Encrypt password
	credentials := map[string]string{"password": input.Password}
	encryptedCreds, err := s.encryptor.EncryptJSON(credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	// Set defaults
	maxRows := input.MaxRows
	if maxRows == 0 {
		maxRows = s.defaultMaxRows
	}
	timeout := input.TimeoutSeconds
	if timeout == 0 {
		timeout = s.defaultTimeout
	}
	sslMode := input.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	now := time.Now()
	conn := &domain.Connection{
		ID:                   uuid.New(),
		WorkspaceID:          workspaceID,
		Name:                 input.Name,
		DatabaseType:         input.DatabaseType,
		Host:                 input.Host,
		Port:                 input.Port,
		Database:             input.Database,
		Username:             input.Username,
		CredentialsEncrypted: encryptedCreds,
		SSLMode:              sslMode,
		ReadOnly:             input.ReadOnly,
		MaxRows:              maxRows,
		TimeoutSeconds:       timeout,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.connectionRepo.Create(ctx, conn); err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	info := conn.ToInfo()
	return &info, nil
}

// GetByID retrieves a connection by ID
func (s *ConnectionService) GetByID(ctx context.Context, userID, workspaceID, connectionID uuid.UUID) (*domain.ConnectionInfo, error) {
	// Check workspace access
	isMember, err := s.workspaceRepo.IsMember(ctx, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return nil, errors.New("access denied")
	}

	conn, err := s.connectionRepo.GetByIDAndWorkspace(ctx, connectionID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	if conn == nil {
		return nil, errors.New("connection not found")
	}

	info := conn.ToInfo()
	return &info, nil
}

// GetFullConnection retrieves a connection with decrypted credentials
func (s *ConnectionService) GetFullConnection(ctx context.Context, userID, workspaceID, connectionID uuid.UUID) (*domain.Connection, string, error) {
	// Check workspace access
	isMember, err := s.workspaceRepo.IsMember(ctx, workspaceID, userID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return nil, "", errors.New("access denied")
	}

	conn, err := s.connectionRepo.GetByIDAndWorkspace(ctx, connectionID, workspaceID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get connection: %w", err)
	}
	if conn == nil {
		return nil, "", errors.New("connection not found")
	}

	// Decrypt credentials
	var credentials map[string]string
	if err := s.encryptor.DecryptJSON(conn.CredentialsEncrypted, &credentials); err != nil {
		return nil, "", fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	return conn, credentials["password"], nil
}

// ListByWorkspace retrieves all connections for a workspace
func (s *ConnectionService) ListByWorkspace(ctx context.Context, userID, workspaceID uuid.UUID) ([]domain.ConnectionInfo, error) {
	// Check workspace access
	isMember, err := s.workspaceRepo.IsMember(ctx, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return nil, errors.New("access denied")
	}

	connections, err := s.connectionRepo.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections: %w", err)
	}

	infos := make([]domain.ConnectionInfo, len(connections))
	for i, conn := range connections {
		infos[i] = conn.ToInfo()
	}

	return infos, nil
}

// Update updates a connection
func (s *ConnectionService) Update(ctx context.Context, userID, workspaceID, connectionID uuid.UUID, input domain.ConnectionUpdate) (*domain.ConnectionInfo, error) {
	// Get existing connection
	conn, err := s.connectionRepo.GetByIDAndWorkspace(ctx, connectionID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	if conn == nil {
		return nil, errors.New("connection not found")
	}

	// Check workspace access
	isMember, err := s.workspaceRepo.IsMember(ctx, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return nil, errors.New("access denied")
	}

	// Apply updates
	if input.Name != nil {
		conn.Name = *input.Name
	}
	if input.Host != nil {
		conn.Host = *input.Host
	}
	if input.Port != nil {
		conn.Port = *input.Port
	}
	if input.Database != nil {
		conn.Database = *input.Database
	}
	if input.Username != nil {
		conn.Username = *input.Username
	}
	if input.Password != nil {
		credentials := map[string]string{"password": *input.Password}
		encryptedCreds, err := s.encryptor.EncryptJSON(credentials)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
		}
		conn.CredentialsEncrypted = encryptedCreds
	}
	if input.SSLMode != nil {
		conn.SSLMode = *input.SSLMode
	}
	if input.ReadOnly != nil {
		conn.ReadOnly = *input.ReadOnly
	}
	if input.MaxRows != nil {
		conn.MaxRows = *input.MaxRows
	}
	if input.TimeoutSeconds != nil {
		conn.TimeoutSeconds = *input.TimeoutSeconds
	}

	if err := s.connectionRepo.Update(ctx, connectionID, conn); err != nil {
		return nil, fmt.Errorf("failed to update connection: %w", err)
	}

	info := conn.ToInfo()
	return &info, nil
}

// Delete deletes a connection
func (s *ConnectionService) Delete(ctx context.Context, userID, workspaceID, connectionID uuid.UUID) error {
	// Check workspace access
	isMember, err := s.workspaceRepo.IsMember(ctx, workspaceID, userID)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return errors.New("access denied")
	}

	// Verify connection exists in workspace
	conn, err := s.connectionRepo.GetByIDAndWorkspace(ctx, connectionID, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	if conn == nil {
		return errors.New("connection not found")
	}

	return s.connectionRepo.Delete(ctx, connectionID)
}
