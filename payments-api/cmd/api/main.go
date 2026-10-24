package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/acquirer"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/config"
	apihttp "github.com/ImSoHappi/test-haulmer/payments-api/internal/http"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/payment"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/postgres"
)

func main() {
	// 1. Load runtime configuration with fail-fast validation
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	// 2. Configure structured JSON logger (PCI-DSS compliant: zero PAN/CVV)
	var logLevel slog.Level
	switch cfg.LogLevel {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	logger.Info("starting payments-api service",
		slog.String("port", cfg.Port),
		slog.Int64("payment_max_amount_clp", cfg.PaymentMaxAmountCLP),
	)

	// 3. Apply database migrations
	if err := postgres.RunMigrations(cfg.DatabaseURL); err != nil {
		logger.Error("failed to apply database migrations", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("database migrations applied successfully")

	// 4. Initialize PostgreSQL connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database pool", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to PostgreSQL pool successfully")

	// 5. Initialize Secondary Adapters
	repo := postgres.NewRepository(pool)
	acquirerClient, err := acquirer.NewClient(acquirer.ClientConfig{
		BaseURL: cfg.AcquirerBaseURL,
	})
	if err != nil {
		logger.Error("failed to initialize acquirer client", "error", err)
		os.Exit(1)
	}

	// 6. Initialize Core Application Service
	paymentService := payment.NewService(repo, acquirerClient, cfg.PaymentMaxAmountCLP)

	// 7. Initialize HTTP Handlers, Router, and Server
	paymentHandler := apihttp.NewPaymentHandler(paymentService)
	router := apihttp.NewRouter(logger, paymentHandler)
	server := apihttp.NewServer(cfg.Port, router)

	// 8. Start HTTP Server in background goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP server listening", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// 9. Graceful Shutdown listener
	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Error("server encountered fatal runtime error", slog.Any("error", err))
		os.Exit(1)
	case sig := <-shutdownSignals:
		logger.Info("received termination signal, initiating graceful shutdown", slog.String("signal", sig.String()))

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("server shutdown timed out or failed", slog.Any("error", err))
			_ = server.Close()
			os.Exit(1)
		}
		logger.Info("payments-api service stopped cleanly")
	}
}
