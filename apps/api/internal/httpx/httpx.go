// Package httpx holds what every handler of the api shares: its JSON answers,
// its failures and its request log.
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// JSON answers with v as the JSON body.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error answers {"error": message}, the shape of every failed request.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}

// BadRequest answers a request a generated router couldn't read.
func BadRequest(w http.ResponseWriter, _ *http.Request, err error) {
	Error(w, http.StatusBadRequest, err.Error())
}

// Fail logs what went wrong on the api's side, and answers without the
// details, which are for whoever reads the logs, not for the visitor.
func Fail(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "err", err)
	Error(w, http.StatusInternalServerError, "something went wrong")
}

// LogRequests logs one JSON line per request, which `docker compose logs` and
// Dozzle show; not the health checks, which Docker makes every few seconds.
func LogRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.InfoContext(r.Context(), "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
