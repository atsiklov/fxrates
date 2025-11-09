package api

import (
	"fxrates/internal/rate/handler"

	"github.com/go-chi/chi/v5"
)

func NewRouter(rateHandler *handler.Handler) *chi.Mux {
	router := chi.NewRouter()
	// todo: add logging middleware, recover, etc.

	router.Post("/api/v1/rates/updates", rateHandler.ScheduleUpdate)
	router.Get("/api/v1/rates/updates/{id}", rateHandler.GetByUpdateID)
	router.Get("/api/v1/rates/{base}/{quote}", rateHandler.GetByCodes)
	return router
}
