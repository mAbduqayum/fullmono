# 0003. Caddy serves the built app and the api from one origin

Status: accepted

## Context

The browser needs the web app's files and the api. If they came from two
origins (`app.example.com` and `api.example.com`), every api call would be
cross-origin: the api would need CORS headers, and the login cookie would have
to cross sites, which `SameSite=Strict` forbids.

The pages are a form, a list and an admin panel: nothing a search engine
needs to read, so they needn't be rendered on a server (SSR, as Next.js does).
A single-page app built to static files needs no server process of its own.

## Decision

[Caddy](../../apps/web/Caddyfile) is the only door: it serves the built React
app and its hashed assets, and passes `/api/*` and `/r/*` to the api, which
has no port of its own outside the stack. Caddy also gets and renews the HTTPS
certificate, and sets the security headers.

## Consequences

- No CORS, and the cookie is first-party and `SameSite=Strict`.
- HTTPS takes one line, the domain, with nothing to renew by hand.
- The app's files are cached forever (their names change with their content),
  and serving them costs Caddy nothing.
- The first page waits for the JavaScript to load before it shows anything.
- The api trusts Caddy's `X-Forwarded-*` headers (HTTPS, the visitor's
  address), which holds only while nothing else can reach it.

## Revisit when

Pages need to be read before their JavaScript runs (search engines, link
previews): then prerendering, or SSR. Or a CDN goes in front, which then sets
the forwarded headers.
