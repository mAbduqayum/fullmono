# fullmono's commands: `just` lists them, `just <recipe>` runs one.
# Each app keeps its own recipes in its folder: `just api test`, `just web dev`.

set dotenv-load

mod api 'apps/api'
mod web 'apps/web'
mod e2e

# List the recipes, in this file's order
[private]
default:
    @just --list --list-submodules --unsorted

# First run: .env from the template, the apps' dependencies, the git hooks
setup:
    [ -f .env ] || cp .env.example .env
    just api setup
    just web setup
    just e2e setup
    lefthook install

# Build and start the whole stack at http://localhost:8000
up:
    docker compose up --build --detach --wait

# Stop the stack, keeping the database
down:
    docker compose down

# Stop the stack and delete the database
[confirm("Delete the database and every link in it? [y/N]")]
reset:
    docker compose down --volumes

# Start Postgres alone and migrate it, for `just dev`
db:
    docker compose up --detach --wait db
    docker compose run --rm --build migrate

# The api and the web dev server on the host, reloading as you edit; Ctrl+C stops both
dev: db
    #!/usr/bin/env bash
    trap 'kill 0' EXIT
    just api dev &
    just web dev &
    wait

# Open the log viewer (Dozzle) at http://localhost:9999
logs:
    docker compose --profile tools up --detach dozzle
    @echo "Dozzle: http://localhost:9999"

# A psql shell in the database
psql:
    docker compose exec db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"

# A few links to click around with
seed:
    ./scripts/seed.sh

# The unit tests, both apps: fast, no Docker, run before every push
test:
    just api test
    just web test

# The api's integration tests, against Postgres in Docker
test-integration: db
    just api test-integration

# The end-to-end tests: a browser in Docker against the whole stack; `just e2e report` shows the result
test-e2e: up
    #!/usr/bin/env bash
    # The report is copied out of the container, not written to a mounted
    # folder, where its files would belong to root.
    docker rm --force fullmono-e2e >/dev/null 2>&1
    docker compose --profile e2e run --build --name fullmono-e2e e2e
    status=$?
    rm -rf e2e/playwright-report
    docker cp --quiet fullmono-e2e:/e2e/playwright-report e2e/
    docker rm fullmono-e2e >/dev/null
    exit $status

# Lint and check formatting, everywhere
lint:
    just api lint
    just web lint
    just e2e lint

# Format everything
fmt:
    just api fmt
    just web fmt
    just e2e fmt

# What CI checks, run here first
ci: lint test test-integration
    just web build
    just test-e2e
