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
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication operations
type AuthService struct {
	userRepo      *postgres.UserRepository
	workspaceRepo *postgres.WorkspaceRepository
	jwtManager    *security.JWTManager
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo *postgres.UserRepository,
	workspaceRepo *postgres.WorkspaceRepository,
	jwtManager *security.JWTManager,
) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		workspaceRepo: workspaceRepo,
		jwtManager:    jwtManager,
	}
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, input domain.UserCreate) (*domain.User, error) {
	// Check if email already exists
	exists, err := s.userRepo.EmailExists(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	now := time.Now()
	user := &domain.User{
		ID:           uuid.New(),
		Email:        input.Email,
		PasswordHash: string(hashedPassword),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, input domain.UserLogin) (*domain.TokenPair, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Get user's workspaces
	workspaces, err := s.workspaceRepo.ListByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspaces: %w", err)
	}

	workspaceIDs := make([]uuid.UUID, len(workspaces))
	for i, ws := range workspaces {
		workspaceIDs[i] = ws.ID
	}

	// Generate tokens
	accessToken, refreshToken, expiresIn, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, workspaceIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// Refresh refreshes the access token using a refresh token
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	// Validate refresh token
	userID, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	// Get user's workspaces
	workspaces, err := s.workspaceRepo.ListByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspaces: %w", err)
	}

	workspaceIDs := make([]uuid.UUID, len(workspaces))
	for i, ws := range workspaces {
		workspaceIDs[i] = ws.ID
	}

	// Generate new tokens
	accessToken, newRefreshToken, expiresIn, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, workspaceIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// GetUserByID retrieves a user by ID
func (s *AuthService) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}
