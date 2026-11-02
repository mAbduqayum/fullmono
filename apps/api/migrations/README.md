# Migrations

The database's schema, as numbered SQL files. `api migrate` applies the ones a
database doesn't have yet, in order, and records each in its
`schema_migrations` table. `just db` runs it on your laptop, and compose's
`migrate` container before the api starts. [schema.sql](schema.sql) is what
they add up to, for reading.

## Writing one

1. Take the next number and name the change, in a pair of files:
   `0003_add_link_title.up.sql` makes it, `0003_add_link_title.down.sql`
   undoes it.
2. Update [schema.sql](schema.sql) to match: `just test-integration` builds
   both and fails if they differ.
3. `just gen`: [sqlc](../sqlc.yaml) reads these files, so its models gain the
   change, and a query that no longer fits fails here rather than at runtime.
4. `just db` applies it to your database.

A file runs as one transaction, so a statement that fails leaves none of the
file's changes behind. That's also why `CREATE INDEX CONCURRENTLY`, which can't
run inside a transaction, needs a file of its own.

## Rules

- **Never change a migration that's on main.** A database that already ran it
  won't run it again, so the edit would reach only new databases, and they'd
  differ from production. Change the schema with a new migration instead:
  [check-migrations.sh](../../../scripts/check-migrations.sh) refuses an edit,
  on `git push` and in CI.
- **Keep the running api working.** A deploy migrates first, while the old api
  still serves, and a rollback runs the old api on the new schema: every
  migration has to work with the code before it too. So a new column is
  nullable or has a default, and a rename or a drop takes steps, each in its
  own release (expand, then contract):
  1. add the new column, and write to both;
  2. copy the old values over, and read the new one;
  3. drop the old column, which nothing uses anymore.
- **Production only moves forward.** Nothing runs the down files: they say how
  to undo a change, for whoever has to by hand. On your laptop, `just reset`
  starts the database over; on a server, a mistake that shipped is fixed by a
  new migration.
