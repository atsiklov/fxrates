package app

import (
	"context"
	"errors"
	"fmt"
	"fxrates/internal/platform/db"
	httpserver "fxrates/internal/platform/http"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"fxrates/internal/adapters/cache"
	"fxrates/internal/adapters/httpclient"
	"fxrates/internal/adapters/postgres"
	"fxrates/internal/api"
	"fxrates/internal/config"
	"fxrates/internal/rate"
	"fxrates/internal/rate/handler"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// Run wires the application components, starts HTTP server and scheduler
func Run() error {
	appCfg, err := config.Init()
	if err != nil {
		return err
	}
	// Logger
	logrus.SetOutput(os.Stdout)
	cfgLevel := appCfg.Logging.Level
	if parsedLevel, parseErr := logrus.ParseLevel(cfgLevel); parseErr != nil {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(parsedLevel)
	}
	logrus.Info("✅ Config initialization successful")

	// Root context bound to OS signals for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Bounded context for startup operations (DB connect, initial reads)
	startupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// DB pool
	pool, err := db.CreatePoolAndPing(startupCtx, appCfg.DbServer)
	if err != nil {
		return fmt.Errorf("failed to establish DB connection: %w", err)
	}
	defer pool.Close()
	logrus.Info("✅ Postgres connection successful")

	// Load supported currencies codes
	supportedCodes, err := loadSupportedCodes(startupCtx, pool)
	if err != nil || len(supportedCodes) == 0 {
		if err == nil {
			err = errors.New("no currencies available")
		}
		return fmt.Errorf("error loading supported currencies: %w", err)
	}

	// Base HTTP client
	httpTimeout := time.Duration(appCfg.HTTPClient.TimeoutSeconds) * time.Second
	if httpTimeout <= 0 {
		httpTimeout = 10 * time.Second
	}
	baseHTTPClient := &http.Client{Timeout: httpTimeout}

	// External clients
	exchangeAPIBaseURL := strings.TrimSuffix(appCfg.ExchangeRateAPI.BaseURL, "/")
	if appCfg.ExchangeRateAPI.APIKey == "" {
		return fmt.Errorf("exchange rate api key is required")
	}
	rateClient := httpclient.NewExchangeRateClient(
		baseHTTPClient,
		fmt.Sprintf("%s/%s/latest", exchangeAPIBaseURL, appCfg.ExchangeRateAPI.APIKey),
	)

	// Repositories
	rateUpdateRepo := postgres.NewRateUpdateRepository(pool)
	rateRepo := postgres.NewRateRepository(pool)

	// Cache
	rateUpdateCache, err := cache.NewRateUpdateCache(appCfg.Cache.RateUpdatesMaxItems)
	if err != nil {
		return fmt.Errorf("cache initialization failed: %w", err)
	}
	defer rateUpdateCache.Close()

	// Services
	rateService := rate.NewService(rateUpdateRepo, rateRepo, rateUpdateCache)
	rateValidator := rate.NewValidator(supportedCodes)
	scheduler := rate.NewScheduler(rateUpdateRepo, rateClient, rateUpdateCache, time.Duration(appCfg.Scheduler.UpdateRatesJobDurationSec)*time.Second)
	// Ensure scheduler stops before DB pool closes
	defer func() {
		if shutDownErr := scheduler.Shutdown(); shutDownErr != nil {
			logrus.Errorf("scheduler shutdown error: %v", shutDownErr)
		}
	}()
	// Start scheduler tied to root context
	if startErr := scheduler.Start(ctx); startErr != nil {
		return fmt.Errorf("scheduler initialization failed: %w", startErr)
	}
	logrus.Info("✅ Scheduler activation successful")

	// Handlers and router
	rateHandler := handler.NewRateHandler(rateValidator, rateService)
	router := api.NewRouter(rateHandler)

	// Block until context is canceled, then perform graceful shutdown.
	if serverErr := httpserver.Start(ctx, appCfg.HTTPServer, router); serverErr != nil {
		// Cancel the root context to stop scheduler and other in-flight work
		stop()
		return fmt.Errorf("HTTP server error: %w", serverErr)
	}
	return nil
}

// loadSupportedCodes loads supported currencies codes from DB
func loadSupportedCodes(ctx context.Context, pool *pgxpool.Pool) (map[string]struct{}, error) {
	rows, err := pool.Query(ctx, `select code from currencies`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]struct{})
	for rows.Next() {
		var c string
		if err = rows.Scan(&c); err != nil {
			return nil, err
		}
		m[c] = struct{}{}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return m, nil
}
