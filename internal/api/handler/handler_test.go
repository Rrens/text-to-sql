package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Rrens/text-to-sql/internal/api/handler"
	"github.com/Rrens/text-to-sql/internal/repository/postgres"
	"github.com/Rrens/text-to-sql/internal/security"
	"github.com/Rrens/text-to-sql/internal/service"
)

// MockDB implements a minimal mock for testing
type MockDB struct{}

func TestAuthHandler_Register(t *testing.T) {
	// This is a simplified test - in real scenario you'd use a test database or mocks
	t.Skip("Requires database connection - run as integration test")
}

func TestAuthHandler_Login(t *testing.T) {
	t.Skip("Requires database connection - run as integration test")
}

func TestHealthCheck(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["success"] != true {
		t.Error("expected success to be true")
	}

	data, ok := response["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data to be a map")
	}

	if data["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", data["status"])
	}
}

// TestAuthFlow tests the complete authentication flow
func TestAuthFlow(t *testing.T) {
	t.Skip("Requires database connection - run as integration test")

	// This would be the integration test flow:
	// 1. Register a new user
	// 2. Login with credentials
	// 3. Use access token to access protected routes
	// 4. Refresh the token
	// 5. Verify the new token works
}

// BenchmarkJWTGeneration benchmarks token generation
func BenchmarkJWTGeneration(b *testing.B) {
	manager := security.NewJWTManager("benchmark-secret-key-32-chars!!", 15*time.Minute, 7*24*time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GenerateAccessToken(
			[16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			"test@example.com",
			nil,
		)
	}
}

// Helper function to create test auth service
func newTestAuthService(db *postgres.DB, jwtManager *security.JWTManager) *service.AuthService {
	userRepo := postgres.NewUserRepository(db)
	workspaceRepo := postgres.NewWorkspaceRepository(db)
	return service.NewAuthService(userRepo, workspaceRepo, jwtManager)
}

// Helper to make JSON request
func makeJSONRequest(method, path string, body any) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	return req
}
