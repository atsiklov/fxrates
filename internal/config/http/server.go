package http

import (
	"errors"
	"fxrates/internal/config"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func StartServer(cfg config.HTTPServer, router *chi.Mux) {
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}
	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Error while starting HTTP server: " + err.Error())
	}
}
