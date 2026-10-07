package handlers

import (
	"apiproject/internal/database"
	"apiproject/internal/models"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

func doRequest(t *testing.T, store TaskStore, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	NewHandlers(store).Routes().ServeHTTP(rec, req)
	return rec
}
func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(rec.Body).Decode(&v); err != nil {
		t.Fatalf("failed to decode response body: %s", err)
	}
	return v
}

func TestHandlers_GetTaskByID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		storeTask  *models.Task
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{name: "found", path: "/tasks/1", storeTask: &models.Task{ID: 1, Title: "Learn Go"}, wantStatus: http.StatusOK},
		{name: "not found", path: "/tasks/999", storeErr: database.ErrNotFound, wantStatus: http.StatusNotFound, wantError: "task not found"},
		{name: "invalid id", path: "/tasks/abc", wantStatus: http.StatusBadRequest, wantError: "invalid task id"},
		{name: "zero id", path: "/tasks/0", wantStatus: http.StatusBadRequest, wantError: "invalid task id"},
		{name: "timeout", path: "/tasks/1", storeErr: context.DeadlineExceeded, wantStatus: http.StatusGatewayTimeout, wantError: "request timed out"},
		{name: "db failure", path: "/tasks/1", storeErr: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantError: "internal server error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{
				getByIDFn: func(ctx context.Context, id int) (*models.Task, error) {
					return tt.storeTask, tt.storeErr
				},
			}
			rec := doRequest(t, store, http.MethodGet, tt.path, "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("want status %d, got %d", tt.wantStatus, rec.Code)
			}
			if tt.wantError != "" {
				body := decodeBody[map[string]string](t, rec)
				if body["error"] != tt.wantError {
					t.Errorf("want error: %s, got %s", tt.wantError, body["error"])
				}
				return
			}
			task := decodeBody[models.Task](t, rec)
			if task.ID != tt.storeTask.ID || task.Title != tt.storeTask.Title {
				t.Errorf("task = %+v, want %+v", task, *tt.storeTask)
			}
		})
	}
}

func TestHandlers_CreateTask(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
		wantCalled bool
	}{
		{name: "valid", body: `{"title":"  Learn Go  "}`, wantStatus: http.StatusCreated, wantCalled: true},
		{name: "empty body", body: ``, wantStatus: http.StatusBadRequest, wantError: "request body is empty"},
		{name: "invalid json", body: `{bad`, wantStatus: http.StatusBadRequest, wantError: "invalid JSON"},
		{name: "unknown field", body: `{"titel":"x"}`, wantStatus: http.StatusBadRequest, wantError: `unknown field "titel"`},
		{name: "blank title", body: `{"title":"   "}`, wantStatus: http.StatusBadRequest, wantError: "title is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotInput *models.CreateTaskInput
			store := &fakeStore{
				createFn: func(ctx context.Context, input *models.CreateTaskInput) (*models.Task, error) {
					gotInput = input
					return &models.Task{ID: 10, Title: input.Title}, nil
				},
			}
			rec := doRequest(t, store, http.MethodPost, "/tasks", tt.body)

			if rec.Code != tt.wantStatus {
				t.Fatalf("want status %d, got %d", tt.wantStatus, rec.Code)
			}
			if called := gotInput != nil; called != tt.wantCalled {
				t.Fatalf("want called %t, got %t", tt.wantCalled, called)
			}
			if tt.wantError != "" {
				body := decodeBody[map[string]string](t, rec)
				if body["error"] != tt.wantError {
					t.Fatalf("want error: %s, got %s", tt.wantError, body["error"])
				}
				return
			}
			if gotInput.Title != "Learn Go" {
				t.Errorf("task = %+v, want %+v", gotInput.Title, "Learn Go")
			}
		})
	}
}

func TestHandlers_DeleteTaskByID(t *testing.T) {
	tests := []struct {
		name       string
		storeErr   error
		wantStatus int
	}{
		{name: "deleted", wantStatus: http.StatusOK},
		{name: "not found", storeErr: database.ErrNotFound, wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotID int
			store := &fakeStore{
				deleteFn: func(ctx context.Context, id int) error {
					gotID = id
					return tt.storeErr
				},
			}
			rec := doRequest(t, store, http.MethodDelete, "/tasks/7", "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("want status %d, got %d", tt.wantStatus, rec.Code)
			}
			if gotID != 7 {
				t.Errorf("task = %+v, want %+v", gotID, 7)
			}
		})
	}
}

func TestMethodNotAllowed(t *testing.T) {
	rec := doRequest(t, &fakeStore{}, http.MethodPatch, "/tasks/1", "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}
