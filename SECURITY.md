# Security

## Reporting a vulnerability

Don't open an issue for it: issues are public. Report it privately instead,
through GitHub's
[private vulnerability reporting](https://github.com/mAbduqayum/fullmono/security/advisories/new):
the Security tab, then **Report a vulnerability**.

Say:

- what an attacker can do with it, and what they need first (an admin's login,
  say);
- how to reproduce it: the requests, or the steps from `just up`;
- the commit you found it on.

Answers are best effort, with no promised response time. A fix lands on `main`
and ships with the next release; the advisory is published after that, with
credit to you if you'd like it.

## What's covered

Only the latest `main` and the images released from it,
`ghcr.io/mabduqayum/fullmono-{api,web}`: older commits get no fixes. That
covers the api, the web app and the Caddy in front of it, the images, the
compose files and the workflows.

Not covered:

- a vulnerability in a dependency that fullmono's code never reaches:
  Dependabot updates the dependency, and CI's govulncheck flags the ones it
  does reach;
- the values in `.env.example`, which are for a laptop and say so.
