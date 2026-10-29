package links

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// Unit tests: the handler alone, through httptest, with no network or database.
// They run in milliseconds, so there can be many, and every commit runs them.

// memoryStore is a fake Store in a map, so the handler's tests need no database.
// PostgresStore has tests of its own, against a real one (store_test.go).
type memoryStore struct {
	mu    sync.Mutex
	links map[string]*Link
}

func newMemoryStore() *memoryStore {
	return &memoryStore{links: map[string]*Link{}}
}

func (s *memoryStore) Create(_ context.Context, code, url string) (Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, taken := s.links[code]; taken {
		return Link{}, ErrCodeTaken
	}
	link := &Link{Code: code, URL: url, CreatedAt: time.Now()}
	s.links[code] = link
	return *link, nil
}

func (s *memoryStore) List(_ context.Context, limit int) ([]Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	links := []Link{}
	for _, link := range s.links {
		if len(links) == limit {
			break
		}
		links = append(links, *link)
	}
	return links, nil
}

func (s *memoryStore) Visit(_ context.Context, code string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	link, ok := s.links[code]
	if !ok {
		return "", ErrNotFound
	}
	link.Clicks++
	return link.URL, nil
}

func (s *memoryStore) Ping(context.Context) error { return nil }

// newestFirst is the links in PostgresStore's order; the caller holds mu.
func (s *memoryStore) newestFirst() []Link {
	links := []Link{}
	for _, link := range s.links {
		links = append(links, *link)
	}
	slices.SortFunc(links, func(a, b Link) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return links
}

func (s *memoryStore) Search(_ context.Context, query string, limit, offset int) ([]Link, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	query = strings.ToLower(query)
	matches := []Link{}
	for _, link := range s.newestFirst() {
		if strings.Contains(strings.ToLower(link.Code), query) || strings.Contains(strings.ToLower(link.URL), query) {
			matches = append(matches, link)
		}
	}
	total := len(matches)
	return matches[min(offset, total):min(offset+limit, total)], total, nil
}

func (s *memoryStore) Delete(_ context.Context, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.links[code]; !ok {
		return ErrNotFound
	}
	delete(s.links, code)
	return nil
}

func (s *memoryStore) Stats(_ context.Context, top int) (Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stats := Stats{Top: []Link{}}
	for _, link := range s.links {
		stats.Links++
		stats.Clicks += link.Clicks
		if link.Clicks > 0 {
			stats.Top = append(stats.Top, *link)
		}
	}
	slices.SortFunc(stats.Top, func(a, b Link) int { return int(b.Clicks - a.Clicks) })
	stats.Top = stats.Top[:min(top, len(stats.Top))]
	return stats, nil
}

// allow and deny stand in for auth.RequireAdmin: these tests are about the
// links, and auth's own tests cover the login.
func allow(next http.Handler) http.Handler { return next }

func deny(http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
}

func serve(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body)))
	return rec
}

