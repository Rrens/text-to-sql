package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Rrens/text-to-sql/internal/api/middleware"
	"github.com/Rrens/text-to-sql/internal/api/response"
	"github.com/Rrens/text-to-sql/internal/domain"
	"github.com/Rrens/text-to-sql/internal/service"
	"github.com/go-playground/validator/v10"
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
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			errors := make(map[string]string)
			for _, e := range validationErrors {
				field := e.Field()
				tag := e.Tag()
				switch tag {
				case "required":
					errors[field] = "field is required"
				case "email":
					errors[field] = "invalid email format"
				case "min":
					errors[field] = "must be at least " + e.Param() + " characters"
				case "max":
					errors[field] = "must be at most " + e.Param() + " characters"
				default:
					errors[field] = "validation failed on " + tag
				}
			}
			response.BadRequest(w, errors)
			return
		}
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

// Me returns the current authenticated user
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Unauthorized(w, "unauthorized")
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), userID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	if user == nil {
		response.Unauthorized(w, "user not found")
		return
	}

	response.OK(w, map[string]any{
		"id":         user.ID,
		"email":      user.Email,
		"llm_config": user.LLMConfig,
	})
}

// UpdateLLMConfig updates user's LLM credentials
func (h *AuthHandler) UpdateLLMConfig(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Unauthorized(w, "unauthorized")
		return
	}

	var config map[string]any
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	user, err := h.authService.UpdateLLMConfig(r.Context(), userID, config)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.OK(w, user)
}
