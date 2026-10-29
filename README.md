# fullmono

A URL shortener, kept small on purpose: it's here to show the tooling around
a fullstack monorepo — one `justfile`, Docker images and compose, Caddy in
front, git hooks, CI/CD on GitHub Actions and Dependabot.

- **api**: Go, `net/http` and Postgres (pgx). `POST /api/links` shortens a URL,
  `GET /api/links` lists them, `GET /r/{code}` redirects and counts the click.
- **web**: React and Vite on bun, served by Caddy, which also routes `/api` and
  `/r` to the api.

## Quick start

You need Docker, [just](https://just.systems), Go, [bun](https://bun.sh) and
[lefthook](https://lefthook.dev).

```sh
just setup   # .env from .env.example, dependencies, git hooks
just up      # build and start everything: http://localhost:8000
just seed    # add a few links
just logs    # optional: every container's logs in Dozzle, http://localhost:9999
just down
```

To work on the code with reload, `just dev` runs Postgres in Docker and the api
(`:8080`) and Vite (`:5173`) on your machine. `just` lists every recipe.

## Layout

```
apps/
  api/            Go: cmd/api (the binary), internal/links, migrations/
  web/            React + Vite, and the Caddyfile it's served with
deploy/           compose.yaml for the server: the released images, HTTPS
scripts/          small helpers the recipes call
.github/          CI, release and Dependabot
compose.yaml      the local stack: db, migrate, api, web, dozzle
justfile          every command, with each app's own in apps/*/justfile
lefthook.yml      git hooks: format and lint on commit, tests on push
```

## CI/CD

- **CI** (`ci.yml`) runs on every pull request, checking only the app that
  changed: golangci-lint, `go test -race` and a build for the api; formatting,
  oxlint and a type-checked build for the web app. `just ci` runs the same checks
  locally.
- **Release** (`release.yml`) runs on every push to `main`: CI again, then both
  images pushed to `ghcr.io/mabduqayum/fullmono-{api,web}`, tagged with the
  commit's sha and `latest`, then the deploy.

### Deploying

The deploy job is skipped until there is a server. To add one:

1. On the server (Docker installed), create `~/fullmono/.env` with
   `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` and `SITE_ADDRESS` (your
   domain, whose DNS points at the server). Caddy fetches its HTTPS certificate
   on the first start.
2. Make the two GHCR packages public, or `docker login ghcr.io` on the server.
3. In the repository's settings, add the variable `DEPLOY_HOST` (`user@host`)
   and the secrets `DEPLOY_SSH_KEY` (a private key the server accepts) and
   `DEPLOY_KNOWN_HOSTS` (`ssh-keyscan <host>`).

Every release then copies `deploy/compose.yaml` to the server and restarts the
stack on the new images; migrations run before the api starts.
