package domain

import (
	"time"

	"github.com/google/uuid"
)

// Workspace represents a tenant workspace
type Workspace struct {
	ID        uuid.UUID      `json:"id"`
	Name      string         `json:"name"`
	Settings  map[string]any `json:"settings,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// WorkspaceCreate represents workspace creation data
type WorkspaceCreate struct {
	Name     string         `json:"name" validate:"required,max=255"`
	Settings map[string]any `json:"settings,omitempty"`
}

// WorkspaceUpdate represents workspace update data
type WorkspaceUpdate struct {
	Name     *string        `json:"name,omitempty" validate:"omitempty,max=255"`
	Settings map[string]any `json:"settings,omitempty"`
}

// WorkspaceMember represents workspace membership
type WorkspaceMember struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
	UserID      uuid.UUID `json:"user_id"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

// Role constants
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)
