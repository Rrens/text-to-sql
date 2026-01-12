package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rensmac/text-to-sql/internal/domain"
)

// ConnectionRepository handles database connection data access
type ConnectionRepository struct {
	db *DB
}

// NewConnectionRepository creates a new connection repository
func NewConnectionRepository(db *DB) *ConnectionRepository {
	return &ConnectionRepository{db: db}
}

// Create creates a new connection
func (r *ConnectionRepository) Create(ctx context.Context, conn *domain.Connection) error {
	query := `
		INSERT INTO connections (
			id, workspace_id, name, database_type, host, port, 
			database_name, username, credentials_encrypted, ssl_mode,
			read_only, max_rows, timeout_seconds, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		conn.ID,
		conn.WorkspaceID,
		conn.Name,
		conn.DatabaseType,
		conn.Host,
		conn.Port,
		conn.Database,
		conn.Username,
		conn.CredentialsEncrypted,
		conn.SSLMode,
		conn.ReadOnly,
		conn.MaxRows,
		conn.TimeoutSeconds,
		conn.CreatedAt,
		conn.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create connection: %w", err)
	}

	return nil
}

// GetByID retrieves a connection by ID
func (r *ConnectionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Connection, error) {
	query := `
		SELECT 
			id, workspace_id, name, database_type, host, port,
			database_name, username, credentials_encrypted, ssl_mode,
			read_only, max_rows, timeout_seconds, created_at, updated_at
		FROM connections
		WHERE id = $1
	`

	var conn domain.Connection
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&conn.ID,
		&conn.WorkspaceID,
		&conn.Name,
		&conn.DatabaseType,
		&conn.Host,
		&conn.Port,
		&conn.Database,
		&conn.Username,
		&conn.CredentialsEncrypted,
		&conn.SSLMode,
		&conn.ReadOnly,
		&conn.MaxRows,
		&conn.TimeoutSeconds,
		&conn.CreatedAt,
		&conn.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	return &conn, nil
}

// GetByIDAndWorkspace retrieves a connection by ID and workspace
func (r *ConnectionRepository) GetByIDAndWorkspace(ctx context.Context, id, workspaceID uuid.UUID) (*domain.Connection, error) {
	query := `
		SELECT 
			id, workspace_id, name, database_type, host, port,
			database_name, username, credentials_encrypted, ssl_mode,
			read_only, max_rows, timeout_seconds, created_at, updated_at
		FROM connections
		WHERE id = $1 AND workspace_id = $2
	`

	var conn domain.Connection
	err := r.db.Pool.QueryRow(ctx, query, id, workspaceID).Scan(
		&conn.ID,
		&conn.WorkspaceID,
		&conn.Name,
		&conn.DatabaseType,
		&conn.Host,
		&conn.Port,
		&conn.Database,
		&conn.Username,
		&conn.CredentialsEncrypted,
		&conn.SSLMode,
		&conn.ReadOnly,
		&conn.MaxRows,
		&conn.TimeoutSeconds,
		&conn.CreatedAt,
		&conn.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	return &conn, nil
}

// ListByWorkspace retrieves all connections for a workspace
func (r *ConnectionRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.Connection, error) {
	query := `
		SELECT 
			id, workspace_id, name, database_type, host, port,
			database_name, username, credentials_encrypted, ssl_mode,
			read_only, max_rows, timeout_seconds, created_at, updated_at
		FROM connections
		WHERE workspace_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections: %w", err)
	}
	defer rows.Close()

	var connections []domain.Connection
	for rows.Next() {
		var conn domain.Connection
		if err := rows.Scan(
			&conn.ID,
			&conn.WorkspaceID,
			&conn.Name,
			&conn.DatabaseType,
			&conn.Host,
			&conn.Port,
			&conn.Database,
			&conn.Username,
			&conn.CredentialsEncrypted,
			&conn.SSLMode,
			&conn.ReadOnly,
			&conn.MaxRows,
			&conn.TimeoutSeconds,
			&conn.CreatedAt,
			&conn.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan connection: %w", err)
		}
		connections = append(connections, conn)
	}

	return connections, nil
}

// Update updates a connection
func (r *ConnectionRepository) Update(ctx context.Context, id uuid.UUID, conn *domain.Connection) error {
	query := `
		UPDATE connections
		SET name = $2,
		    host = $3,
		    port = $4,
		    database_name = $5,
		    username = $6,
		    credentials_encrypted = $7,
		    ssl_mode = $8,
		    read_only = $9,
		    max_rows = $10,
		    timeout_seconds = $11,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Pool.Exec(ctx, query,
		id,
		conn.Name,
		conn.Host,
		conn.Port,
		conn.Database,
		conn.Username,
		conn.CredentialsEncrypted,
		conn.SSLMode,
		conn.ReadOnly,
		conn.MaxRows,
		conn.TimeoutSeconds,
	)
	if err != nil {
		return fmt.Errorf("failed to update connection: %w", err)
	}

	return nil
}

// Delete deletes a connection
func (r *ConnectionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM connections WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	return nil
}
