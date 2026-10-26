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

	_ "weathertracker/docs"
	"weathertracker/internal/config"
	"weathertracker/internal/handler"
	"weathertracker/internal/repository/memory"
	"weathertracker/internal/repository/weather"
	"weathertracker/internal/service"
)

const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// @title           Weather Tracker API
// @version         1.0
// @description     MVP backend for the Weather Tracker service. Implements User Story 1 (city ranking for trip planning by weather priorities), User Story 2 (windsurfing spot and time-slot recommendations), and User Story 3 (outdoor activity time window recommendations).
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

	_ = godotenv.Load()

	cfg := config.Load()

	userStore := memory.NewUserStore()
	catalog := memory.NewLocationStore()
	savedStore := memory.NewSavedLocationStore()
	profileStore := memory.NewWindsurfProfileStore()

	openMeteo := weather.NewClient(cfg.WeatherBaseURL, cfg.GeocodingBaseURL, cfg.UpstreamTimeout)
	cachedWeather := weather.NewCachedProvider(openMeteo, cfg.WeatherCacheTTL)

	authService := service.NewAuthService(userStore, cfg.JWTSecret, cfg.JWTTTL)
	locationService := service.NewLocationService(openMeteo, catalog, savedStore)
	rankingService := service.NewRankingService(catalog, cachedWeather, cfg.StaleThreshold)
	windsurfService := service.NewWindsurfService(profileStore, savedStore, catalog, cachedWeather, cfg.StaleThreshold)
	activityService := service.NewActivityService(catalog, savedStore, cachedWeather, cfg.StaleThreshold)

	router := handler.NewRouter(handler.Deps{
		Auth:           authService,
		Locations:      locationService,
		Ranking:        rankingService,
		Windsurf:       windsurfService,
		Activity:       activityService,
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

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

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

	signal.Stop(signals)

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
