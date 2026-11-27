# 0004. An admin's login is a JWT in an httpOnly cookie

Status: accepted

## Context

After an admin logs in, each admin request must say who sent it. The usual
ways:

- **A session table**: a random id in a cookie, looked up in Postgres on every
  request. Logging out, or out everywhere, deletes rows.
- **A signed token (JWT)**: the admin's name and an expiry, signed with a
  secret, checked without a lookup. Nothing to delete, so nothing to revoke.

And where the browser keeps it: `localStorage` is readable by any script on
the page, an injected one too; an httpOnly cookie isn't.

## Decision

A JWT, HS256, signed with `JWT_SECRET` and valid for 8 hours, in a cookie
that's `HttpOnly`, `SameSite=Strict`, `Secure` behind HTTPS, and sent only
with `/api/admin` requests ([token.go](../../apps/api/internal/auth/token.go),
[handler.go](../../apps/api/internal/auth/handler.go)).

## Consequences

- An admin request costs no database lookup, and the api keeps no state.
- No script on the page can read the token, and no other site's form or
  request carries the cookie, which rules out CSRF too.
- Logging out drops the cookie, but a copied token stays valid until it
  expires. Nothing can revoke one admin's login: changing `JWT_SECRET` logs
  every admin out.

## Revisit when

There are more admins than a few, roles, or a need to end one login, such as
a stolen laptop: a session table can.
