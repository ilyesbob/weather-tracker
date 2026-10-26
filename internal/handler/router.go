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

type Deps struct {
	Auth           *service.AuthService
	Locations      *service.LocationService
	Ranking        *service.RankingService
	Windsurf       *service.WindsurfService
	Activity       *service.ActivityService
	AllowedOrigins []string
	Logger         *slog.Logger
}

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
	activityHandler := NewActivityHandler(d.Activity)

	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/index.html", http.StatusMovedPermanently)
	})
	r.Handle("/docs/*", httpSwagger.WrapHandler)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Get("/locations/search", locationHandler.Search)

		r.Group(func(r chi.Router) {
			r.Use(auth)

			r.Get("/users/me/locations", locationHandler.ListSaved)
			r.Post("/users/me/locations", locationHandler.AddSaved)
			r.Delete("/users/me/locations/{id}", locationHandler.DeleteSaved)

			r.Get("/users/me/windsurf-profile", windsurfHandler.GetProfile)
			r.Put("/users/me/windsurf-profile", windsurfHandler.SaveProfile)

			r.Post("/weather/rank", rankingHandler.Rank)
			r.Post("/windsurf/recommendations", windsurfHandler.Recommend)
			r.Post("/activity/recommendations", activityHandler.Recommend)
		})
	})

	return r
}
