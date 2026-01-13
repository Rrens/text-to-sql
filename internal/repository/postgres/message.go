package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Rrens/text-to-sql/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MessageRepository implements domain.MessageRepository
type MessageRepository struct {
	pool *pgxpool.Pool
}

// NewMessageRepository creates a new message repository
func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{pool: pool}
}

// MessageRepository defines the interface for message storage (copying interface here to avoid import cycle if in domain, but it is in domain) - ah this is implementation.

// Create inserts a new message
func (r *MessageRepository) Create(ctx context.Context, message *domain.Message) error {
	query := `
		INSERT INTO chat_messages (id, workspace_id, user_id, session_id, role, content, sql, result, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	// Marshal metadata and result to JSON if needed
	var resultJSON, metadataJSON []byte
	if message.Result != nil {
		var err error
		resultJSON, err = json.Marshal(message.Result)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
	}
	if message.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(message.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	_, err := r.pool.Exec(ctx, query,
		message.ID,
		message.WorkspaceID,
		message.UserID,
		message.SessionID,
		message.Role,
		message.Content,
		message.SQL,
		resultJSON,   // Pass JSON bytes
		metadataJSON, // Pass JSON bytes
		message.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}

// ListBySession retrieves messages for a specific session
func (r *MessageRepository) ListBySession(ctx context.Context, sessionID uuid.UUID, limit int) ([]domain.Message, error) {
	query := `
		SELECT id, workspace_id, user_id, session_id, role, content, sql, result, metadata, created_at
		FROM chat_messages
		WHERE session_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		var m domain.Message
		var roleStr string

		if err := rows.Scan(
			&m.ID,
			&m.WorkspaceID,
			&m.UserID,
			&m.SessionID,
			&roleStr,
			&m.Content,
			&m.SQL,
			&m.Result,
			&m.Metadata,
			&m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		m.Role = domain.MessageRole(roleStr)
		messages = append(messages, m)
	}

	// Reverse to return chronological order (oldest first)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// ListByWorkspace retrieves recent messages for a workspace (Deprecated or use for overview?)
// Keeping it but maybe modifying to only show latest messages globally or just use session list.
func (r *MessageRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit int) ([]domain.Message, error) {
	// ... existing implementation but adding session_id scan ...
	query := `
		SELECT id, workspace_id, user_id, session_id, role, content, sql, result, metadata, created_at
		FROM chat_messages
		WHERE workspace_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	// ... rest of implementation similar to ListBySession but with workspace_id filter

	rows, err := r.pool.Query(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		var m domain.Message
		var roleStr string

		if err := rows.Scan(
			&m.ID,
			&m.WorkspaceID,
			&m.UserID,
			&m.SessionID,
			&roleStr,
			&m.Content,
			&m.SQL,
			&m.Result,
			&m.Metadata,
			&m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		m.Role = domain.MessageRole(roleStr)
		messages = append(messages, m)
	}

	// Reverse to return chronological order (oldest first)
	// because we ordered by DESC to get the *latest* N messages
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetMostFrequentQuestions retrieves the most frequent user questions for a workspace
func (r *MessageRepository) GetMostFrequentQuestions(ctx context.Context, workspaceID uuid.UUID, limit int) ([]string, error) {
	query := `
		SELECT content
		FROM chat_messages
		WHERE workspace_id = $1 AND role = 'user'
		GROUP BY content
		ORDER BY COUNT(*) DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query frequent questions: %w", err)
	}
	defer rows.Close()

	var questions []string
	for rows.Next() {
		var q string
		if err := rows.Scan(&q); err != nil {
			return nil, fmt.Errorf("failed to scan question: %w", err)
		}
		questions = append(questions, q)
	}

	return questions, nil
}
