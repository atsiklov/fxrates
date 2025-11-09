package main

import (
	"context"
	"fxrates/internal/adapters/httpclient"
	"fxrates/internal/adapters/postgres"
	"fxrates/internal/config"
	"fxrates/internal/config/db"
	myHTTP "fxrates/internal/config/http"
	"fxrates/internal/rate"
	"fxrates/internal/rate/handler"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

func main() {
	appCfg := config.Init()
	logrus.Info("Config initialization successful")

	// Create a root context tied to OS signals for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Use a bounded startup context for DB connectivity and initial reads
	startupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := db.CreatePool(startupCtx, appCfg.DbServer)
	if err != nil {
		logrus.WithError(err).Error("Error connecting to postgres")
		return
	}
	defer pool.Close()
	logrus.Info("Postgres connection successful")

	// Load supported currencies from DB
	supportedCurrencies, err := config.LoadSupportedCurrencies(startupCtx, pool)
	if err != nil {
		logrus.WithError(err).Error("Failed to load supported currencies")
		return
	}
	logrus.Info("Supported currencies loaded")

	baseHTTPClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Clients
	rateClient := httpclient.NewExchangeRateClient(baseHTTPClient, "https://example.com/latest")

	// Repositories
	rateUpdatesRepo := postgres.NewRateUpdatesRepository(pool)
	rateRepo := postgres.NewRateRepository(pool)

	// Services
	rateService := rate.NewService(rateUpdatesRepo, rateRepo)
	rateValidator := rate.NewValidator(supportedCurrencies)
	scheduler := rate.NewScheduler(rateUpdatesRepo, rateClient)
	// Ensure scheduler stops before DB pool closes
	defer func() {
		if shutDownErr := scheduler.Shutdown(); shutDownErr != nil {
			logrus.Errorf("Scheduler shutdown error: %v", shutDownErr)
		}
	}()
	// Start scheduler tied to root context; handle error gracefully
	if startErr := scheduler.Start(ctx); startErr != nil {
		logrus.WithError(startErr).Error("Failed to start scheduler")
		return
	}
	logrus.Info("Scheduler activation successful")

	// handlers
	rateHandler := handler.NewRateHandler(rateValidator, rateService)

	router := chi.NewRouter() // todo: add logging
	router.Post("/api/v1/rates/updates", rateHandler.ScheduleUpdate)
	router.Get("/api/v1/rates/updates/{id}", rateHandler.GetByUpdateID)
	router.Get("/api/v1/rates/{base}/{quote}", rateHandler.GetByCodes)

	logrus.Info("Starting http server")
	// Block until context is canceled, then perform graceful shutdown.
	if serverErr := myHTTP.StartServer(ctx, appCfg.HTTPServer, router); serverErr != nil {
		// Cancel the root context to stop scheduler and other in-flight work
		stop()
		logrus.Errorf("HTTP server error: %v", serverErr)
		return
	}
}
