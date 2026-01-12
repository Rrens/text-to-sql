package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Rrens/text-to-sql/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// WorkspaceRepository handles workspace data access
type WorkspaceRepository struct {
	db *DB
}

// NewWorkspaceRepository creates a new workspace repository
func NewWorkspaceRepository(db *DB) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

// Create creates a new workspace
func (r *WorkspaceRepository) Create(ctx context.Context, workspace *domain.Workspace) error {
	settings, err := json.Marshal(workspace.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	query := `
		INSERT INTO workspaces (id, name, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = r.db.Pool.Exec(ctx, query,
		workspace.ID,
		workspace.Name,
		settings,
		workspace.CreatedAt,
		workspace.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	return nil
}

// GetByID retrieves a workspace by ID
func (r *WorkspaceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Workspace, error) {
	query := `
		SELECT id, name, settings, created_at, updated_at
		FROM workspaces
		WHERE id = $1
	`

	var workspace domain.Workspace
	var settingsJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&workspace.ID,
		&workspace.Name,
		&settingsJSON,
		&workspace.CreatedAt,
		&workspace.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	if len(settingsJSON) > 0 {
		if err := json.Unmarshal(settingsJSON, &workspace.Settings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
		}
	}

	return &workspace, nil
}

// ListByUserID retrieves all workspaces for a user
func (r *WorkspaceRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	query := `
		SELECT w.id, w.name, w.settings, w.created_at, w.updated_at
		FROM workspaces w
		INNER JOIN workspace_members wm ON w.id = wm.workspace_id
		WHERE wm.user_id = $1
		ORDER BY w.created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}
	defer rows.Close()

	var workspaces []domain.Workspace
	for rows.Next() {
		var workspace domain.Workspace
		var settingsJSON []byte

		if err := rows.Scan(
			&workspace.ID,
			&workspace.Name,
			&settingsJSON,
			&workspace.CreatedAt,
			&workspace.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan workspace: %w", err)
		}

		if len(settingsJSON) > 0 {
			json.Unmarshal(settingsJSON, &workspace.Settings)
		}

		workspaces = append(workspaces, workspace)
	}

	return workspaces, nil
}

// Update updates a workspace
func (r *WorkspaceRepository) Update(ctx context.Context, id uuid.UUID, update *domain.WorkspaceUpdate) error {
	settings, err := json.Marshal(update.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	query := `
		UPDATE workspaces
		SET name = COALESCE($2, name),
		    settings = COALESCE($3, settings),
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err = r.db.Pool.Exec(ctx, query, id, update.Name, settings)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	return nil
}

// Delete deletes a workspace
func (r *WorkspaceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM workspaces WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	return nil
}

// AddMember adds a member to a workspace
func (r *WorkspaceRepository) AddMember(ctx context.Context, member *domain.WorkspaceMember) error {
	query := `
		INSERT INTO workspace_members (workspace_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (workspace_id, user_id) DO UPDATE SET role = $3
	`

	_, err := r.db.Pool.Exec(ctx, query,
		member.WorkspaceID,
		member.UserID,
		member.Role,
		member.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	return nil
}

// GetMember retrieves a workspace member
func (r *WorkspaceRepository) GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*domain.WorkspaceMember, error) {
	query := `
		SELECT workspace_id, user_id, role, created_at
		FROM workspace_members
		WHERE workspace_id = $1 AND user_id = $2
	`

	var member domain.WorkspaceMember
	err := r.db.Pool.QueryRow(ctx, query, workspaceID, userID).Scan(
		&member.WorkspaceID,
		&member.UserID,
		&member.Role,
		&member.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	return &member, nil
}

// IsMember checks if a user is a member of a workspace
func (r *WorkspaceRepository) IsMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM workspace_members 
			WHERE workspace_id = $1 AND user_id = $2
		)
	`

	var exists bool
	err := r.db.Pool.QueryRow(ctx, query, workspaceID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return exists, nil
}

// RemoveMember removes a member from a workspace
func (r *WorkspaceRepository) RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error {
	query := `DELETE FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`

	_, err := r.db.Pool.Exec(ctx, query, workspaceID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	return nil
}
