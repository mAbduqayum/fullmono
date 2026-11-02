package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mAbduqayum/fullmono/apps/api"
	"github.com/mAbduqayum/fullmono/apps/api/internal/auth"
	"github.com/mAbduqayum/fullmono/apps/api/internal/links"
)

// Unit tests of what openapi.yaml turns away before a handler runs. The
// handlers' own tests are in their packages.

// noLinks lists no links. Its other methods panic, as the embedded Store is
// nil: a request the contract turns away must never reach the store.
type noLinks struct{ links.Store }

func (noLinks) List(context.Context, int) ([]links.Link, error) { return []links.Link{}, nil }

func newHandler(t *testing.T) (http.Handler, *auth.Tokens) {
	t.Helper()
	tokens, err := auth.NewTokens(strings.Repeat("k", 32))
	if err != nil {
		t.Fatal(err)
	}
	// No admins either: no test here logs in.
	h, err := api.NewHandler(noLinks{}, nil, tokens)
	if err != nil {
		t.Fatal(err)
	}
	return h, tokens
}

// send answers the route ("METHOD /path"), and returns the status and the error message.
func send(t *testing.T, h http.Handler, route, body string, edit func(*http.Request)) (int, string) {
	t.Helper()
	method, path, _ := strings.Cut(route, " ")
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if edit != nil {
		edit(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var answer struct{ Error string }
	_ = json.NewDecoder(rec.Body).Decode(&answer) // a 2xx may not be an error, or JSON
	return rec.Code, answer.Error
}

var adminRoutes = []string{
	"GET /api/admin/links",
	"DELETE /api/admin/links/abc1234",
	"GET /api/admin/stats",
	"GET /api/admin/session",
}

func TestAdminRoutesNeedALogin(t *testing.T) {
	h, _ := newHandler(t)
	forged := func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "admin_token", Value: "forged"}) }
	for _, route := range adminRoutes {
		if status, message := send(t, h, route, "", nil); status != http.StatusUnauthorized || message != "log in first" {
			t.Errorf("%s: %d %q, want 401 log in first", route, status, message)
		}
		if status, message := send(t, h, route, "", forged); status != http.StatusUnauthorized || message != "log in again" {
			t.Errorf("%s with a forged token: %d %q, want 401 log in again", route, status, message)
		}
	}
	// The visitors' routes stay open.
	if status, _ := send(t, h, "GET /api/links", "", nil); status != http.StatusOK {
		t.Errorf("GET /api/links: status %d, want 200", status)
	}
}

func TestALoginReachesTheHandler(t *testing.T) {
	h, tokens := newHandler(t)
	token, _, err := tokens.Issue("alice")
	if err != nil {
		t.Fatal(err)
	}
	withLogin := func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "admin_token", Value: token}) }
	if status, _ := send(t, h, "GET /api/admin/session", "", withLogin); status != http.StatusOK {
		t.Errorf("status %d, want 200", status)
	}
}

func TestMalformedRequestsAreRefused(t *testing.T) {
	h, tokens := newHandler(t)
	token, _, err := tokens.Issue("alice")
	if err != nil {
		t.Fatal(err)
	}
	withLogin := func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "admin_token", Value: token}) }
	for _, test := range []struct{ route, body, want string }{
		{"GET /api/admin/links?page=0", "", `parameter "page"`},
		{"GET /api/admin/links?page=two", "", `parameter "page"`},
		{"GET /api/admin/links?q=" + strings.Repeat("x", 2049), "", `parameter "q"`},
		{"POST /api/links", "not json", "request body"},
		{"POST /api/links", `{"link": "https://go.dev"}`, "request body"},
		{"POST /api/links", `{"url": "https://go.dev/` + strings.Repeat("x", 2048) + `"}`, "request body"},
		{"POST /api/links", `{"url": "` + strings.Repeat("x", 8<<10) + `"}`, "request body"},
	} {
		status, message := send(t, h, test.route, test.body, withLogin)
		if status != http.StatusBadRequest || !strings.Contains(message, test.want) {
			t.Errorf("%.50s %.30s: %d %q, want 400 about %s", test.route, test.body, status, message, test.want)
		}
	}
}

func TestUnknownRoutes(t *testing.T) {
	h, _ := newHandler(t)
	if status, message := send(t, h, "GET /api/nope", "", nil); status != http.StatusNotFound || message == "" {
		t.Errorf("GET /api/nope: %d %q, want 404", status, message)
	}
	if status, message := send(t, h, "PUT /api/links", "", nil); status != http.StatusMethodNotAllowed || message == "" {
		t.Errorf("PUT /api/links: %d %q, want 405", status, message)
	}
}
