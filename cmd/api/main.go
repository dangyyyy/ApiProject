package main

import (
	"apiproject/internal/database"
	"apiproject/internal/handlers"
	"apiproject/internal/logger"
	"apiproject/internal/middleware"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const shutdownTimeout = 10 * time.Second

func main() {
	slog.SetDefault(logger.New(os.Getenv("LOG_FORMAT"), os.Getenv("LOG_LEVEL")))
	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://taskuser:taskpass@localhost:5432/tasksdb?sslmode=disable"
	}
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := database.Connect(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	slog.Info("database connected")

	taskStore := database.NewTaskStore(db)
	handler := handlers.NewHandlers(taskStore)

	handlerChain := middleware.Chain(handler.Routes(),
		middleware.RequestID,
		middleware.Logging,
		middleware.Recover,
		middleware.JSONErrors,
	)
	srv := &http.Server{
		Addr:              ":" + serverPort,
		Handler:           handlerChain,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting server", "port", serverPort)
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
