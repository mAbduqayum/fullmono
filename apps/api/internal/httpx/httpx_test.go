package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// logTo sends the default logger's lines to a buffer, as LogHandler wraps
// the api's own, until the test ends.
func logTo(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(LogHandler{Handler: slog.NewJSONHandler(&buf, nil)}))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

// lines reads the JSON lines logged.
func lines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var all []map[string]any
	for line := range strings.Lines(buf.String()) {
		var fields map[string]any
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			t.Fatalf("log line %q: %v", line, err)
		}
		all = append(all, fields)
	}
	return all
}

func TestEveryLineOfARequestNamesIt(t *testing.T) {
	buf := logTo(t)
	h := RequestID(LogRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Fail(w, r, errors.New("the database is gone"))
	})))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/links", nil)
	// A visitor's own id is ignored.
	req.Header.Set("X-Request-Id", "forged")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	id := rec.Header().Get("X-Request-Id")
	if id == "" || id == "forged" {
		t.Fatalf("X-Request-Id %q", id)
	}
	logged := lines(t, buf)
	if len(logged) != 2 {
		t.Fatalf("want the failure and the request logged, got %v", logged)
	}
	for _, line := range logged {
		if line["request_id"] != id {
			t.Errorf("line %v: want request_id %q", line, id)
		}
	}
	// The answer names the request, never the error.
	body := rec.Body.String()
	if !strings.Contains(body, id) || strings.Contains(body, "database") {
		t.Errorf("body %s", body)
	}
}

func TestEachRequestGetsANewID(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	ids := map[string]bool{}
	for range 3 {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
		ids[rec.Header().Get("X-Request-Id")] = true
	}
	if len(ids) != 3 {
		t.Errorf("ids %v", ids)
	}
}

func TestALineOutsideARequestHasNoID(t *testing.T) {
	buf := logTo(t)
	slog.InfoContext(t.Context(), "listening")
	if line := lines(t, buf)[0]; line["request_id"] != nil {
		t.Errorf("line %v", line)
	}
}
