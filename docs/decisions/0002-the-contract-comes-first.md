# 0002. The routes are written in openapi.yaml, and code made from it

Status: accepted

## Context

The api and its client must agree on every route: its path, parameters, body
and answers. Written twice, once in Go and once in TypeScript, the two drift,
and the drift shows up at runtime. The ways to have one source:

- **Code first**: annotate the Go handlers and generate the spec from them.
  The spec is then whatever the code happens to do, and the client waits for
  the api.
- **One language on both sides**, as tRPC does with TypeScript: not an option
  with a Go api.
- **gRPC or Connect**: typed, but a browser can't speak gRPC itself, and plain
  JSON over HTTP is easier to read and to curl.
- **Contract first**: write the routes in an OpenAPI file, by hand, and
  generate both sides from it.

## Decision

Contract first. [openapi.yaml](../../apps/api/openapi.yaml) is the source;
`just gen` makes the api's router, models and a typed interface per tag
(oapi-codegen's strict server) and the web app's types (openapi-typescript),
which its client, openapi-fetch, checks every call against. The api also
checks every request against the contract before a handler runs, so the
contract decides what's refused, logins included.

## Consequences

- A route changes in the YAML first; then the compiler, in both apps, points
  at every place that must follow.
- A handler gets only requests that fit, so it doesn't re-check them.
- CI fails if the generated code doesn't match the contract.
- The YAML is long, and the generators decide what can be expressed: a feature
  of OpenAPI they don't support is one we can't use.

## Revisit when

The api has clients we don't control: then the contract needs versions, and
changes that never break an old client.
