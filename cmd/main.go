package main

import (
	"context"
	"fxrates/internal/adapters/postgres"
	"fxrates/internal/adapters/ratesapi"
	"fxrates/internal/config"
	"fxrates/internal/config/db"
	myHTTP "fxrates/internal/config/http"
	"fxrates/internal/rates"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
)

func main() {
	log.SetOutput(os.Stdout)
	appCfg := config.Init()

	ctx := context.Background()
	pool := db.CreatePool(ctx, appCfg.DbServer)
	if pool == nil {
		panic("")
	} // todo: ...
	defer pool.Close()
	log.Println("Successfully connected to DB")

	// set up an http client to make external rates requests
	baseHTTPClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	ratesClient := ratesapi.NewClient(baseHTTPClient, "https://example.com/latest")

	rateUpdatesRepo := postgres.NewPostgresRateUpdatesRepository(pool)
	rateRepo := postgres.NewPostgresRateRepository(pool)
	scheduler := rates.Scheduler{RateUpdatesRepo: rateUpdatesRepo, Client: ratesClient}
	scheduler.CreateAndRun()

	rateService := rates.RateService{Repo: rateRepo}
	rateHandler := rates.RateHandler{RateService: &rateService}

	router := chi.NewRouter() // todo: add logging
	router.Post("/api/v1/rates/updates", rateHandler.UpdateRate)
	router.Get("/api/v1/rates/updates/{id}", rateHandler.GetByUpdateID)
	router.Get("/api/v1/rates/{code}", rateHandler.GetRate)
	myHTTP.StartServer(appCfg.HTTPServer, router)
}
