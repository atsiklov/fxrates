package main

import (
	"context"
	"fxrates/internal/config"
	"fxrates/internal/config/db"
	"fxrates/internal/config/http"
	"fxrates/internal/delivery"
	"fxrates/internal/services"
	"fxrates/internal/storage"
	"log"
	"os"

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

	rateRepo := storage.NewRatePgRepository(pool)
	rateService := services.RateService{RateRepo: rateRepo}
	rateHandler := delivery.RateHandler{RateService: &rateService}

	router := chi.NewRouter()
	// router.Use(// todo: add logging)
	router.Post("/api/v1/rates/updates", rateHandler.UpdateRate)
	router.Get("/api/v1/rates/updates/{id}", rateHandler.GetRateInfo)
	router.Get("/api/v1/rates/{code}", rateHandler.GetRate)
	http.StartServer(appCfg.HTTPServer, router)
}
