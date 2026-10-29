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

# Build and start the whole stack at http://localhost:8000, then list its containers
up:
    docker compose up --build --detach --wait
    docker compose ps

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

# Write the migrated schema to apps/api/migrations/schema.sql; commit it with the migration that changed it
schema: db
    #!/usr/bin/env bash
    set -euo pipefail
    # Structure only: no rows, owners, grants, or the migrator's own table.
    # The sed drops what changes on every dump (the Postgres version, and
    # \restrict's random key), so the file changes only when the schema does.
    {
        echo '-- The schema the migrations add up to, written by `just schema`. Change it with a migration.'
        docker compose exec -T db sh -c \
            'pg_dump --schema-only --no-owner --no-privileges --exclude-table=schema_migrations -U "$POSTGRES_USER" "$POSTGRES_DB"' \
            | sed -E '/^-- Dumped (from|by)/d; /^\\(un)?restrict /d'
    } | cat -s > apps/api/migrations/schema.sql

# A few links to click around with
seed:
    ./scripts/seed.sh

# Create an admin for http://localhost:8000/admin, or change their password
[positional-arguments]
admin username:
    #!/usr/bin/env bash
    set -euo pipefail
    read -rsp "Password for $1 (12 characters or more): " password
    echo
    # Piped in, so the password never shows in the process list or the shell's history.
    printf '%s\n' "$password" | docker compose run --rm --build -T api set-admin "$1"

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
