package handlers

import (
	"apiproject/internal/models"
	"context"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func ptr[T any](v T) *T {
	return &v
}

func TestParseTaskFilter(t *testing.T) {
	defaults := models.TaskFilter{Sort: "created_at", Order: "desc", Limit: defaultLimit}

	tests := []struct {
		name    string
		query   string
		want    models.TaskFilter
		wantErr string
	}{
		{name: "defaults", query: "", want: defaults},
		{
			name:  "all params",
			query: "completed=true&search=+go+&sort=title&order=asc&limit=5&offset=10",
			want:  models.TaskFilter{Completed: ptr(true), Search: "go", Sort: "title", Order: "asc", Limit: 5, Offset: 10},
		},
		{name: "completed false", query: "completed=false", want: models.TaskFilter{Completed: ptr(false), Sort: "created_at", Order: "desc", Limit: defaultLimit}},
		{name: "bad completed", query: "completed=yes", wantErr: "completed must be true or false"},
		{name: "limit zero", query: "limit=0", wantErr: "limit must be between 1 and 100"},
		{name: "limit too big", query: "limit=101", wantErr: "limit must be between 1 and 100"},
		{name: "limit not a number", query: "limit=abc", wantErr: "limit must be between 1 and 100"},
		{name: "negative offset", query: "offset=-1", wantErr: "offset must be a non-negative integer"},
		{name: "unknown sort", query: "sort=id%3BDROP+TABLE+tasks", wantErr: "sort must be created_at or title"},
		{name: "bad order", query: "order=up", wantErr: "order must be asc or desc"},
		{name: "search too long", query: "search=" + strings.Repeat("a", 101), wantErr: "search must be at most 100 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("parse query: %v", err)
			}

			got, err := parseTaskFilter(q)

			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filter = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestListTasks(t *testing.T) {
	var gotFilter models.TaskFilter
	store := &fakeStore{
		listFn: func(ctx context.Context, filter models.TaskFilter) ([]models.Task, int, error) {
			gotFilter = filter
			return []models.Task{{ID: 3, Title: "Learn Go"}}, 42, nil
		},
	}

	rec := doRequest(t, store, http.MethodGet, "/tasks?completed=true&limit=1&offset=2", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotFilter.Completed == nil || !*gotFilter.Completed || gotFilter.Limit != 1 || gotFilter.Offset != 2 {
		t.Errorf("store got filter %+v", gotFilter)
	}

	page := decodeBody[models.TaskPage](t, rec)
	if page.Total != 42 || page.Limit != 1 || page.Offset != 2 || len(page.Items) != 1 || page.Items[0].ID != 3 {
		t.Errorf("page = %+v", page)
	}
}

func TestListTasks_BadQueryDoesNotHitStore(t *testing.T) {
	rec := doRequest(t, &fakeStore{}, http.MethodGet, "/tasks?limit=1000", "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
