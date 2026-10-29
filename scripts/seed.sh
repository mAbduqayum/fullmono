#!/usr/bin/env bash
# Shortens a few URLs through the running stack, so the page has links to click.
# Usage: scripts/seed.sh [base URL], default http://localhost:8000
set -euo pipefail

base_url="${1:-http://localhost:8000}"

for url in \
    https://go.dev/doc/effective_go \
    https://react.dev/learn \
    https://caddyserver.com/docs/caddyfile \
    https://just.systems/man/en/ \
    https://docs.docker.com/compose/; do
    curl --fail --silent --show-error \
        --header 'Content-Type: application/json' \
        --data "{\"url\": \"$url\"}" \
        "$base_url/api/links" >/dev/null
    echo "shortened $url"
done
