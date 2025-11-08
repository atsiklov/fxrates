package http

import (
	"errors"
	"fxrates/internal/config"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

func StartServer(cfg config.HTTPServer, router *chi.Mux) {
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}
	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logrus.Fatalf("Error starting server: %s", err) // todo: handle
	}
}
