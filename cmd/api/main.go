// Package main is the entrypoint of the Weather Tracker API. It wires the
// dependencies, starts the HTTP server and performs a graceful shutdown.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	_ "weathertracker/docs" // generated OpenAPI spec registered on init
	"weathertracker/internal/config"
	"weathertracker/internal/handler"
	"weathertracker/internal/repository/memory"
	"weathertracker/internal/repository/weather"
	"weathertracker/internal/service"
)

// HTTP server timeouts and shutdown budget. Named so the whole team can tune
// them in one place.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// @title           Weather Tracker API
// @version         1.0
// @description     MVP backend for the Weather Tracker service. Implements User Story 1 (city ranking for trip planning by weather priorities) and User Story 2 (windsurfing spot and time-slot recommendations).
// @termsOfService  http://swagger.io/terms/
//
// @contact.name    Weather Tracker Team
// @contact.email   team@weathertracker.dev
//
// @host      localhost:8080
// @BasePath  /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT returned by /api/v1/auth/login.
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// Load a local .env file if present. A missing file is fine: in production
	// configuration is provided through real environment variables. Existing
	// environment variables always win over .env values.
	_ = godotenv.Load()

	cfg := config.Load()

	// Infrastructure: in-memory adapters for the MVP. Swapping them for
	// PostgreSQL/Redis requires no change in the service layer thanks to the
	// repository interfaces.
	userStore := memory.NewUserStore()
	catalog := memory.NewLocationStore()
	savedStore := memory.NewSavedLocationStore()
	profileStore := memory.NewWindsurfProfileStore()

	openMeteo := weather.NewClient(cfg.WeatherBaseURL, cfg.GeocodingBaseURL, cfg.UpstreamTimeout)
	cachedWeather := weather.NewCachedProvider(openMeteo, cfg.WeatherCacheTTL)

	// Services.
	authService := service.NewAuthService(userStore, cfg.JWTSecret, cfg.JWTTTL)
	locationService := service.NewLocationService(openMeteo, catalog, savedStore)
	rankingService := service.NewRankingService(catalog, cachedWeather, cfg.StaleThreshold)
	windsurfService := service.NewWindsurfService(profileStore, savedStore, catalog, cachedWeather, cfg.StaleThreshold)

	router := handler.NewRouter(handler.Deps{
		Auth:           authService,
		Locations:      locationService,
		Ranking:        rankingService,
		Windsurf:       windsurfService,
		AllowedOrigins: cfg.AllowedOrigins,
		Logger:         logger,
	})

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	// Tier 1: catch the shutdown signals (Ctrl+C -> SIGINT, container stop ->
	// SIGTERM) and run a graceful drain before the process exits.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	// ListenAndServe blocks, so it runs in a goroutine while main waits for a
	// signal or a fatal server error.
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting",
			slog.String("addr", cfg.Addr),
			slog.String("swagger_path", "/docs/index.html"),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case sig := <-signals:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))
	case err := <-serverErr:
		logger.Error("server failed", slog.String("error", err.Error()))
	}

	// Stop intercepting signals: a second Ctrl+C now triggers the default
	// behaviour (immediate termination) as an escape hatch if a request hangs.
	signal.Stop(signals)

	// Give in-flight requests up to shutdownTimeout to finish, then force close
	// so a stuck handler cannot block the process forever.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed, forcing close", slog.String("error", err.Error()))
		if closeErr := server.Close(); closeErr != nil {
			logger.Error("forced close failed", slog.String("error", closeErr.Error()))
		}
	}
	logger.Info("server stopped")
}
