package postgres

import (
	"context"
	"fmt"

	"github.com/Rrens/text-to-sql/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionRepository implements domain.SessionRepository
type SessionRepository struct {
	pool *pgxpool.Pool
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

func (r *SessionRepository) Create(ctx context.Context, session *domain.ChatSession) error {
	query := `
		INSERT INTO chat_sessions (id, workspace_id, user_id, title, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		session.ID,
		session.WorkspaceID,
		session.UserID,
		session.Title,
		session.CreatedAt,
		session.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

func (r *SessionRepository) Get(ctx context.Context, id uuid.UUID) (*domain.ChatSession, error) {
	query := `
		SELECT id, workspace_id, user_id, title, created_at, updated_at
		FROM chat_sessions
		WHERE id = $1
	`
	var s domain.ChatSession
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID,
		&s.WorkspaceID,
		&s.UserID,
		&s.Title,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &s, nil
}

func (r *SessionRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit int, offset int) ([]domain.ChatSession, error) {
	query := `
		SELECT id, workspace_id, user_id, title, created_at, updated_at
		FROM chat_sessions
		WHERE workspace_id = $1
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, workspaceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []domain.ChatSession
	for rows.Next() {
		var s domain.ChatSession
		if err := rows.Scan(
			&s.ID,
			&s.WorkspaceID,
			&s.UserID,
			&s.Title,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *SessionRepository) Update(ctx context.Context, session *domain.ChatSession) error {
	query := `
		UPDATE chat_sessions
		SET title = $1, updated_at = $2
		WHERE id = $3
	`
	_, err := r.pool.Exec(ctx, query, session.Title, session.UpdatedAt, session.ID)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

func (r *SessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM chat_sessions WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}
