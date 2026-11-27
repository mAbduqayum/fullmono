// Package httpx holds what every handler of the api shares: its JSON answers,
// its failures, its request ids and its request log.
package httpx

import (
	"context"
	"crypto/rand"
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
// details, which are for whoever reads the logs, not for the visitor. The
// answer names the request, so a visitor's report leads to its log lines.
func Fail(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "err", err)
	message := "something went wrong"
	if id := RequestIDOf(r.Context()); id != "" {
		message += " (request " + id + ")"
	}
	Error(w, http.StatusInternalServerError, message)
}

type requestIDKey struct{}

// RequestID gives each request an id of its own: in its context, where each
// line it logs picks it up (LogHandler), and in the answer's X-Request-Id
// header. One the request arrives with is ignored: a visitor can send any.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := rand.Text()
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// RequestIDOf is the id RequestID gave a request, or "" outside one.
func RequestIDOf(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// LogHandler adds the request's id to each line logged with its context
// (slog.InfoContext and the like), so one request's lines can be found among
// everyone's.
type LogHandler struct {
	slog.Handler
}

func (h LogHandler) Handle(ctx context.Context, record slog.Record) error {
	if id := RequestIDOf(ctx); id != "" {
		record.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, record)
}

func (h LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return LogHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h LogHandler) WithGroup(name string) slog.Handler {
	return LogHandler{Handler: h.Handler.WithGroup(name)}
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
