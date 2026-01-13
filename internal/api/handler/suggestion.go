package handler

import (
	"net/http"

	"github.com/Rrens/text-to-sql/internal/api/middleware"
	"github.com/Rrens/text-to-sql/internal/api/response"
	"github.com/Rrens/text-to-sql/internal/service"
)

type SuggestionHandler struct {
	queryService *service.QueryService
}

func NewSuggestionHandler(queryService *service.QueryService) *SuggestionHandler {
	return &SuggestionHandler{queryService: queryService}
}

// GetSuggestions returns suggested questions for the workspace
func (h *SuggestionHandler) GetSuggestions(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := middleware.GetWorkspaceID(r.Context())
	if !ok {
		response.BadRequest(w, "missing workspace ID")
		return
	}

	suggestions, err := h.queryService.GetSuggestedQuestions(r.Context(), workspaceID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.OK(w, suggestions)
}
