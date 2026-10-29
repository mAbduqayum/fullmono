# fullmono's commands: `just` lists them, `just <recipe>` runs one.
# Each app keeps its own recipes in its folder: `just api test`, `just web dev`.

set dotenv-load

mod api 'apps/api'
mod web 'apps/web'

# List the recipes, in this file's order
[private]
default:
    @just --list --list-submodules --unsorted

# First run: .env from the template, the apps' dependencies, the git hooks
setup:
    [ -f .env ] || cp .env.example .env
    just api setup
    just web setup
    lefthook install

# Build and start the whole stack at http://localhost:8000
up:
    docker compose up --build --detach --wait

# Stop the stack, keeping the database
down:
    docker compose down

# Stop the stack and delete the database
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

# The api's tests (the web app is type-checked by its build)
test:
    just api test

# Lint and check formatting, both apps
lint:
    just api lint
    just web lint

# Format both apps
fmt:
    just api fmt
    just web fmt

# What CI checks, run here first
ci: lint test
    just web build
