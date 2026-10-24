package handler

import (
	"net/http"

	"weathertracker/internal/domain"
	"weathertracker/internal/middleware"
	"weathertracker/internal/service"
)

// RankingHandler exposes the city-ranking endpoint (User Story 1).
type RankingHandler struct {
	ranking *service.RankingService
}

// NewRankingHandler builds the ranking handler.
func NewRankingHandler(ranking *service.RankingService) *RankingHandler {
	return &RankingHandler{ranking: ranking}
}

// Rank godoc
//
//	@Summary      Rank cities by weather priorities
//	@Description  Scores up to 5 cities against the traveller's priorities, returns them in descending order and highlights the leader. Cities with data older than 5 minutes are flagged with isStale=true.
//	@Tags         ranking
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        request body service.RankingInput true "Cities and ranking criteria"
//	@Success      200 {object} domain.RankingResult
//	@Failure      400 {object} errorResponse
//	@Failure      401 {object} errorResponse
//	@Failure      422 {object} errorResponse
//	@Failure      500 {object} errorResponse
//	@Router       /api/v1/weather/rank [post]
func (h *RankingHandler) Rank(w http.ResponseWriter, r *http.Request) {
	if _, ok := middleware.UserID(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req service.RankingInput
	if err := decodeJSON(w, r, &req); err != nil {
		handleError(w, err)
		return
	}

	var result domain.RankingResult
	result, err := h.ranking.Rank(r.Context(), req)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
