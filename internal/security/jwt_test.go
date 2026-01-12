package security_test

import (
	"testing"
	"time"

	"github.com/Rrens/text-to-sql/internal/security"
	"github.com/google/uuid"
)

func TestJWTManager_GenerateAndValidate(t *testing.T) {
	manager := security.NewJWTManager("test-secret-key-with-32-chars!!", 15*time.Minute, 7*24*time.Hour)

	userID := uuid.New()
	email := "test@example.com"
	workspaces := []uuid.UUID{uuid.New(), uuid.New()}

	// Generate access token
	accessToken, err := manager.GenerateAccessToken(userID, email, workspaces)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	if accessToken == "" {
		t.Error("access token is empty")
	}

	// Validate access token
	claims, err := manager.ValidateAccessToken(accessToken)
	if err != nil {
		t.Fatalf("failed to validate access token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("user ID mismatch: got %v, want %v", claims.UserID, userID)
	}

	if claims.Email != email {
		t.Errorf("email mismatch: got %v, want %v", claims.Email, email)
	}

	if len(claims.Workspaces) != len(workspaces) {
		t.Errorf("workspaces count mismatch: got %d, want %d", len(claims.Workspaces), len(workspaces))
	}
}

func TestJWTManager_GenerateTokenPair(t *testing.T) {
	manager := security.NewJWTManager("test-secret-key-with-32-chars!!", 15*time.Minute, 7*24*time.Hour)

	userID := uuid.New()
	email := "test@example.com"

	accessToken, refreshToken, expiresIn, err := manager.GenerateTokenPair(userID, email, nil)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	if accessToken == "" {
		t.Error("access token is empty")
	}

	if refreshToken == "" {
		t.Error("refresh token is empty")
	}

	if expiresIn != int64((15 * time.Minute).Seconds()) {
		t.Errorf("expires in mismatch: got %d, want %d", expiresIn, int64((15 * time.Minute).Seconds()))
	}

	// Validate refresh token
	extractedUserID, err := manager.ValidateRefreshToken(refreshToken)
	if err != nil {
		t.Fatalf("failed to validate refresh token: %v", err)
	}

	if extractedUserID != userID {
		t.Errorf("user ID from refresh token mismatch: got %v, want %v", extractedUserID, userID)
	}
}

func TestJWTManager_InvalidToken(t *testing.T) {
	manager := security.NewJWTManager("test-secret-key-with-32-chars!!", 15*time.Minute, 7*24*time.Hour)

	// Invalid token format
	_, err := manager.ValidateAccessToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token, got nil")
	}

	// Empty token
	_, err = manager.ValidateAccessToken("")
	if err == nil {
		t.Error("expected error for empty token, got nil")
	}

	// Token signed with different secret
	otherManager := security.NewJWTManager("different-secret-key-32-chars!!", 15*time.Minute, 7*24*time.Hour)
	token, _ := otherManager.GenerateAccessToken(uuid.New(), "test@example.com", nil)

	_, err = manager.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error for token signed with different secret, got nil")
	}
}

func TestJWTManager_AccessTokenTTL(t *testing.T) {
	accessTTL := 30 * time.Minute
	manager := security.NewJWTManager("test-secret-key-with-32-chars!!", accessTTL, 7*24*time.Hour)

	if manager.AccessTokenTTL() != accessTTL {
		t.Errorf("access token TTL mismatch: got %v, want %v", manager.AccessTokenTTL(), accessTTL)
	}
}
