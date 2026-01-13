package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// MessageRole represents the sender of a message
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

// Message represents a chat message in a workspace
type Message struct {
	ID          uuid.UUID   `json:"id"`
	WorkspaceID uuid.UUID   `json:"workspace_id"`
	UserID      *uuid.UUID  `json:"user_id,omitempty"` // Null for assistant messages
	SessionID   *uuid.UUID  `json:"session_id,omitempty"`
	Role        MessageRole `json:"role"`
	Content     string      `json:"content"`
	SQL         string      `json:"sql,omitempty"`
	Result      any         `json:"result,omitempty"`
	Metadata    any         `json:"metadata,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

// MessageRepository defines the interface for message storage
type MessageRepository interface {
	Create(ctx context.Context, message *Message) error
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit int) ([]Message, error)
	ListBySession(ctx context.Context, sessionID uuid.UUID, limit int) ([]Message, error)
	GetMostFrequentQuestions(ctx context.Context, workspaceID uuid.UUID, limit int) ([]string, error)
}
