package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Rrens/text-to-sql/internal/api/response"
	"github.com/Rrens/text-to-sql/internal/repository/redis"
	"github.com/Rrens/text-to-sql/internal/security"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type contextKey string

const (
	UserIDKey      contextKey = "userID"
	UserEmailKey   contextKey = "userEmail"
	WorkspaceIDKey contextKey = "workspaceID"
)

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	jwtManager *security.JWTManager
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(jwtManager *security.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{jwtManager: jwtManager}
}

// Authenticate validates the JWT token
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Error(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Error(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		claims, err := m.jwtManager.ValidateAccessToken(parts[1])
		if err != nil {
			response.Unauthorized(w, "invalid or expired token: "+err.Error())
			return
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UserEmailKey, claims.Email)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID gets the user ID from context
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
	return userID, ok
}

// GetUserEmail gets the user email from context
func GetUserEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(UserEmailKey).(string)
	return email, ok
}

// GetWorkspaceID gets the workspace ID from context
func GetWorkspaceID(ctx context.Context) (uuid.UUID, bool) {
	workspaceID, ok := ctx.Value(WorkspaceIDKey).(uuid.UUID)
	return workspaceID, ok
}

// WorkspaceContext extracts workspace ID from URL and adds to context
func WorkspaceContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		workspaceIDStr := chi.URLParam(r, "workspaceID")
		if workspaceIDStr == "" {
			response.Error(w, http.StatusBadRequest, "missing workspace ID")
			return
		}

		workspaceID, err := uuid.Parse(workspaceIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid workspace ID")
			return
		}

		ctx := context.WithValue(r.Context(), WorkspaceIDKey, workspaceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RateLimitMiddleware handles rate limiting
type RateLimitMiddleware struct {
	rateLimiter *redis.RateLimiter
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(rateLimiter *redis.RateLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{rateLimiter: rateLimiter}
}

// Limit applies rate limiting based on user ID
func (m *RateLimitMiddleware) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		allowed, remaining, resetTime, err := m.rateLimiter.Allow(r.Context(), userID.String())
		if err != nil {
			// If rate limiter fails, allow the request but log the error
			next.ServeHTTP(w, r)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Remaining", string(rune(remaining)))
		w.Header().Set("X-RateLimit-Reset", resetTime.Format("2006-01-02T15:04:05Z"))

		if !allowed {
			response.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}
