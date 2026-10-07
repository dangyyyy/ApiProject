package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
)

type jsonErrorWriter struct {
	http.ResponseWriter
	intercepted bool
}

func (w *jsonErrorWriter) WriteHeader(code int) {
	isRouterError := code == http.StatusNotFound || code == http.StatusMethodNotAllowed
	isPlainText := strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain")

	if !isRouterError || !isPlainText {
		w.ResponseWriter.WriteHeader(code)
		return
	}
	w.intercepted = true
	w.Header().Set("Content-Type", "application/json")
	w.ResponseWriter.WriteHeader(code)
	message := strings.ToLower(http.StatusText(code))
	_ = json.NewEncoder(w.ResponseWriter).Encode(map[string]string{"error": message})
}

func (w *jsonErrorWriter) Write(b []byte) (int, error) {
	if w.intercepted {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}
func (w *jsonErrorWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
func JSONErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&jsonErrorWriter{ResponseWriter: w}, r)
	})
}
