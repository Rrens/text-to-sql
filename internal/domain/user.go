package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents a platform user
type User struct {
	ID           uuid.UUID      `json:"id"`
	Email        string         `json:"email"`
	PasswordHash string         `json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	LLMConfig    map[string]any `json:"llm_config"`
}

// UserCreate represents user registration data
type UserCreate struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// UserLogin represents login credentials
type UserLogin struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// TokenPair represents JWT token pair
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}
