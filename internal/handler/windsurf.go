package handler

import (
	"net/http"

	"weathertracker/internal/domain"
	"weathertracker/internal/middleware"
	"weathertracker/internal/service"
)

// WindsurfHandler exposes the rider profile and recommendation endpoints
// (User Story 2).
type WindsurfHandler struct {
	windsurf *service.WindsurfService
}

// NewWindsurfHandler builds the windsurf handler.
func NewWindsurfHandler(windsurf *service.WindsurfService) *WindsurfHandler {
	return &WindsurfHandler{windsurf: windsurf}
}

// GetProfile godoc
//
//	@Summary      Get the windsurf profile
//	@Description  Returns the rider's level and target wind range.
//	@Tags         windsurf
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200 {object} domain.WindsurfProfile
//	@Failure      401 {object} errorResponse
//	@Failure      404 {object} errorResponse
//	@Failure      500 {object} errorResponse
//	@Router       /api/v1/users/me/windsurf-profile [get]
func (h *WindsurfHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var profile domain.WindsurfProfile
	profile, err := h.windsurf.Profile(r.Context(), userID)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

// SaveProfile godoc
//
//	@Summary      Create or update the windsurf profile
//	@Description  Stores the rider's level and acceptable wind speed range.
//	@Tags         windsurf
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        request body service.ProfileInput true "Rider profile"
//	@Success      200 {object} domain.WindsurfProfile
//	@Failure      400 {object} errorResponse
//	@Failure      401 {object} errorResponse
//	@Failure      500 {object} errorResponse
//	@Router       /api/v1/users/me/windsurf-profile [put]
func (h *WindsurfHandler) SaveProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req service.ProfileInput
	if err := decodeJSON(w, r, &req); err != nil {
		handleError(w, err)
		return
	}

	var profile domain.WindsurfProfile
	profile, err := h.windsurf.SaveProfile(r.Context(), userID, req)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

// Recommend godoc
//
//	@Summary      Recommend a windsurfing spot and time slot
//	@Description  Analyses the forecast across several spots, accounts for wind direction relative to the shore (onshore/offshore/cross-shore) and the rider's level, then returns ranked recommendations with reasons. Falls back to saved locations when no spots are supplied.
//	@Tags         windsurf
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        request body service.RecommendInput true "Recommendation parameters"
//	@Success      200 {object} domain.RecommendationResult
//	@Failure      400 {object} errorResponse
//	@Failure      401 {object} errorResponse
//	@Failure      500 {object} errorResponse
//	@Router       /api/v1/windsurf/recommendations [post]
func (h *WindsurfHandler) Recommend(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req service.RecommendInput
	if err := decodeJSON(w, r, &req); err != nil {
		handleError(w, err)
		return
	}

	var result domain.RecommendationResult
	result, err := h.windsurf.Recommend(r.Context(), userID, req)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
