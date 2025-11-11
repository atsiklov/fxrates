package api

import (
	_ "fxrates/docs"
	"fxrates/internal/rate/handler"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	swagger "github.com/swaggo/http-swagger"
)

func NewRouter(rateHandler *handler.Handler) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Use(middleware.Heartbeat("/healthz"))

	// Swagger UI
	router.Get("/swagger/*", swagger.WrapHandler)

	router.Post("/api/v1/rates/updates", rateHandler.ScheduleUpdate)
	router.Get("/api/v1/rates/updates/{id}", rateHandler.GetByUpdateID)
	router.Get("/api/v1/rates/supported-currencies", rateHandler.GetSupportedCodes)
	router.Get("/api/v1/rates/{base:[A-Za-z]{3}}/{quote:[A-Za-z]{3}}", rateHandler.GetByCodes)
	return router
}
