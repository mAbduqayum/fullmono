package links

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"

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
)

type handler struct {
	store   Store
	newCode func() string
}

// NewHandler serves the JSON API under /api and the redirects under /r.
func NewHandler(store Store) http.Handler {
	h := &handler{store: store, newCode: randomCode}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", h.health)
	mux.HandleFunc("GET /api/links", h.list)
	mux.HandleFunc("POST /api/links", h.create)
	mux.HandleFunc("GET /r/{code}", h.redirect)
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
