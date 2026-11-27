# 0007. The SQL is written by hand, and sqlc makes Go of it

Status: accepted

## Context

The api's queries need to reach Go somehow:

- **An ORM** (GORM, ent): Go structs and method chains in place of SQL. The
  SQL it runs is hidden, and a mistake shows only at runtime.
- **A query builder** (squirrel): SQL assembled in Go, flexible for queries
  with optional parts, and unchecked until it runs.
- **database/sql or pgx by hand**: plain SQL in Go strings, with each row's
  columns scanned by hand, and nothing to check the two match.
- **sqlc**: plain SQL in `.sql` files, checked against the schema, made into
  typed Go methods.

## Decision

sqlc. Each query is a named block in
[internal/db](../../apps/api/internal/db/links.sql); `just gen` checks it
against the schema the migrations build and generates a method on
`db.Queries`, which the stores call through pgx.

## Consequences

- The SQL is what runs: it can be read, and pasted into `psql`.
- A query that doesn't fit the schema, a column renamed say, fails at
  `just gen`, before anything runs.
- No SQL strings in Go.
- A query with optional parts (filter by this, or by that) is awkward: one
  query with `@param IS NULL OR …` for each, or several queries.

## Revisit when

Queries are mostly built from a user's choices (reports, filters): a query
builder fits that better, and can sit beside sqlc.
