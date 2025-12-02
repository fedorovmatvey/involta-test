package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fedorovmatvey/involta-test/internal/cache"
	"github.com/fedorovmatvey/involta-test/internal/config"
	"github.com/fedorovmatvey/involta-test/internal/handler"
	"github.com/fedorovmatvey/involta-test/internal/service"
	"github.com/fedorovmatvey/involta-test/internal/storage"
	_ "github.com/restream/reindexer/v3/bindings/cproto"
)

// @title Involta Reindexer Service
// @version 1.0
// @description Microservice for document management with Reindexer storage.
// @host localhost:8080
// @BasePath /
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}

	slog.Info("Starting application", "env", cfg.App.Env, "port", cfg.Server.Port)

	store, err := storage.New(cfg.Reindexer.DSN, cfg.Reindexer.Namespace)
	if err != nil {
		return fmt.Errorf("storage init: %w", err)
	}

	defer func() {
		slog.Info("Closing storage connection...")
		if err := store.Close(); err != nil {
			slog.Error("Failed to close storage", "error", err)
		}
	}()

	initCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := store.CheckConnection(initCtx); err != nil {
		return fmt.Errorf("storage connection check: %w", err)
	}
	slog.Info("Storage connection established")

	documentCache := cache.New(cfg.Cache.TTL, cfg.Cache.CleanupInterval, cfg.Cache.Capacity)
	defer func() {
		slog.Info("Stopping cache cleanup...")
		documentCache.Stop()
	}()

	srv := service.New(store, documentCache)
	h := handler.New(srv)

	router := h.InitRoutes()

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	serverErr := make(chan error, 1)

	go func() {
		slog.Info("Server listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("server listen: %w", err)
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		slog.Info("Shutting down server...")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	slog.Info("Server stopped gracefully")
	return nil
}
