package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ChatSession represents a conversation thread in a workspace
type ChatSession struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	Title       string     `json:"title"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// SessionRepository defines the interface for session storage
type SessionRepository interface {
	Create(ctx context.Context, session *ChatSession) error
	Get(ctx context.Context, id uuid.UUID) (*ChatSession, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit int, offset int) ([]ChatSession, error)
	Update(ctx context.Context, session *ChatSession) error
	Delete(ctx context.Context, id uuid.UUID) error
}
