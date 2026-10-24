package handler

import (
	"net/http"
	"time"

	"weathertracker/internal/service"
)

// AuthHandler exposes registration and login endpoints.
type AuthHandler struct {
	auth *service.AuthService
}

// NewAuthHandler builds the auth handler.
func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type credentialsRequest struct {
	Email    string `json:"email" example:"rider@example.com"`
	Password string `json:"password" example:"s3cret-passphrase"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type authResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

// Register godoc
//
//	@Summary      Register a new account
//	@Description  Creates a user and returns a JWT the React SPA can store.
//	@Tags         auth
//	@Accept       json
//	@Produce      json
//	@Param        request body credentialsRequest true "Registration credentials"
//	@Success      201 {object} authResponse
//	@Failure      400 {object} errorResponse
//	@Failure      409 {object} errorResponse
//	@Failure      500 {object} errorResponse
//	@Router       /api/v1/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		handleError(w, err)
		return
	}

	user, token, err := h.auth.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{
		Token: token,
		User:  userResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt},
	})
}

// Login godoc
//
//	@Summary      Authenticate and obtain a JWT
//	@Description  Verifies credentials and returns a Bearer token.
//	@Tags         auth
//	@Accept       json
//	@Produce      json
//	@Param        request body credentialsRequest true "Login credentials"
//	@Success      200 {object} authResponse
//	@Failure      400 {object} errorResponse
//	@Failure      401 {object} errorResponse
//	@Failure      500 {object} errorResponse
//	@Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		handleError(w, err)
		return
	}

	user, token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, authResponse{
		Token: token,
		User:  userResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt},
	})
}
