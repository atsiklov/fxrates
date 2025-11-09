package api

import (
	"fxrates/internal/rate/handler"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(rateHandler *handler.Handler) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Use(middleware.Heartbeat("/healthz"))

	router.Post("/api/v1/rates/updates", rateHandler.ScheduleUpdate)
	router.Get("/api/v1/rates/updates/{id}", rateHandler.GetByUpdateID)
	router.Get("/api/v1/rates/{base}/{quote}", rateHandler.GetByCodes)
	return router
}
