package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rensmac/text-to-sql/internal/api/middleware"
	"github.com/rensmac/text-to-sql/internal/api/response"
	"github.com/rensmac/text-to-sql/internal/domain"
	"github.com/rensmac/text-to-sql/internal/service"
)

// QueryHandler handles query endpoints
type QueryHandler struct {
	queryService *service.QueryService
}

// NewQueryHandler creates a new query handler
func NewQueryHandler(queryService *service.QueryService) *QueryHandler {
	return &QueryHandler{queryService: queryService}
}

// Execute handles text-to-SQL query execution
func (h *QueryHandler) Execute(w http.ResponseWriter, r *http.Request) {
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

	var req domain.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := validate.Struct(req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	result, err := h.queryService.ExecuteQuery(r.Context(), userID, workspaceID, req)
	if err != nil {
		if err.Error() == "access denied" {
			response.Forbidden(w, err.Error())
			return
		}
		response.InternalError(w, err.Error())
		return
	}

	response.OK(w, result)
}

// Generate handles SQL generation without execution
func (h *QueryHandler) Generate(w http.ResponseWriter, r *http.Request) {
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

	var req domain.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	// Force execute to false for generate-only
	req.Execute = false

	if err := validate.Struct(req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	result, err := h.queryService.ExecuteQuery(r.Context(), userID, workspaceID, req)
	if err != nil {
		if err.Error() == "access denied" {
			response.Forbidden(w, err.Error())
			return
		}
		response.InternalError(w, err.Error())
		return
	}

	response.OK(w, result)
}

// GetSchema returns the schema for a connection
func (h *QueryHandler) GetSchema(w http.ResponseWriter, r *http.Request) {
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

	schema, err := h.queryService.GetSchema(r.Context(), userID, workspaceID, connectionID)
	if err != nil {
		if err.Error() == "access denied" {
			response.Forbidden(w, err.Error())
			return
		}
		response.InternalError(w, err.Error())
		return
	}

	response.OK(w, schema)
}

// RefreshSchema forces a schema refresh for a connection
func (h *QueryHandler) RefreshSchema(w http.ResponseWriter, r *http.Request) {
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

	schema, err := h.queryService.RefreshSchema(r.Context(), userID, workspaceID, connectionID)
	if err != nil {
		if err.Error() == "access denied" {
			response.Forbidden(w, err.Error())
			return
		}
		response.InternalError(w, err.Error())
		return
	}

	response.OK(w, schema)
}
