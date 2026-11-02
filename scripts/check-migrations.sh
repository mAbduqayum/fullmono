#!/usr/bin/env bash
# Fails when this branch edits, renames or deletes a migration that <base>
# already has: a database that ran it won't run it again, so a change to the
# schema is a new migration (apps/api/migrations/README.md). Adding one is fine.
# Usage: scripts/check-migrations.sh <base>, like origin/main
set -euo pipefail

base="${1:?usage: scripts/check-migrations.sh <base>, like origin/main}"

# A push that creates a branch has no base (all zeros), and one after a force
# push may name a commit this clone doesn't have: there's nothing to compare.
if ! git rev-parse --quiet --verify "$base^{commit}" >/dev/null; then
    echo "check-migrations: no commit $base here, so nothing to compare with" >&2
    exit 0
fi

# base...HEAD: only what this branch changed since it left base, not what base
# gained since (such as a newer migration, which would look deleted here).
changed=$(git diff --name-only --diff-filter=MDR "$base...HEAD" -- 'apps/api/migrations/[0-9]*.sql')
if [[ -n $changed ]]; then
    echo "These migrations are already on $base; write a new one instead of changing them:" >&2
    echo "$changed" >&2
    exit 1
fi
