# fullmono

A URL shortener, kept small on purpose: it's here to show the tooling around
a fullstack monorepo — mise for the tools, one `justfile`, Docker images and
compose, Caddy in front, git hooks, CI/CD on GitHub Actions and Dependabot.

- **api**: Go, `net/http` and Postgres (pgx). `POST /api/links` shortens a URL,
  `GET /api/links` lists them, `GET /r/{code}` redirects and counts the click.
  Under `/api/admin`, admins search, delete and count the links, logged in with
  a JWT in an httpOnly cookie.
- **web**: React and Vite on bun, served by Caddy, which also routes `/api` and
  `/r` to the api. The admin panel is at `/admin`.

How the parts fit together: [docs/architecture.md](docs/architecture.md). How to
change them: [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md).

## Quick start

You need `docker` and [`mise`](https://mise.jdx.dev), which installs the rest at
the versions [mise.toml](mise.toml) pins: `go`, `bun`, `just`, `lefthook` and
`golangci-lint`.

```sh
mise install      # the tools
just setup        # .env from .env.example, dependencies, git hooks
just up           # build and start everything: http://localhost:8000
just seed         # add a few links
just admin alice  # an admin for http://localhost:8000/admin; asks for a password
just logs         # optional: every container's logs in Dozzle, http://localhost:9999
just down
```

`just up` ends by listing the containers. The site is the `web` one, Caddy, at
the port its PORTS column shows (`0.0.0.0:8000->80/tcp`). If something else on
your machine already has 8000, set `WEB_PORT` in `.env` and run `just up` again.

To work on the code with reload, `just dev` runs Postgres in Docker and the api
(`:8080`) and Vite (`:5173`) on your machine. `just` lists every recipe.

## Layout

```
apps/
  api/            Go: cmd/api (the binary), internal/links and auth, migrations/
  web/            React + Vite, and the Caddyfile it's served with
e2e/              Playwright tests of the whole stack, run in Docker
deploy/           compose.yaml for the server: the released images, HTTPS
docs/             how the parts fit together, and how to contribute
scripts/          small helpers the recipes call
.github/          CI, release and Dependabot
compose.yaml      the local stack: db, migrate, api, web, and dozzle and e2e on demand
mise.toml         the tools the recipes run, and their versions
justfile          every command, with each folder's own in its justfile
lefthook.yml      git hooks: format and lint on commit, unit tests on push
```

## Tests

Three kinds, from many fast ones to a few slow ones that cover the most:

| Kind        | Where                                                                                           | What it checks                                                                                        | Run it                                  |
| ----------- | ----------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- | --------------------------------------- |
| Unit        | `apps/api/**/*_test.go` (fake store), `apps/web/src/*.test.ts(x)` (Vitest, jsdom, fake `fetch`) | One piece alone: the handler, URL checking, the api client, the page's behaviour                      | `just test`, and before every push      |
| Integration | `apps/api/internal/links/store_test.go`                                                         | The SQL, the migrations and the unique index, against a real Postgres (each test on a fresh database) | `just test-integration`                 |
| End-to-end  | `e2e/tests/*.spec.ts` (Playwright)                                                              | A browser using the whole stack through Caddy: shorten, list, redirect, count                         | `just test-e2e`, then `just e2e report` |

The integration tests skip unless `TEST_DATABASE_URL` is set, so `go test`
alone stays fast. The end-to-end tests run in Playwright's Docker image, so no
browser is installed on your machine (or in CI).

## CI/CD

- **CI** (`ci.yml`) runs on every pull request, checking only what changed:
  golangci-lint, the unit and integration tests (Postgres as a job service) and
  a build for the api; formatting, oxlint, the tests and a type-checked build for
  the web app; and, beside them, `just test-e2e`. `just ci` runs the same checks
  locally.
- **Release** (`release.yml`) runs on every push to `main`: CI again, then both
  images pushed to `ghcr.io/mabduqayum/fullmono-{api,web}`, tagged with the
  commit's sha and `latest`, then the deploy.

### Deploying

The deploy job is skipped until there is a server. To add one:

1. On the server (Docker installed), create `~/fullmono/.env` with
   `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `SITE_ADDRESS` (your
   domain, whose DNS points at the server) and `JWT_SECRET`
   (`openssl rand -base64 32`). Caddy fetches its HTTPS certificate on the
   first start.
2. Make the two GHCR packages public, or `docker login ghcr.io` on the server.
3. In the repository's settings, add the variable `DEPLOY_HOST` (`user@host`)
   and the secrets `DEPLOY_SSH_KEY` (a private key the server accepts) and
   `DEPLOY_KNOWN_HOSTS` (`ssh-keyscan <host>`).

Every release then copies `deploy/compose.yaml` to the server and restarts the
stack on the new images; migrations run before the api starts. To add an admin
there, in `~/fullmono`:

```sh
read -rsp 'Password: ' p && echo && printf '%s\n' "$p" | docker compose run --rm -T api set-admin alice
```
