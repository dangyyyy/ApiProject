package middleware

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func errorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return body["error"]
}

func TestRecover(t *testing.T) {
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	rec := serve(Recover(panicking), httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if got := errorMessage(t, rec); got != "internal server error" {
		t.Errorf("error = %q, want %q", got, "internal server error")
	}
}

func TestRecover_AbortHandlerIsRepanicked(t *testing.T) {
	aborting := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	})

	defer func() {
		if rec := recover(); rec != http.ErrAbortHandler {
			t.Fatalf("recovered %v, want http.ErrAbortHandler", rec)
		}
	}()

	serve(Recover(aborting), httptest.NewRequest(http.MethodGet, "/", nil))
	t.Fatal("expected panic to propagate")
}

func TestJSONErrors(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
		wantBody   string
	}{
		{
			name: "plain text 404 becomes json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			},
			wantStatus: http.StatusNotFound,
			wantBody:   `{"error":"not found"}`,
		},
		{
			name: "plain text 405 becomes json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "x", http.StatusMethodNotAllowed)
			},
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   `{"error":"method not allowed"}`,
		},
		{
			name: "own json 404 is untouched",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error":"task not found"}`))
			},
			wantStatus: http.StatusNotFound,
			wantBody:   `{"error":"task not found"}`,
		},
		{
			name: "plain text 400 is untouched",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad", http.StatusBadRequest)
			},
			wantStatus: http.StatusBadRequest,
			wantBody:   "bad",
		},
		{
			name: "200 body passes through",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("hello"))
			},
			wantStatus: http.StatusOK,
			wantBody:   "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(JSONErrors(tt.handler), httptest.NewRequest(http.MethodGet, "/", nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func TestJSONErrors_SetsJSONContentType(t *testing.T) {
	notFound := http.HandlerFunc(http.NotFound)

	rec := serve(JSONErrors(notFound), httptest.NewRequest(http.MethodGet, "/", nil))

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}
}

func TestRequestID(t *testing.T) {
	tests := []struct {
		name     string
		incoming string
		wantID   string
	}{
		{name: "generated when missing"},
		{name: "incoming is reused", incoming: "abc", wantID: "abc"},
		{name: "too long is replaced", incoming: strings.Repeat("x", 65)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var idInHandler string
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				idInHandler = RequestIDFromContext(r.Context())
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.incoming != "" {
				req.Header.Set(requestIDHeader, tt.incoming)
			}

			rec := serve(RequestID(inner), req)
			idInResponse := rec.Header().Get(requestIDHeader)

			if idInResponse == "" {
				t.Fatal("response has no request id header")
			}
			if idInResponse != idInHandler {
				t.Errorf("id in response %q != id in handler %q", idInResponse, idInHandler)
			}
			if tt.wantID != "" && idInResponse != tt.wantID {
				t.Errorf("id = %q, want %q", idInResponse, tt.wantID)
			}
			if tt.incoming != "" && tt.wantID == "" && idInResponse == tt.incoming {
				t.Errorf("incoming id %q should have been replaced", tt.incoming)
			}
		})
	}
}

func TestLogging_RecordsStatusAndBytes(t *testing.T) {
	var gotRecorder *statusRecorder
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRecorder = w.(*statusRecorder)
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("12345"))
	})

	rec := serve(Logging(inner), httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}
	if gotRecorder.status != http.StatusTeapot {
		t.Errorf("recorded status = %d, want %d", gotRecorder.status, http.StatusTeapot)
	}
	if gotRecorder.bytes != 5 {
		t.Errorf("recorded bytes = %d, want 5", gotRecorder.bytes)
	}
}

func TestChain_Order(t *testing.T) {
	var calls []string
	mark := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "h")
	})

	serve(Chain(final, mark("A"), mark("B")), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []string{"A", "B", "h"}
	if !slices.Equal(calls, want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}
