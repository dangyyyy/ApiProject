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
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxBodyBytes    = 1 << 20
	dbTimeout       = 3 * time.Second
	defaultLimit    = 20
	maxLimit        = 100
	maxSearchLength = 100
)

type TaskStore interface {
	List(ctx context.Context, filter models.TaskFilter) ([]models.Task, int, error)
	GetByID(ctx context.Context, id int) (*models.Task, error)
	Create(ctx context.Context, input *models.CreateTaskInput) (*models.Task, error)
	Update(ctx context.Context, id int, input *models.UpdateTaskInput) (*models.Task, error)
	Delete(ctx context.Context, id int) error
}
type Handlers struct {
	store TaskStore
}

func NewHandlers(store TaskStore) *Handlers {
	return &Handlers{store: store}
}
func (h *Handlers) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /tasks", h.ListTasks)
	mux.HandleFunc("POST /tasks", h.CreateTask)
	mux.HandleFunc("GET /tasks/{id}", h.GetTaskByID)
	mux.HandleFunc("PUT /tasks/{id}", h.UpdateTaskByID)
	mux.HandleFunc("DELETE /tasks/{id}", h.DeleteTaskByID)
	return mux
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

func parseTaskFilter(q url.Values) (models.TaskFilter, error) {
	f := models.TaskFilter{
		Search: strings.TrimSpace(q.Get("search")),
		Sort:   "created_at",
		Order:  "desc",
		Limit:  defaultLimit,
	}

	if v := q.Get("completed"); v != "" {
		completed, err := strconv.ParseBool(v)
		if err != nil {
			return f, errors.New("completed must be true or false")
		}
		f.Completed = &completed
	}

	if v := q.Get("limit"); v != "" {
		limit, err := strconv.Atoi(v)
		if err != nil || limit < 1 || limit > maxLimit {
			return f, fmt.Errorf("limit must be between 1 and %d", maxLimit)
		}
		f.Limit = limit
	}

	if v := q.Get("offset"); v != "" {
		offset, err := strconv.Atoi(v)
		if err != nil || offset < 0 {
			return f, errors.New("offset must be a non-negative integer")
		}
		f.Offset = offset
	}

	if v := q.Get("sort"); v != "" {
		if v != "created_at" && v != "title" {
			return f, errors.New("sort must be created_at or title")
		}
		f.Sort = v
	}

	if v := q.Get("order"); v != "" {
		if v != "asc" && v != "desc" {
			return f, errors.New("order must be asc or desc")
		}
		f.Order = v
	}

	if utf8.RuneCountInString(f.Search) > maxSearchLength {
		return f, fmt.Errorf("search must be at most %d characters", maxSearchLength)
	}

	return f, nil
}

func (h *Handlers) ListTasks(w http.ResponseWriter, r *http.Request) {
	filter, err := parseTaskFilter(r.URL.Query())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()

	tasks, total, err := h.store.List(ctx, filter)
	if err != nil {
		respondWithStoreError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, models.TaskPage{
		Items:  tasks,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
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
