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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

func main() {
	appCfg := config.Init()
	logrus.Info("Config initialization successful")

	ctx := context.Background()
	pool, err := db.CreatePool(ctx, appCfg.DbServer)
	if err != nil {
		logrus.Fatalf("Error connecting to postrgres") // todo: handle
	}
	defer pool.Close()
	logrus.Info("Postgres connection successful")

	// load supported currencies from db
	supportedCurrencies, err := config.LoadSupportedCurrencies(ctx, pool)
	if err != nil {
		logrus.Fatalf("Failed to load supported currencies") // todo: handle
	}
	logrus.Info("Supported currencies loaded")

	baseHTTPClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// clients
	rateClient := httpclient.NewExchangeRateClient(baseHTTPClient, "https://example.com/latest")

	// repositories
	rateUpdatesRepo := postgres.NewRateUpdatesRepository(pool)
	rateRepo := postgres.NewRateRepository(pool)

	// services
	rateService := rate.NewService(rateUpdatesRepo, rateRepo)
	rateValidator := rate.NewValidator(supportedCurrencies)
	scheduler := rate.NewScheduler(rateUpdatesRepo, rateClient)
	scheduler.CreateAndRun()
	logrus.Info("Scheduler activation successful")

	// handlers
	rateHandler := handler.NewRateHandler(rateService, rateValidator)

	router := chi.NewRouter() // todo: add logging
	router.Post("/api/v1/rates/updates", rateHandler.ScheduleUpdate)
	router.Get("/api/v1/rates/updates/{id}", rateHandler.GetByUpdateID)
	router.Get("/api/v1/rates/{base}/{quote}", rateHandler.GetByCodes)

	logrus.Info("Starting http server")
	myHTTP.StartServer(appCfg.HTTPServer, router)
}
