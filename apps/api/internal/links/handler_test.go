package links

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// memoryStore is a Store in a map, so the handler's tests need no database.
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

func serve(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body)))
	return rec
}

func TestCreateThenRedirect(t *testing.T) {
	store := newMemoryStore()
	h := NewHandler(store)

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

func TestCreateRejectsBadURLs(t *testing.T) {
	h := NewHandler(newMemoryStore())
	for _, body := range []string{
		`not json`,
		`{"url": ""}`,
		`{"url": "go.dev"}`,
		`{"url": "javascript:alert(1)"}`,
		`{"url": "ftp://example.com/file"}`,
		`{"url": "https://` + strings.Repeat("a", maxURLLength) + `.com"}`,
	} {
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
	if rec := serve(t, NewHandler(newMemoryStore()), http.MethodGet, "/r/nope", ""); rec.Code != http.StatusNotFound {
		t.Errorf("status %d, want 404", rec.Code)
	}
}

func TestHealth(t *testing.T) {
	if rec := serve(t, NewHandler(newMemoryStore()), http.MethodGet, "/api/health", ""); rec.Code != http.StatusOK {
		t.Errorf("status %d, want 200", rec.Code)
	}
}
