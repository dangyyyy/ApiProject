package main

import (
	"apiproject/internal/database"
	"apiproject/internal/handlers"
	"apiproject/internal/middleware"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		log.Fatalf("%+v", err)
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
	log.Println("Database connected")

	taskStore := database.NewTaskStore(db)
	handler := handlers.NewHandlers(taskStore)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /tasks", handler.GetAllTasks)
	mux.HandleFunc("POST /tasks", handler.CreateTask)
	mux.HandleFunc("GET /tasks/{id}", handler.GetTaskByID)
	mux.HandleFunc("PUT /tasks/{id}", handler.UpdateTaskByID)
	mux.HandleFunc("DELETE /tasks/{id}", handler.DeleteTaskByID)
	handlerChain := middleware.Chain(mux,
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
		log.Printf("Starting server on port %s", serverPort)
		errCh <- srv.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)

	case <-ctx.Done():
		log.Println("Shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}
	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server  error: %w", err)
	}
	log.Println("Server shutdown")
	return nil
}
