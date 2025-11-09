package http

import (
	"context"
	"errors"
	"fxrates/internal/config"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// StartServer runs HTTP server and shuts it down gracefully on ctx cancellation.
func StartServer(ctx context.Context, cfg config.HTTPServer, router *chi.Mux) error {
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errCh:
		return err
	}
}
