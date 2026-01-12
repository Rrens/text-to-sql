package handler

import (
	"encoding/json"
	"net/http"

	"github.com/rensmac/text-to-sql/internal/api/middleware"
	"github.com/rensmac/text-to-sql/internal/api/response"
	"github.com/rensmac/text-to-sql/internal/domain"
	"github.com/rensmac/text-to-sql/internal/service"
)

// WorkspaceHandler handles workspace endpoints
type WorkspaceHandler struct {
	workspaceService *service.WorkspaceService
}

// NewWorkspaceHandler creates a new workspace handler
func NewWorkspaceHandler(workspaceService *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{workspaceService: workspaceService}
}

// Create handles workspace creation
func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Unauthorized(w, "unauthorized")
		return
	}

	var input domain.WorkspaceCreate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := validate.Struct(input); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	workspace, err := h.workspaceService.Create(r.Context(), userID, input)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Created(w, workspace)
}

// List handles listing user's workspaces
func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Unauthorized(w, "unauthorized")
		return
	}

	workspaces, err := h.workspaceService.ListByUser(r.Context(), userID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.OK(w, workspaces)
}

// Get handles getting a workspace by ID
func (h *WorkspaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Unauthorized(w, "unauthorized")
		return
	}

	workspaceID, ok := middleware.GetWorkspaceID(r.Context())
	if !ok {
		response.BadRequest(w, "missing workspace ID")
		return
	}

	workspace, err := h.workspaceService.GetByID(r.Context(), userID, workspaceID)
	if err != nil {
		if err.Error() == "access denied" {
			response.Forbidden(w, err.Error())
			return
		}
		if err.Error() == "workspace not found" {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, err.Error())
		return
	}

	response.OK(w, workspace)
}

// Update handles updating a workspace
func (h *WorkspaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Unauthorized(w, "unauthorized")
		return
	}

	workspaceID, ok := middleware.GetWorkspaceID(r.Context())
	if !ok {
		response.BadRequest(w, "missing workspace ID")
		return
	}

	var input domain.WorkspaceUpdate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	workspace, err := h.workspaceService.Update(r.Context(), userID, workspaceID, input)
	if err != nil {
		if err.Error() == "access denied" || err.Error() == "admin access required" {
			response.Forbidden(w, err.Error())
			return
		}
		response.InternalError(w, err.Error())
		return
	}

	response.OK(w, workspace)
}

// Delete handles deleting a workspace
func (h *WorkspaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Unauthorized(w, "unauthorized")
		return
	}

	workspaceID, ok := middleware.GetWorkspaceID(r.Context())
	if !ok {
		response.BadRequest(w, "missing workspace ID")
		return
	}

	err := h.workspaceService.Delete(r.Context(), userID, workspaceID)
	if err != nil {
		if err.Error() == "access denied" || err.Error() == "owner access required" {
			response.Forbidden(w, err.Error())
			return
		}
		response.InternalError(w, err.Error())
		return
	}

	response.NoContent(w)
}
