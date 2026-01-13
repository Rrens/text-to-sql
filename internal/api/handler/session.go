package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Rrens/text-to-sql/internal/api/middleware"
	"github.com/Rrens/text-to-sql/internal/api/response"
	"github.com/Rrens/text-to-sql/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type SessionHandler struct {
	queryService *service.QueryService
}

func NewSessionHandler(queryService *service.QueryService) *SessionHandler {
	return &SessionHandler{queryService: queryService}
}

// List returns all sessions for a workspace
func (h *SessionHandler) List(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := middleware.GetWorkspaceID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Missing workspace ID")
		return
	}

	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	sessions, err := h.queryService.ListSessions(r.Context(), workspaceID, limit, offset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list sessions")
		return
	}

	response.JSON(w, http.StatusOK, sessions)
}

// Create creates a new session
func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := middleware.GetWorkspaceID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Missing workspace ID")
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "User ID not found")
		return
	}

	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Optional body
	}

	session, err := h.queryService.CreateSession(r.Context(), userID, workspaceID, req.Title)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	response.JSON(w, http.StatusCreated, session)
}

// GetHistory returns history for a specific session
func (h *SessionHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := chi.URLParam(r, "sessionID")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	history, err := h.queryService.GetSessionHistory(r.Context(), sessionID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch session history")
		return
	}

	response.JSON(w, http.StatusOK, history)
}

// Delete deletes a session
func (h *SessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := chi.URLParam(r, "sessionID")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	if err := h.queryService.DeleteSession(r.Context(), sessionID); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete session")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Session deleted"})
}
