package http

import (
	"context"
	"errors"
	"fxrates/internal/config"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

// Start runs HTTP server and shuts it down gracefully on ctx cancellation.
func Start(ctx context.Context, cfg config.HTTPServer, router *chi.Mux) error {
	listener, listenErr := net.Listen("tcp", ":"+cfg.Port)
	if listenErr != nil {
		return listenErr
	}
	logrus.Infof("✅ HTTP server listening on %s", cfg.Port)

	server := &http.Server{Handler: router}
	errCh := make(chan error, 1)
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
			return shutdownErr
		}
		return nil
	case serveErr := <-errCh:
		return serveErr
	}
}
