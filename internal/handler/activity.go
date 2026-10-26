package handler

import (
	"net/http"

	"weathertracker/internal/domain"
	"weathertracker/internal/middleware"
	"weathertracker/internal/service"
)

type ActivityHandler struct {
	activity *service.ActivityService
}

func NewActivityHandler(activity *service.ActivityService) *ActivityHandler {
	return &ActivityHandler{activity: activity}
}

// Recommend godoc
//
//	@Summary      Recommend optimal time windows for outdoor activity
//	@Description  Analyses hourly forecast for a city and date, recommending the best time windows for outdoor workout or walking.
//	@Tags         activity
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        request body service.ActivityInput true "Activity recommendation criteria"
//	@Success      200 {object} domain.ActivityRecommendationResult
//	@Failure      400 {object} errorResponse
//	@Failure      401 {object} errorResponse
//	@Failure      422 {object} errorResponse
//	@Failure      500 {object} errorResponse
//	@Router       /api/v1/activity/recommendations [post]
func (h *ActivityHandler) Recommend(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req service.ActivityInput
	if err := decodeJSON(w, r, &req); err != nil {
		handleError(w, err)
		return
	}

	var result domain.ActivityRecommendationResult
	result, err := h.activity.Recommend(r.Context(), userID, req)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
