package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rensmac/text-to-sql/internal/domain"
	"github.com/rensmac/text-to-sql/internal/repository/postgres"
)

// WorkspaceService handles workspace operations
type WorkspaceService struct {
	workspaceRepo *postgres.WorkspaceRepository
}

// NewWorkspaceService creates a new workspace service
func NewWorkspaceService(workspaceRepo *postgres.WorkspaceRepository) *WorkspaceService {
	return &WorkspaceService{workspaceRepo: workspaceRepo}
}

// Create creates a new workspace and adds the creator as owner
func (s *WorkspaceService) Create(ctx context.Context, userID uuid.UUID, input domain.WorkspaceCreate) (*domain.Workspace, error) {
	now := time.Now()
	workspace := &domain.Workspace{
		ID:        uuid.New(),
		Name:      input.Name,
		Settings:  input.Settings,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create workspace
	if err := s.workspaceRepo.Create(ctx, workspace); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Add creator as owner
	member := &domain.WorkspaceMember{
		WorkspaceID: workspace.ID,
		UserID:      userID,
		Role:        domain.RoleOwner,
		CreatedAt:   now,
	}

	if err := s.workspaceRepo.AddMember(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	return workspace, nil
}

// GetByID retrieves a workspace by ID with access check
func (s *WorkspaceService) GetByID(ctx context.Context, userID, workspaceID uuid.UUID) (*domain.Workspace, error) {
	// Check membership
	isMember, err := s.workspaceRepo.IsMember(ctx, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return nil, errors.New("access denied")
	}

	workspace, err := s.workspaceRepo.GetByID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}
	if workspace == nil {
		return nil, errors.New("workspace not found")
	}

	return workspace, nil
}

// ListByUser retrieves all workspaces for a user
func (s *WorkspaceService) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	workspaces, err := s.workspaceRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}
	return workspaces, nil
}

// Update updates a workspace
func (s *WorkspaceService) Update(ctx context.Context, userID, workspaceID uuid.UUID, input domain.WorkspaceUpdate) (*domain.Workspace, error) {
	// Check if user is admin or owner
	member, err := s.workspaceRepo.GetMember(ctx, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}
	if member == nil {
		return nil, errors.New("access denied")
	}
	if member.Role != domain.RoleOwner && member.Role != domain.RoleAdmin {
		return nil, errors.New("admin access required")
	}

	// Update workspace
	if err := s.workspaceRepo.Update(ctx, workspaceID, &input); err != nil {
		return nil, fmt.Errorf("failed to update workspace: %w", err)
	}

	return s.workspaceRepo.GetByID(ctx, workspaceID)
}

// Delete deletes a workspace (owner only)
func (s *WorkspaceService) Delete(ctx context.Context, userID, workspaceID uuid.UUID) error {
	// Check if user is owner
	member, err := s.workspaceRepo.GetMember(ctx, workspaceID, userID)
	if err != nil {
		return fmt.Errorf("failed to get member: %w", err)
	}
	if member == nil {
		return errors.New("access denied")
	}
	if member.Role != domain.RoleOwner {
		return errors.New("owner access required")
	}

	return s.workspaceRepo.Delete(ctx, workspaceID)
}

// AddMember adds a member to a workspace
func (s *WorkspaceService) AddMember(ctx context.Context, requesterID, workspaceID, userID uuid.UUID, role string) error {
	// Check if requester is admin or owner
	member, err := s.workspaceRepo.GetMember(ctx, workspaceID, requesterID)
	if err != nil {
		return fmt.Errorf("failed to get member: %w", err)
	}
	if member == nil {
		return errors.New("access denied")
	}
	if member.Role != domain.RoleOwner && member.Role != domain.RoleAdmin {
		return errors.New("admin access required")
	}

	// Validate role
	if role != domain.RoleMember && role != domain.RoleAdmin {
		return errors.New("invalid role")
	}

	newMember := &domain.WorkspaceMember{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        role,
		CreatedAt:   time.Now(),
	}

	return s.workspaceRepo.AddMember(ctx, newMember)
}

// RemoveMember removes a member from a workspace
func (s *WorkspaceService) RemoveMember(ctx context.Context, requesterID, workspaceID, userID uuid.UUID) error {
	// Check if requester is admin or owner
	member, err := s.workspaceRepo.GetMember(ctx, workspaceID, requesterID)
	if err != nil {
		return fmt.Errorf("failed to get member: %w", err)
	}
	if member == nil {
		return errors.New("access denied")
	}
	if member.Role != domain.RoleOwner && member.Role != domain.RoleAdmin {
		return errors.New("admin access required")
	}

	// Cannot remove owner
	targetMember, err := s.workspaceRepo.GetMember(ctx, workspaceID, userID)
	if err != nil {
		return fmt.Errorf("failed to get target member: %w", err)
	}
	if targetMember != nil && targetMember.Role == domain.RoleOwner {
		return errors.New("cannot remove owner")
	}

	return s.workspaceRepo.RemoveMember(ctx, workspaceID, userID)
}

// IsMember checks if a user is a member of a workspace
func (s *WorkspaceService) IsMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	return s.workspaceRepo.IsMember(ctx, workspaceID, userID)
}
