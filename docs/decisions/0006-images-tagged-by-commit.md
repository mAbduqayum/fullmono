# 0006. Images are tagged with their commit, and deployed by it

Status: accepted

## Context

A deploy must say which images to run, and a rollback which ones ran before.
`latest` says neither: it moves with every release, so the server can't tell
what it runs, and an old one can't be named. Version numbers (semver) tell the
users of a package what changed, but nobody runs these images but us.

## Decision

[release.yml](../../.github/workflows/release.yml) tags both images with the
commit's full sha (and `latest`, for a look by hand, which no deploy uses).
[deploy.yml](../../.github/workflows/deploy.yml) deploys
one sha, and the server's compose file requires it. Every push to main is a
release; a rollback is the deploy of an earlier sha.

## Consequences

- What runs is a commit: its code, its compose file, its migrations.
- A rollback takes as long as a deploy, with nothing to rebuild, and needs no
  revert on main first.
- A rollback runs older code on the newer schema, so every migration must work
  with the code before it, and the older api must leave the newer schema as it
  is ([migrations](../../apps/api/migrations/README.md)).
- Images pile up in the registry, one pair per push, until something deletes
  the old ones.

## Revisit when

Someone else runs these images, and needs to know which changes break things:
then semver tags beside the sha.
