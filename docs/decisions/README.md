# Decisions

Why fullmono is built the way it is: an architecture decision record (ADR) for
each choice that would be costly to undo. [architecture.md](../architecture.md)
says how the parts fit together; these say why they're these parts, what each
choice costs, and what would make it the wrong one.

| ADR                                      | Decision                                                      |
| ---------------------------------------- | ------------------------------------------------------------- |
| [0001](0001-one-repo-for-both-apps.md)   | The api, the web app and their tooling share one repo         |
| [0002](0002-the-contract-comes-first.md) | The routes are written in openapi.yaml, and code made from it |
| [0003](0003-one-origin-behind-caddy.md)  | Caddy serves the built app and the api from one origin        |
| [0004](0004-a-jwt-in-a-cookie.md)        | An admin's login is a JWT in an httpOnly cookie               |
| [0005](0005-compose-on-one-server.md)    | Production is docker compose on one server                    |
| [0006](0006-images-tagged-by-commit.md)  | Images are tagged with their commit, and deployed by it       |
| [0007](0007-sql-through-sqlc.md)         | The SQL is written by hand, and sqlc makes Go of it           |

0001 to 0007 were written after the fact, for the choices the repo began with.
From now on, one comes with the change it decides.

## Writing one

A choice gets an ADR when it's hard to undo or a newcomer would ask about it: a
dependency that shapes the code, a new service, a change to how the stack is
deployed or how logins work. A bug fix or a refactor doesn't.

1. Take the next number, and name the file after the decision:
   `0008-links-expire.md`.
2. Copy the template below, fill it in, and add a row to the table above.
3. Put it in the pull request that makes the change, so that the review
   covers both.

An ADR on main isn't edited, except for its status. A decision that replaces
it is a new ADR, and the old one's status becomes "superseded by 0009", with a
link: the record keeps why each choice was made when it was, the ones that
turned out wrong too.

The format is [Michael Nygard's](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions),
with a "Revisit when" added:

```md
# 0008. The decision, as a sentence

Status: proposed, accepted, or superseded by a later one

## Context

What forces a choice: the problem, the constraints, the options weighed.

## Decision

What we do, in a few sentences.

## Consequences

What it makes easy, and what it costs or rules out.

## Revisit when

What would make it the wrong choice.
```
