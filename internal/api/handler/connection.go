package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Rrens/text-to-sql/internal/api/middleware"
	"github.com/Rrens/text-to-sql/internal/api/response"
	"github.com/Rrens/text-to-sql/internal/domain"
	"github.com/Rrens/text-to-sql/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ConnectionHandler handles database connection endpoints
type ConnectionHandler struct {
	connectionService *service.ConnectionService
}

// NewConnectionHandler creates a new connection handler
func NewConnectionHandler(connectionService *service.ConnectionService) *ConnectionHandler {
	return &ConnectionHandler{connectionService: connectionService}
}

// Create handles connection creation
func (h *ConnectionHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	var input domain.ConnectionCreate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := validate.Struct(input); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	conn, err := h.connectionService.Create(r.Context(), userID, workspaceID, input)
	if err != nil {
		if err.Error() == "access denied" {
			response.Forbidden(w, err.Error())
			return
		}
		response.InternalError(w, err.Error())
		return
	}

	response.Created(w, conn)
}

// List handles listing connections in a workspace
func (h *ConnectionHandler) List(w http.ResponseWriter, r *http.Request) {
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

	connections, err := h.connectionService.ListByWorkspace(r.Context(), userID, workspaceID)
	if err != nil {
		if err.Error() == "access denied" {
			response.Forbidden(w, err.Error())
			return
		}
		response.InternalError(w, err.Error())
		return
	}

	response.OK(w, connections)
}

// Get handles getting a connection by ID
func (h *ConnectionHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	connectionIDStr := chi.URLParam(r, "connectionID")
	connectionID, err := uuid.Parse(connectionIDStr)
	if err != nil {
		response.BadRequest(w, "invalid connection ID")
		return
	}

	conn, err := h.connectionService.GetByID(r.Context(), userID, workspaceID, connectionID)
	if err != nil {
		if err.Error() == "access denied" {
			response.Forbidden(w, err.Error())
			return
		}
		if err.Error() == "connection not found" {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, err.Error())
		return
	}

	response.OK(w, conn)
}

// Update handles updating a connection
func (h *ConnectionHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	connectionIDStr := chi.URLParam(r, "connectionID")
	connectionID, err := uuid.Parse(connectionIDStr)
	if err != nil {
		response.BadRequest(w, "invalid connection ID")
		return
	}

	var input domain.ConnectionUpdate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	conn, err := h.connectionService.Update(r.Context(), userID, workspaceID, connectionID, input)
	if err != nil {
		if err.Error() == "access denied" {
			response.Forbidden(w, err.Error())
			return
		}
		if err.Error() == "connection not found" {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, err.Error())
		return
	}

	response.OK(w, conn)
}

// Delete handles deleting a connection
func (h *ConnectionHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	connectionIDStr := chi.URLParam(r, "connectionID")
	connectionID, err := uuid.Parse(connectionIDStr)
	if err != nil {
		response.BadRequest(w, "invalid connection ID")
		return
	}

	err = h.connectionService.Delete(r.Context(), userID, workspaceID, connectionID)
	if err != nil {
		if err.Error() == "access denied" {
			response.Forbidden(w, err.Error())
			return
		}
		if err.Error() == "connection not found" {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, err.Error())
		return
	}

	response.NoContent(w)
}

// Test handles testing a connection
func (h *ConnectionHandler) Test(w http.ResponseWriter, r *http.Request) {
	var input domain.ConnectionCreate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := validate.Struct(input); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	err := h.connectionService.TestConnection(r.Context(), input)
	if err != nil {
		response.BadRequest(w, map[string]any{
			"connected": false,
			"error":     err.Error(),
		})
		return
	}

	response.OK(w, map[string]any{
		"connected": true,
		"message":   "Connection successful",
	})
}
