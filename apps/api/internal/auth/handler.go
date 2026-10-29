package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/mAbduqayum/fullmono/apps/api/internal/httpx"
)

const (
	cookieName = "admin_token"
	// cookiePath sends the cookie with the admin api's requests and no others.
	cookiePath   = "/api/admin"
	maxBodyBytes = 1 << 10
)

type session struct {
	Username string `json:"username"`
}

type handler struct {
	admins Admins
	tokens *Tokens
}

// NewHandler serves /api/admin/session: POST logs in, GET says who's logged
// in, DELETE logs out.
func NewHandler(admins Admins, tokens *Tokens) http.Handler {
	h := &handler{admins: admins, tokens: tokens}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/admin/session", h.logIn)
	mux.Handle("GET /api/admin/session", RequireAdmin(tokens)(http.HandlerFunc(h.whoAmI)))
	mux.HandleFunc("DELETE /api/admin/session", h.logOut)
	return mux
}

func (h *handler) logIn(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, `expected a JSON body like {"username": "…", "password": "…"}`)
		return
	}
	hash, err := h.admins.PasswordHash(r.Context(), body.Username)
	known := err == nil
	if !known && !errors.Is(err, ErrNoAdmin) {
		h.fail(w, r, err)
		return
	}
	if !known {
		// Check a stand-in hash anyway: answering an unknown name faster
		// would tell which names are admins.
		hash = unknownAdminHash()
	}
	if !passwordMatches(hash, body.Password) || !known {
		slog.WarnContext(r.Context(), "admin login failed", "username", body.Username)
		// One answer for both, for the same reason.
		httpx.Error(w, http.StatusUnauthorized, "wrong username or password")
		return
	}
	token, expires, err := h.tokens.Issue(body.Username)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	http.SetCookie(w, tokenCookie(r, token, expires))
	httpx.JSON(w, http.StatusOK, session{Username: body.Username})
}

func (h *handler) whoAmI(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, session{Username: Admin(r.Context())})
}

// logOut drops the cookie. The token itself stays valid until it expires: the
// api checks a JWT rather than looking it up, so there's nothing to delete.
func (h *handler) logOut(w http.ResponseWriter, r *http.Request) {
	cookie := tokenCookie(r, "", time.Time{})
	cookie.MaxAge = -1
	http.SetCookie(w, cookie)
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "err", err)
	httpx.Error(w, http.StatusInternalServerError, "something went wrong")
}

// tokenCookie keeps the token where the page's scripts can't read it
// (HttpOnly), that the browser never sends with another site's requests
// (SameSite=Strict), and, behind HTTPS, off plain HTTP (Secure).
func tokenCookie(r *http.Request, token string, expires time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     cookiePath,
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   overHTTPS(r),
	}
}

// overHTTPS says whether the visitor used HTTPS. Caddy ends TLS and says so in
// X-Forwarded-Proto. Locally it's plain HTTP, where a Secure cookie would be
// dropped (the e2e tests' browser visits http://web).
func overHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

var unknownAdminHash = sync.OnceValue(func() []byte {
	// A short constant password can't fail to hash.
	hash, _ := bcrypt.GenerateFromPassword([]byte("no admin has this password"), passwordCost)
	return hash
})

type adminKey struct{}

// RequireAdmin lets a request through only with a valid login token, and tells
// the handler whose it is (Admin). Anything else gets a 401.
func RequireAdmin(tokens *Tokens) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "log in first")
				return
			}
			username, err := tokens.Check(cookie.Value)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "log in again")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), adminKey{}, username)))
		})
	}
}

// Admin is the admin RequireAdmin let through.
func Admin(ctx context.Context) string {
	username, _ := ctx.Value(adminKey{}).(string)
	return username
}