func TestCreateThenRedirect(t *testing.T) {
	store := newMemoryStore()
	h := NewHandler(store, allow)

	rec := serve(t, h, http.MethodPost, "/api/links", `{"url": "https://go.dev/doc/"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: status %d, body %s", rec.Code, rec.Body)
	}
	var link Link
	if err := json.NewDecoder(rec.Body).Decode(&link); err != nil {
		t.Fatal(err)
	}
	if len(link.Code) != codeLength {
		t.Errorf("code %q: want %d characters", link.Code, codeLength)
	}

	rec = serve(t, h, http.MethodGet, "/r/"+link.Code, "")
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "https://go.dev/doc/" {
		t.Fatalf("redirect: status %d, location %q", rec.Code, rec.Header().Get("Location"))
	}
	if clicks := store.links[link.Code].Clicks; clicks != 1 {
		t.Errorf("clicks = %d, want 1", clicks)
	}
}

// Which URLs are bad is TestCheckURL's job: this checks the handler answers 400
// for a body it can't read and for a URL checkURL rejects.
func TestCreateRejectsBadInput(t *testing.T) {
	h := NewHandler(newMemoryStore(), allow)
	for _, body := range []string{`not json`, `{"url": "go.dev"}`} {
		if rec := serve(t, h, http.MethodPost, "/api/links", body); rec.Code != http.StatusBadRequest {
			t.Errorf("%.40s: status %d, want 400", body, rec.Code)
		}
	}
}

func TestCreateRetriesATakenCode(t *testing.T) {
	store := newMemoryStore()
	codes := []string{"taken00", "taken00", "fresh00"}
	h := &handler{store: store, newCode: func() string {
		code := codes[0]
		codes = codes[1:]
		return code
	}}
	if _, err := store.Create(t.Context(), "taken00", "https://example.com"); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/links", strings.NewReader(`{"url": "https://go.dev"}`))
	h.create(rec, req)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), "fresh00") {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
}

func TestUnknownCodeIsNotFound(t *testing.T) {
	if rec := serve(t, NewHandler(newMemoryStore(), allow), http.MethodGet, "/r/nope", ""); rec.Code != http.StatusNotFound {
		t.Errorf("status %d, want 404", rec.Code)
	}
}

func TestHealth(t *testing.T) {
	if rec := serve(t, NewHandler(newMemoryStore(), allow), http.MethodGet, "/api/health", ""); rec.Code != http.StatusOK {
		t.Errorf("status %d, want 200", rec.Code)
	}
}

func TestAdminRoutesNeedALogin(t *testing.T) {
	h := NewHandler(newMemoryStore(), deny)
	for _, route := range []string{"GET /api/admin/links", "DELETE /api/admin/links/abc1234", "GET /api/admin/stats"} {
		method, path, _ := strings.Cut(route, " ")
		if rec := serve(t, h, method, path, ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: status %d, want 401", route, rec.Code)
		}
	}
	// The visitors' routes stay open.
	if rec := serve(t, h, http.MethodGet, "/api/links", ""); rec.Code != http.StatusOK {
		t.Errorf("GET /api/links: status %d, want 200", rec.Code)
	}
}

func TestAdminSearchPagesAndFilters(t *testing.T) {
	store := newMemoryStore()
	for i := range 30 {
		if _, err := store.Create(t.Context(), fmt.Sprintf("code%03d", i), fmt.Sprintf("https://example.com/%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	h := NewHandler(store, allow)
	for query, want := range map[string]struct{ links, total int }{
		"":                   {adminPageSize, 30},
		"?page=2":            {30 - adminPageSize, 30},
		"?q=CODE01":          {10, 10},
		"?q=example.com/29":  {1, 1},
		"?q=nothing+like+it": {0, 0},
	} {
		rec := serve(t, h, http.MethodGet, "/api/admin/links"+query, "")
		var page searchPage
		if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK || len(page.Links) != want.links || page.Total != want.total {
			t.Errorf("%q: status %d, %d links of %d; want %d of %d", query, rec.Code, len(page.Links), page.Total, want.links, want.total)
		}
	}
	for _, query := range []string{"?page=0", "?page=two"} {
		if rec := serve(t, h, http.MethodGet, "/api/admin/links"+query, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("%q: status %d, want 400", query, rec.Code)
		}
	}
}

func TestAdminDeletes(t *testing.T) {
	store := newMemoryStore()
	if _, err := store.Create(t.Context(), "spam000", "https://example.com/spam"); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(store, allow)
	if rec := serve(t, h, http.MethodDelete, "/api/admin/links/spam000", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: status %d, want 204", rec.Code)
	}
	if rec := serve(t, h, http.MethodGet, "/r/spam000", ""); rec.Code != http.StatusNotFound {
		t.Errorf("redirect after delete: status %d, want 404", rec.Code)
	}
	if rec := serve(t, h, http.MethodDelete, "/api/admin/links/spam000", ""); rec.Code != http.StatusNotFound {
		t.Errorf("second delete: status %d, want 404", rec.Code)
	}
}

func TestAdminStats(t *testing.T) {
	store := newMemoryStore()
	for code, visits := range map[string]int{"often00": 2, "once000": 1, "never00": 0} {
		if _, err := store.Create(t.Context(), code, "https://example.com/"+code); err != nil {
			t.Fatal(err)
		}
		for range visits {
			if _, err := store.Visit(t.Context(), code); err != nil {
				t.Fatal(err)
			}
		}
	}
	rec := serve(t, NewHandler(store, allow), http.MethodGet, "/api/admin/stats", "")
	var stats Stats
	if err := json.NewDecoder(rec.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}
	top := []string{}
	for _, link := range stats.Top {
		top = append(top, link.Code)
	}
	if stats.Links != 3 || stats.Clicks != 3 || !slices.Equal(top, []string{"often00", "once000"}) {
		t.Errorf("stats %d links, %d clicks, top %v; want 3, 3, [often00 once000]", stats.Links, stats.Clicks, top)
	}
}
