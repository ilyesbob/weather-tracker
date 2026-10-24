package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"weathertracker/internal/middleware"
	"weathertracker/internal/service"
)

// Deps bundles every dependency the router needs.
type Deps struct {
	Auth           *service.AuthService
	Locations      *service.LocationService
	Ranking        *service.RankingService
	Windsurf       *service.WindsurfService
	AllowedOrigins []string
	Logger         *slog.Logger
}

// NewRouter wires middleware and routes into a ready-to-serve http.Handler.
func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(middleware.Logging(d.Logger))
	r.Use(middleware.CORS(d.AllowedOrigins))

	auth := middleware.Auth(d.Auth)
	authHandler := NewAuthHandler(d.Auth)
	locationHandler := NewLocationHandler(d.Locations)
	rankingHandler := NewRankingHandler(d.Ranking)
	windsurfHandler := NewWindsurfHandler(d.Windsurf)

	// Interactive OpenAPI documentation for the team.
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/index.html", http.StatusMovedPermanently)
	})
	r.Handle("/docs/*", httpSwagger.WrapHandler)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Public endpoints.
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Get("/locations/search", locationHandler.Search)

		// Endpoints requiring a Bearer token.
		r.Group(func(r chi.Router) {
			r.Use(auth)

			r.Get("/users/me/locations", locationHandler.ListSaved)
			r.Post("/users/me/locations", locationHandler.AddSaved)
			r.Delete("/users/me/locations/{id}", locationHandler.DeleteSaved)

			r.Get("/users/me/windsurf-profile", windsurfHandler.GetProfile)
			r.Put("/users/me/windsurf-profile", windsurfHandler.SaveProfile)

			r.Post("/weather/rank", rankingHandler.Rank)
			r.Post("/windsurf/recommendations", windsurfHandler.Recommend)
		})
	})

	return r
}
