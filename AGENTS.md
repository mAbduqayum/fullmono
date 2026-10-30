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
- `just ci`: everything CI checks

One app alone: `just api <recipe>`, `just web <recipe>`, `just e2e <recipe>`.

## Rules

- Never read or print `.env`: it holds this machine's secrets. A new setting
  goes in `.env.example`, and in `deploy/compose.yaml` if the server needs it.
- A change comes with its tests, beside the code: a handler's run on the fake
  store (`handler_test.go`), SQL's against Postgres (`store_test.go`).
- A schema change is a new numbered migration in `apps/api/migrations/`, with
  `schema.sql` updated to match.
- A tool's version changes in `mise.toml`; bun's also in each `package.json`'s
  `packageManager` and the Dockerfiles' `oven/bun` tag.
- Commit messages are Conventional Commits, scoped `api`, `web` or `e2e`. The
  hooks format, lint and check them: fix what they report, never
  `--no-verify`.
