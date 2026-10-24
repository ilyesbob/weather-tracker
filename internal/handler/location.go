package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"weathertracker/internal/domain"
	"weathertracker/internal/middleware"
	"weathertracker/internal/service"
)

// LocationHandler exposes geocoding search and the per-user comparison list.
type LocationHandler struct {
	locations *service.LocationService
}

// NewLocationHandler builds the location handler.
func NewLocationHandler(locations *service.LocationService) *LocationHandler {
	return &LocationHandler{locations: locations}
}

type locationListResponse struct {
	Items []domain.Location `json:"items"`
}

type savedLocationListResponse struct {
	Items []domain.SavedLocation `json:"items"`
	Limit int                    `json:"limit"`
}

// Search godoc
//
//	@Summary      Search locations by name
//	@Description  Geocodes a free-text query and returns candidates with stable IDs.
//	@Tags         locations
//	@Produce      json
//	@Param        q     query    string true  "City name to search for" example(Berlin)
//	@Param        limit query    int    false "Maximum number of results (default 10)"
//	@Success      200   {object} locationListResponse
//	@Failure      400   {object} errorResponse
//	@Failure      500   {object} errorResponse
//	@Router       /api/v1/locations/search [get]
func (h *LocationHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	items, err := h.locations.Search(r.Context(), query, limit)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, locationListResponse{Items: items})
}

// ListSaved godoc
//
//	@Summary      List saved locations
//	@Description  Returns the authenticated user's comparison list (User Story 1).
//	@Tags         users
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200 {object} savedLocationListResponse
//	@Failure      401 {object} errorResponse
//	@Failure      500 {object} errorResponse
//	@Router       /api/v1/users/me/locations [get]
func (h *LocationHandler) ListSaved(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	items, err := h.locations.ListSaved(r.Context(), userID)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, savedLocationListResponse{Items: items, Limit: service.MaxSavedLocations})
}

// AddSaved godoc
//
//	@Summary      Add a location to the comparison list
//	@Description  Adds a city (max 5) to the authenticated user's comparison list.
//	@Tags         users
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        request body service.AddSavedInput true "Location to save"
//	@Success      201 {object} domain.SavedLocation
//	@Failure      400 {object} errorResponse
//	@Failure      401 {object} errorResponse
//	@Failure      409 {object} errorResponse
//	@Failure      422 {object} errorResponse
//	@Router       /api/v1/users/me/locations [post]
func (h *LocationHandler) AddSaved(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req service.AddSavedInput
	if err := decodeJSON(w, r, &req); err != nil {
		handleError(w, err)
		return
	}

	saved, err := h.locations.AddSaved(r.Context(), userID, req)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

// DeleteSaved godoc
//
//	@Summary      Remove a location from the comparison list
//	@Description  Deletes a city from the authenticated user's comparison list.
//	@Tags         users
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id path int true "Location ID"
//	@Success      204
//	@Failure      400 {object} errorResponse
//	@Failure      401 {object} errorResponse
//	@Failure      404 {object} errorResponse
//	@Failure      500 {object} errorResponse
//	@Router       /api/v1/users/me/locations/{id} [delete]
func (h *LocationHandler) DeleteSaved(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid location id")
		return
	}

	if err := h.locations.DeleteSaved(r.Context(), userID, id); err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
