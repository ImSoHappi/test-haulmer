package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ImSoHappi/test-haulmer/acquirer-mock/internal/handler"
	"github.com/ImSoHappi/test-haulmer/acquirer-mock/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	port := strings.TrimSpace(os.Getenv("HTTP_PORT"))
	if port == "" {
		logger.Error("HTTP_PORT environment variable is required")
		os.Exit(1)
	}

	timeoutDelayStr := os.Getenv("TIMEOUT_DELAY_MS")
	timeoutDelay := 3500 * time.Millisecond
	if timeoutDelayStr != "" {
		if d, err := time.ParseDuration(timeoutDelayStr + "ms"); err == nil {
			timeoutDelay = d
		}
	}

	store := storage.NewMemoryStore()
	h := handler.NewAuthorizationHandler(
		store,
		handler.WithTimeoutDelay(timeoutDelay),
		handler.WithLogger(logger),
	)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("starting acquirer-mock HTTP service",
			"port", port,
			"timeout_delay", timeoutDelay.String(),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutting down acquirer-mock gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("acquirer-mock stopped cleanly")
	fmt.Println("acquirer-mock exit")
}
