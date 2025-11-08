package main

import (
	"context"
	"fxrates/internal/adapters/httpclient"
	"fxrates/internal/adapters/postgres"
	"fxrates/internal/config"
	"fxrates/internal/config/db"
	myHTTP "fxrates/internal/config/http"
	"fxrates/internal/rate"
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

	// set up an http client to make external rate requests
	baseHTTPClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// clients
	rateClient := httpclient.NewExchangeRateClient(baseHTTPClient, "https://example.com/latest")

	// repositories
	rateUpdatesRepo := postgres.NewPostgresRateUpdatesRepository(pool)
	rateRepo := postgres.NewPostgresRateRepository(pool)

	// services
	rateService := rate.NewService(rateRepo)
	scheduler := rate.NewScheduler(rateUpdatesRepo, rateClient)
	scheduler.CreateAndRun()
	logrus.Info("Scheduler activation successful")

	// handlers
	rateHandler := rate.NewHandler(rateService)

	router := chi.NewRouter() // todo: add logging
	router.Post("/api/v1/rates/updates", rateHandler.UpdateRate)
	router.Get("/api/v1/rates/updates/{id}", rateHandler.GetByUpdateID)
	router.Get("/api/v1/rates/{code}", rateHandler.GetRate)

	logrus.Info("Starting http server")
	myHTTP.StartServer(appCfg.HTTPServer, router)
}
