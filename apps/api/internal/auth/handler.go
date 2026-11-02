package auth

//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../openapi.yaml

import (
	"context"
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
	cookiePath = "/api/admin"
)

// server implements the operations openapi.yaml tags auth (api.gen.go).
type server struct {
	admins Admins
	tokens *Tokens
}

var _ StrictServerInterface = (*server)(nil)

// NewHandler serves /api/admin/session: POST logs in, GET says who's logged
// in, DELETE logs out. Like links.NewHandler, it trusts the requests to have
// been checked against openapi.yaml, a GET's login included.
func NewHandler(admins Admins, tokens *Tokens) http.Handler {
	strict := NewStrictHandlerWithOptions(&server{admins: admins, tokens: tokens}, nil, StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  httpx.BadRequest,
		ResponseErrorHandlerFunc: httpx.Fail,
	})
	return HandlerWithOptions(strict, StdHTTPServerOptions{ErrorHandlerFunc: httpx.BadRequest})
}

func (s *server) LogIn(ctx context.Context, request LogInRequestObject) (LogInResponseObject, error) {
	username := request.Body.Username
	hash, err := s.admins.PasswordHash(ctx, username)
	known := err == nil
	if !known && !errors.Is(err, ErrNoAdmin) {
		return nil, err
	}
	if !known {
		// Check a stand-in hash anyway: answering an unknown name faster
		// would tell which names are admins.
		hash = unknownAdminHash()
	}
	if !passwordMatches(hash, request.Body.Password) || !known {
		slog.WarnContext(ctx, "admin login failed", "username", username)
		// One answer for both, for the same reason.
		return LogIn401JSONResponse{Error: "wrong username or password"}, nil
	}
	token, expires, err := s.tokens.Issue(username)
	if err != nil {
		return nil, err
	}
	cookie := tokenCookie(token, expires, overHTTPS(request.Params.XForwardedProto))
	return LogIn200JSONResponse{
		Body:    Session{Username: username},
		Headers: LogIn200ResponseHeaders{SetCookie: cookie.String()},
	}, nil
}

func (s *server) GetSession(ctx context.Context, _ GetSessionRequestObject) (GetSessionResponseObject, error) {
	return GetSession200JSONResponse{Username: Admin(ctx)}, nil
}

// LogOut drops the cookie. The token itself stays valid until it expires: the
// api checks a JWT rather than looking it up, so there's nothing to delete.
func (s *server) LogOut(_ context.Context, request LogOutRequestObject) (LogOutResponseObject, error) {
	cookie := tokenCookie("", time.Time{}, overHTTPS(request.Params.XForwardedProto))
	cookie.MaxAge = -1
	return LogOut204Response{Headers: LogOut204ResponseHeaders{SetCookie: cookie.String()}}, nil
}

// tokenCookie keeps the token where the page's scripts can't read it
// (HttpOnly), that the browser never sends with another site's requests
// (SameSite=Strict), and, behind HTTPS, off plain HTTP (Secure).
func tokenCookie(token string, expires time.Time, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     cookiePath,
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	}
}

// overHTTPS says whether the visitor used HTTPS. Caddy ends TLS and says so in
// X-Forwarded-Proto. Locally it's plain HTTP, where a Secure cookie would be
// dropped (the e2e tests' browser visits http://web).
func overHTTPS(forwardedProto *string) bool {
	return forwardedProto != nil && *forwardedProto == "https"
}

var unknownAdminHash = sync.OnceValue(func() []byte {
	// A short constant password can't fail to hash.
	hash, _ := bcrypt.GenerateFromPassword([]byte("no admin has this password"), passwordCost)
	return hash
})

type adminKey struct{}

// Identify tells the handlers (Admin) whose login a request's cookie holds,
// when it holds a valid one. It turns nobody away: openapi.yaml says which
// operations need a login, and CheckLogin is how the api checks it.
func Identify(tokens *Tokens) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cookie, err := r.Cookie(cookieName); err == nil {
				if username, err := tokens.Check(cookie.Value); err == nil {
					r = r.WithContext(context.WithValue(r.Context(), adminKey{}, username))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

var (
	errNoLogin      = errors.New("log in first")
	errLoginExpired = errors.New("log in again")
)

// CheckLogin says why a request Identify saw isn't logged in, or nil when it
// is: no cookie, or one with a token that's expired or not the api's.
func CheckLogin(r *http.Request) error {
	if Admin(r.Context()) != "" {
		return nil
	}
	if _, err := r.Cookie(cookieName); err != nil {
		return errNoLogin
	}
	return errLoginExpired
}

// Admin is the admin Identify found.
func Admin(ctx context.Context) string {
	username, _ := ctx.Value(adminKey{}).(string)
	return username
}
