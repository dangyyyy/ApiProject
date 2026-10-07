package handlers

import (
	"apiproject/internal/database"
	"apiproject/internal/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	maxBodyBytes = 1 << 20
	dbTimeout    = 3 * time.Second
)

type Handlers struct {
	store *database.TaskStore
}

func NewHandlers(store *database.TaskStore) *Handlers {
	return &Handlers{store: store}
}

func respondWithJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	respondWithJSON(w, statusCode, map[string]string{"error": message})
}

func respondWithStoreError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()
	switch {
	case errors.Is(err, database.ErrNotFound):
		respondWithError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, context.DeadlineExceeded):
		slog.WarnContext(ctx, "store timeout", "error", err)
		respondWithError(w, http.StatusGatewayTimeout, "request timed out")
	case errors.Is(err, context.Canceled):
		slog.InfoContext(ctx, "request canceled by client", "error", err)
	default:
		slog.ErrorContext(ctx, "store error", "error", err)
		respondWithError(w, http.StatusInternalServerError, "internal server error")
	}
}
func parseID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		return 0, errors.New("invalid task id")
	}
	return id, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		switch {
		case errors.Is(err, io.EOF):
			return errors.New("request body is empty")
		case errors.As(err, &maxErr):
			return errors.New("request body is too large")
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			return fmt.Errorf("unknown field %s", strings.TrimPrefix(err.Error(), "json: unknown field "))
		default:
			return errors.New("invalid JSON")
		}
	}
	return nil
}

func (h *Handlers) GetAllTasks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()
	tasks, err := h.store.GetAll(ctx)
	if err != nil {
		respondWithStoreError(w, r, err)
		return
	}
	respondWithJSON(w, http.StatusOK, tasks)
}

func (h *Handlers) GetTaskByID(w http.ResponseWriter, r *http.Request) {

	id, err := parseID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()
	task, err := h.store.GetByID(ctx, id)
	if err != nil {
		respondWithStoreError(w, r, err)
		return
	}
	respondWithJSON(w, http.StatusOK, task)
}

func (h *Handlers) CreateTask(w http.ResponseWriter, r *http.Request) {

	var input models.CreateTaskInput
	if err := decodeJSON(w, r, &input); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := input.Validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()
	task, err := h.store.Create(ctx, &input)
	if err != nil {
		respondWithStoreError(w, r, err)
		return
	}
	respondWithJSON(w, http.StatusCreated, task)
}

func (h *Handlers) UpdateTaskByID(w http.ResponseWriter, r *http.Request) {

	id, err := parseID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	var input models.UpdateTaskInput
	if err := decodeJSON(w, r, &input); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := input.Validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()
	task, err := h.store.Update(ctx, id, &input)
	if err != nil {
		respondWithStoreError(w, r, err)
		return
	}
	respondWithJSON(w, http.StatusOK, task)
}

func (h *Handlers) DeleteTaskByID(w http.ResponseWriter, r *http.Request) {

	id, err := parseID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()
	if err := h.store.Delete(ctx, id); err != nil {
		respondWithStoreError(w, r, err)
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})
}
