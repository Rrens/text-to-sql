package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// DatabaseType represents supported database types
type DatabaseType string

const (
	DatabaseTypePostgres   DatabaseType = "postgres"
	DatabaseTypeClickHouse DatabaseType = "clickhouse"
	DatabaseTypeMySQL      DatabaseType = "mysql"
)

// WorkspaceRepository defines the interface for workspace storage
type WorkspaceRepository interface {
	Create(ctx context.Context, workspace *Workspace) error
	GetByID(ctx context.Context, id uuid.UUID) (*Workspace, error)
	Update(ctx context.Context, id uuid.UUID, update *WorkspaceUpdate) error
	AddMember(ctx context.Context, member *WorkspaceMember) error
	GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*WorkspaceMember, error)
	IsMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]Workspace, error)
}

// Connection represents a database connection configuration
type Connection struct {
	ID                   uuid.UUID    `json:"id"`
	WorkspaceID          uuid.UUID    `json:"workspace_id"`
	Name                 string       `json:"name"`
	DatabaseType         DatabaseType `json:"database_type"`
	Host                 string       `json:"host"`
	Port                 int          `json:"port"`
	Database             string       `json:"database"`
	Username             string       `json:"username"`
	CredentialsEncrypted []byte       `json:"-"`
	SSLMode              string       `json:"ssl_mode"`
	ReadOnly             bool         `json:"read_only"`
	MaxRows              int          `json:"max_rows"`
	TimeoutSeconds       int          `json:"timeout_seconds"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

// ConnectionCreate represents connection creation data
type ConnectionCreate struct {
	Name           string       `json:"name" validate:"required,max=255"`
	DatabaseType   DatabaseType `json:"database_type" validate:"required,oneof=postgres clickhouse mysql"`
	Host           string       `json:"host" validate:"required,max=255"`
	Port           int          `json:"port" validate:"required,min=1,max=65535"`
	Database       string       `json:"database" validate:"required,max=255"`
	Username       string       `json:"username" validate:"required,max=255"`
	Password       string       `json:"password" validate:"required"`
	SSLMode        string       `json:"ssl_mode" validate:"omitempty,oneof=disable require verify-ca verify-full"`
	ReadOnly       bool         `json:"read_only"`
	MaxRows        int          `json:"max_rows" validate:"omitempty,min=1,max=10000"`
	TimeoutSeconds int          `json:"timeout_seconds" validate:"omitempty,min=1,max=300"`
}

// ConnectionUpdate represents connection update data
type ConnectionUpdate struct {
	Name           *string `json:"name,omitempty" validate:"omitempty,max=255"`
	Host           *string `json:"host,omitempty" validate:"omitempty,max=255"`
	Port           *int    `json:"port,omitempty" validate:"omitempty,min=1,max=65535"`
	Database       *string `json:"database,omitempty" validate:"omitempty,max=255"`
	Username       *string `json:"username,omitempty" validate:"omitempty,max=255"`
	Password       *string `json:"password,omitempty"`
	SSLMode        *string `json:"ssl_mode,omitempty" validate:"omitempty,oneof=disable require verify-ca verify-full"`
	ReadOnly       *bool   `json:"read_only,omitempty"`
	MaxRows        *int    `json:"max_rows,omitempty" validate:"omitempty,min=1,max=10000"`
	TimeoutSeconds *int    `json:"timeout_seconds,omitempty" validate:"omitempty,min=1,max=300"`
}

// ConnectionInfo represents connection info without sensitive data
type ConnectionInfo struct {
	ID           uuid.UUID    `json:"id"`
	WorkspaceID  uuid.UUID    `json:"workspace_id"`
	Name         string       `json:"name"`
	DatabaseType DatabaseType `json:"database_type"`
	Host         string       `json:"host"`
	Port         int          `json:"port"`
	Database     string       `json:"database"`
	Username     string       `json:"username"`
	SSLMode      string       `json:"ssl_mode"`
	ReadOnly     bool         `json:"read_only"`
	MaxRows      int          `json:"max_rows"`
	CreatedAt    time.Time    `json:"created_at"`
}

// ConnectionRepository defines the interface for connection storage
type ConnectionRepository interface {
	Create(ctx context.Context, conn *Connection) error
	GetByID(ctx context.Context, id uuid.UUID) (*Connection, error)
	GetByIDAndWorkspace(ctx context.Context, id, workspaceID uuid.UUID) (*Connection, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]Connection, error)
	Update(ctx context.Context, id uuid.UUID, conn *Connection) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ToInfo converts Connection to ConnectionInfo (without sensitive data)
func (c *Connection) ToInfo() ConnectionInfo {
	return ConnectionInfo{
		ID:           c.ID,
		WorkspaceID:  c.WorkspaceID,
		Name:         c.Name,
		DatabaseType: c.DatabaseType,
		Host:         c.Host,
		Port:         c.Port,
		Database:     c.Database,
		Username:     c.Username,
		SSLMode:      c.SSLMode,
		ReadOnly:     c.ReadOnly,
		MaxRows:      c.MaxRows,
		CreatedAt:    c.CreatedAt,
	}
}
