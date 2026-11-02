package links

//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../openapi.yaml

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
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
	listLimit      = 50
	adminPageSize  = 25
	statsTop       = 5
	noSuchLink     = "no link has that code"
)

// server implements the operations openapi.yaml tags links, as the interface
// generated from it (api.gen.go) declares them: a typed request in, a typed
// answer out, and an error only for a failure, which answers 500.
type server struct {
	store   Store
	newCode func() string
}

var _ StrictServerInterface = (*server)(nil)

// NewHandler serves the operations openapi.yaml tags links. It trusts the
// requests to have been checked against the contract first (api.NewHandler
// does): the admin's operations to come with a login, and the parameters and
// bodies to have their shapes and limits.
func NewHandler(store Store) http.Handler {
	strict := NewStrictHandlerWithOptions(&server{store: store, newCode: randomCode}, nil, StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  httpx.BadRequest,
		ResponseErrorHandlerFunc: httpx.Fail,
	})
	return HandlerWithOptions(strict, StdHTTPServerOptions{ErrorHandlerFunc: httpx.BadRequest})
}

func (s *server) GetHealth(ctx context.Context, _ GetHealthRequestObject) (GetHealthResponseObject, error) {
	if err := s.store.Ping(ctx); err != nil {
		return GetHealth503JSONResponse{Error: "database unreachable"}, nil
	}
	return GetHealth200JSONResponse{Status: "ok"}, nil
}

func (s *server) ListLinks(ctx context.Context, _ ListLinksRequestObject) (ListLinksResponseObject, error) {
	links, err := s.store.List(ctx, listLimit)
	if err != nil {
		return nil, err
	}
	return ListLinks200JSONResponse(links), nil
}

func (s *server) CreateLink(ctx context.Context, request CreateLinkRequestObject) (CreateLinkResponseObject, error) {
	target := request.Body.URL
	if problem := checkURL(target); problem != "" {
		return CreateLink400JSONResponse{Error: problem}, nil
	}
	for range createAttempts {
		link, err := s.store.Create(ctx, s.newCode(), target)
		if errors.Is(err, ErrCodeTaken) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return CreateLink201JSONResponse(link), nil
	}
	return nil, ErrCodeTaken
}

func (s *server) FollowLink(ctx context.Context, request FollowLinkRequestObject) (FollowLinkResponseObject, error) {
	target, err := s.store.Visit(ctx, request.Code)
	if errors.Is(err, ErrNotFound) {
		return FollowLink404TextResponse(noSuchLink + "\n"), nil
	}
	if err != nil {
		return nil, err
	}
	return FollowLink302Response{Headers: FollowLink302ResponseHeaders{Location: target}}, nil
}

func (s *server) SearchLinks(ctx context.Context, request SearchLinksRequestObject) (SearchLinksResponseObject, error) {
	page, query := 1, ""
	if request.Params.Page != nil {
		page = *request.Params.Page
	}
	if request.Params.Q != nil {
		query = strings.TrimSpace(*request.Params.Q)
	}
	links, total, err := s.store.Search(ctx, query, adminPageSize, (page-1)*adminPageSize)
	if err != nil {
		return nil, err
	}
	return SearchLinks200JSONResponse{Links: links, Total: total, PageSize: adminPageSize}, nil
}

func (s *server) DeleteLink(ctx context.Context, request DeleteLinkRequestObject) (DeleteLinkResponseObject, error) {
	err := s.store.Delete(ctx, request.Code)
	if errors.Is(err, ErrNotFound) {
		return DeleteLink404JSONResponse{Error: noSuchLink}, nil
	}
	if err != nil {
		return nil, err
	}
	// Who deleted what, for whoever asks later.
	slog.InfoContext(ctx, "link deleted", "code", request.Code, "admin", auth.Admin(ctx))
	return DeleteLink204Response{}, nil
}

func (s *server) GetStats(ctx context.Context, _ GetStatsRequestObject) (GetStatsResponseObject, error) {
	stats, err := s.store.Stats(ctx, statsTop)
	if err != nil {
		return nil, err
	}
	return GetStats200JSONResponse(stats), nil
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
