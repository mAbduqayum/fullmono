// Package api is the api as a whole: its contract, openapi.yaml, and the
// handler that serves it, checking every request against the contract first.
package api

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/mAbduqayum/fullmono/apps/api/internal/auth"
	"github.com/mAbduqayum/fullmono/apps/api/internal/httpx"
	"github.com/mAbduqayum/fullmono/apps/api/internal/links"
)

//go:embed openapi.yaml
var contract []byte

// maxBodyBytes bounds a request's body, which the check reads whole. The
// largest the contract takes, a link to shorten, is a 2048-character URL.
const maxBodyBytes = 4 << 10

// NewHandler serves the operations openapi.yaml describes. A request the
// contract doesn't allow never reaches a handler: a route it doesn't have, a
// parameter or body of the wrong shape, or an admin's operation without a
// login gets its 4xx here.
func NewHandler(store links.Store, admins auth.Admins, tokens *auth.Tokens) (http.Handler, error) {
	spec, err := openapi3.NewLoader().LoadFromData(contract)
	if err != nil {
		return nil, fmt.Errorf("load openapi.yaml: %w", err)
	}
	if err := spec.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("openapi.yaml: %w", err)
	}
	check := nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		Options:              openapi3filter.Options{AuthenticationFunc: authenticate},
		ErrorHandlerWithOpts: refuse,
	})

	mux := http.NewServeMux()
	mux.Handle("/api/admin/session", auth.NewHandler(admins, tokens))
	mux.Handle("/", links.NewHandler(store))
	// Identify first, so that the check knows who's logged in.
	return http.MaxBytesHandler(auth.Identify(tokens)(check(mux)), maxBodyBytes), nil
}

// authenticate is the check behind each security scheme openapi.yaml
// declares, which the validator runs on the operations that name it.
func authenticate(_ context.Context, input *openapi3filter.AuthenticationInput) error {
	if input.SecuritySchemeName != "cookieAuth" {
		return fmt.Errorf("no check for the security scheme %q", input.SecuritySchemeName)
	}
	return auth.CheckLogin(input.RequestValidationInput.Request)
}

// refuse answers a request the contract doesn't allow, in the api's
// {"error": …} shape: the validator's own answer is plain text.
func refuse(_ context.Context, err error, w http.ResponseWriter, r *http.Request, _ nethttpmiddleware.ErrorHandlerOpts) {
	var login *openapi3filter.SecurityRequirementsError
	var request *openapi3filter.RequestError
	switch {
	case errors.As(err, &login) && len(login.Errors) > 0:
		httpx.Error(w, http.StatusUnauthorized, login.Errors[0].Error())
	case errors.As(err, &request):
		// The first line says what's wrong; the rest quotes the schema.
		message, _, _ := strings.Cut(request.Error(), "\n")
		httpx.Error(w, http.StatusBadRequest, message)
	case errors.Is(err, routers.ErrMethodNotAllowed):
		httpx.Error(w, http.StatusMethodNotAllowed, "the route doesn't take that method")
	case errors.Is(err, routers.ErrPathNotFound):
		httpx.Error(w, http.StatusNotFound, "the api has no such route")
	default:
		httpx.Fail(w, r, err)
	}
}
