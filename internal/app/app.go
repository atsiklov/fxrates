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
	if parsedLvl, parseErr := logrus.ParseLevel(cfgLevel); parseErr != nil {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(parsedLvl)
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
		logrus.WithError(err).Error("Error connecting to db")
		return err
	}
	defer pool.Close()
	logrus.Info("✅ Postgres connection successful")

	// Load supported currencies codes
	supportedCodes, err := loadSupportedCodes(startupCtx, pool)
	if err != nil || len(supportedCodes) == 0 {
		if err == nil {
			err = errors.New("no supported currencies available")
		}
		logrus.WithError(err).Error("Failed to load supported currencies")
		return err
	}
	logrus.Info("✅ Supported currencies loaded")

	// Base HTTP client (configurable timeout)
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

	// Services
	rateService := rate.NewService(rateUpdateRepo, rateRepo)
	rateValidator := rate.NewValidator(supportedCodes)
	scheduler := rate.NewScheduler(rateUpdateRepo, rateClient, time.Duration(appCfg.Scheduler.JobDurationSec)*time.Second)
	// Ensure scheduler stops before DB pool closes
	defer func() {
		if shutDownErr := scheduler.Shutdown(); shutDownErr != nil {
			logrus.Errorf("Scheduler shutdown error: %v", shutDownErr)
		}
	}()
	// Start scheduler tied to root context
	if startErr := scheduler.Start(ctx); startErr != nil {
		logrus.WithError(startErr).Error("Failed to start scheduler")
		return startErr
	}
	logrus.Info("✅ Scheduler activation successful")

	// Handlers and router
	rateHandler := handler.NewRateHandler(rateValidator, rateService)
	router := api.NewRouter(rateHandler)

	logrus.Info("Starting http server")
	// Block until context is canceled, then perform graceful shutdown.
	if serverErr := httpserver.Start(ctx, appCfg.HTTPServer, router); serverErr != nil {
		// Cancel the root context to stop scheduler and other in-flight work
		stop()
		logrus.Errorf("HTTP server error: %v", serverErr)
		return serverErr
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
