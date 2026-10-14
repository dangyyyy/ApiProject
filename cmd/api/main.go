package main

import (
	"apiproject/internal/config"
	"apiproject/internal/database"
	"apiproject/internal/handlers"
	"apiproject/internal/logger"
	"apiproject/internal/middleware"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Error("failed to load .env file", "error", err)
		os.Exit(1)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	slog.SetDefault(logger.New(cfg.LogFormat, cfg.LogLevel))

	if err := run(cfg); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := database.Connect(ctx, cfg.DB.URL())
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close database", "error", err)
		}
	}()
	slog.Info("database connected")
	if err := database.Migrate(ctx, db); err != nil {
		return err
	}

	taskStore := database.NewTaskStore(db)
	handler := handlers.NewHandlers(taskStore)

	handlerChain := middleware.Chain(handler.Routes(),
		middleware.RequestID,
		middleware.Logging,
		middleware.Recover,
		middleware.JSONErrors,
	)
	srv := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           handlerChain,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting server", "port", cfg.ServerPort)
		errCh <- srv.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)

	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}
	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}
	slog.Info("server stopped")
	return nil
}
