# 0001. The api, the web app and their tooling share one repo

Status: accepted

## Context

The web app is the api's only client, and most changes touch both: a new
field is a column, a route, a type in the client and a form. In two repos,
that change is two pull requests, merged and released in the right order, with
the client's copy of the api's types going stale in between.

The tools that manage many packages in one repo (Nx, Turborepo, Bazel) earn
their setup with many packages that build on one another. Here there are two
apps in two languages, which share no code, only a contract.

## Decision

One repo: `apps/api`, `apps/web`, the end-to-end tests, the deploy files and
CI. No workspace tool: [mise.toml](../../mise.toml) pins every tool once, and
the root [justfile](../../justfile) runs each app's own recipes
(`just api test`), so each app stays a plain Go module or bun package.

## Consequences

- A change lands whole, in one pull request that its tests and the end-to-end
  tests check together, and a release always ships an api and a web app that
  were tested together.
- The contract has one copy, `apps/api/openapi.yaml`, which both apps generate
  code from ([0002](0002-the-contract-comes-first.md)).
- CI must not run everything for every change: `ci.yml` checks which folders
  changed and runs only their jobs.
- Both apps share one history and one release: a fix to the web app rebuilds
  and redeploys the api too.

## Revisit when

Each app has its own team and release schedule, or the apps start sharing code
that one build should order, which is what a workspace tool is for.
