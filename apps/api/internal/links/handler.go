package links

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mAbduqayum/fullmono/apps/api/internal/auth"
	"github.com/mAbduqayum/fullmono/apps/api/internal/httpx"
)

const (
	// codeLength characters of codeAlphabet give 62^7, about 3.5 trillion, codes.
	codeLength   = 7
	codeAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	// createAttempts is how many random codes a create tries before giving up.
	createAttempts = 3
	maxURLLength   = 2048
	maxBodyBytes   = 4 << 10
	listLimit      = 50
	adminPageSize  = 25
	// maxAdminPage is far past any real list, and short of overflowing the offset.
	maxAdminPage = 1 << 20
	statsTop     = 5
)

type handler struct {
	store   Store
	newCode func() string
}

// NewHandler serves the JSON API under /api and the redirects under /r, and
// the admin routes under /api/admin to the requests requireAdmin lets through.
func NewHandler(store Store, requireAdmin func(http.Handler) http.Handler) http.Handler {
	h := &handler{store: store, newCode: randomCode}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", h.health)
	mux.HandleFunc("GET /api/links", h.list)
	mux.HandleFunc("POST /api/links", h.create)
	mux.HandleFunc("GET /r/{code}", h.redirect)

	admin := func(pattern string, handle http.HandlerFunc) {
		mux.Handle(pattern, requireAdmin(handle))
	}
	admin("GET /api/admin/links", h.search)
	admin("DELETE /api/admin/links/{code}", h.delete)
	admin("GET /api/admin/stats", h.stats)
	return mux
}

func (h *handler) health(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Ping(r.Context()); err != nil {
		httpx.Error(w, http.StatusServiceUnavailable, "database unreachable")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	links, err := h.store.List(r.Context(), listLimit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, links)
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "expected a JSON body like {\"url\": \"https://…\"}")
		return
	}
	if problem := checkURL(body.URL); problem != "" {
		httpx.Error(w, http.StatusBadRequest, problem)
		return
	}
	for range createAttempts {
		link, err := h.store.Create(r.Context(), h.newCode(), body.URL)
		if errors.Is(err, ErrCodeTaken) {
			continue
		}
		if err != nil {
			h.fail(w, r, err)
			return
		}
		httpx.JSON(w, http.StatusCreated, link)
		return
	}
	h.fail(w, r, ErrCodeTaken)
}

func (h *handler) redirect(w http.ResponseWriter, r *http.Request) {
	target, err := h.store.Visit(r.Context(), r.PathValue("code"))
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	// 302, not 301: browsers cache a permanent redirect and stop asking, so visits would go uncounted.
	http.Redirect(w, r, target, http.StatusFound)
}

// searchPage is one page of an admin search, with what paging through it needs.
type searchPage struct {
	Links    []Link `json:"links"`
	Total    int    `json:"total"`
	PageSize int    `json:"pageSize"`
}

func (h *handler) search(w http.ResponseWriter, r *http.Request) {
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxAdminPage {
			httpx.Error(w, http.StatusBadRequest, "page must be a whole number from 1")
			return
		}
		page = n
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) > maxURLLength {
		httpx.Error(w, http.StatusBadRequest, "the search is too long")
		return
	}
	links, total, err := h.store.Search(r.Context(), query, adminPageSize, (page-1)*adminPageSize)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, searchPage{Links: links, Total: total, PageSize: adminPageSize})
}

func (h *handler) delete(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	err := h.store.Delete(r.Context(), code)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "no link has that code")
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	// Who deleted what, for whoever asks later.
	slog.InfoContext(r.Context(), "link deleted", "code", code, "admin", auth.Admin(r.Context()))
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.Stats(r.Context(), statsTop)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, stats)
}

func (h *handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "err", err)
	httpx.Error(w, http.StatusInternalServerError, "something went wrong")
}

// checkURL says what's wrong with a URL to shorten, or "" when nothing is.
func checkURL(raw string) string {
	if len(raw) > maxURLLength {
		return "the URL is too long"
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "the URL must start with http:// or https://"
	}
	return ""
}

func randomCode() string {
	code := make([]byte, codeLength)
	alphabetSize := big.NewInt(int64(len(codeAlphabet)))
	for i := range code {
		// crypto/rand's reader never fails (Go 1.24+), so neither does Int.
		n, _ := rand.Int(rand.Reader, alphabetSize)
		code[i] = codeAlphabet[n.Int64()]
	}
	return string(code)
}
