# AGENTS.md

What a coding agent needs on top of the [README](README.md), which sets the
project up, and [CONTRIBUTING](docs/CONTRIBUTING.md), which says how a change
lands. `CLAUDE.md` is a link to this file.

## Commands

The tools are the ones [mise.toml](mise.toml) pins (`mise install`); Docker is
the one it can't install. Run everything through `just`, from the root:

- `just test`: the unit tests, both apps, no Docker
- `just test-integration`: the api's tests against Postgres in Docker
- `just test-e2e`: Playwright against the whole stack, in Docker
- `just lint`, `just fmt`: every app
- `just gen`: the code both apps generate from `apps/api/openapi.yaml`, and
  the api's from its SQL queries
- `just ci`: everything CI checks

One app alone: `just api <recipe>`, `just web <recipe>`, `just e2e <recipe>`.

## Rules

- Never read or print `.env`: it holds this machine's secrets. A new setting
  goes in `.env.example`, and in `deploy/compose.yaml` if the server needs it.
- The api's routes are its contract, `apps/api/openapi.yaml`: change it first,
  then `just gen`, then the handlers and the web calls the compiler points at.
  Never edit a `*.gen.*` file: it's regenerated.
- SQL lives in `apps/api/internal/db/*.sql`, one `-- name:` query at a time;
  `just gen` (sqlc) turns each into a method of `db.Queries`, which the stores
  call. No SQL strings in Go, and never edit the generated `.go` files there.
- A change comes with its tests, beside the code: a handler's run on the fake
  store (`handler_test.go`), what the contract refuses through the whole
  handler (`apps/api/api_test.go`), SQL's against Postgres (`store_test.go`).
- A schema change is a new numbered migration in `apps/api/migrations/`, by the
  rules in its README, with `schema.sql` updated to match: `schema_test.go`
  checks it does.
- A tool's version changes in `mise.toml`; bun's also in each `package.json`'s
  `packageManager` and the Dockerfiles' `oven/bun` tag.
- Commit messages are Conventional Commits, scoped `api`, `web` or `e2e`. The
  hooks format, lint and check them: fix what they report, never
  `--no-verify`.
