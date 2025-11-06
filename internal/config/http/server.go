package http

import (
	"errors"
	"fxrates/internal/config"
	"log"
	"net/http"
)

func StartServer(cfg config.HTTPServer) {
	server := &http.Server{
		Addr: ":" + cfg.Port,
	}
	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Error while starting HTTP server: " + err.Error())
	}
}
