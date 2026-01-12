package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/rensmac/text-to-sql/internal/api/response"
	"github.com/rensmac/text-to-sql/internal/domain"
	"github.com/rensmac/text-to-sql/internal/service"
)

var validate = validator.New()

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input domain.UserCreate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := validate.Struct(input); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	user, err := h.authService.Register(r.Context(), input)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	response.Created(w, map[string]any{
		"id":    user.ID,
		"email": user.Email,
	})
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input domain.UserLogin
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := validate.Struct(input); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	tokens, err := h.authService.Login(r.Context(), input)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}

	response.OK(w, tokens)
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := validate.Struct(input); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	tokens, err := h.authService.Refresh(r.Context(), input.RefreshToken)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}

	response.OK(w, tokens)
}
