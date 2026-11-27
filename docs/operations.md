# Operations

Running fullmono in production: the first deploy, then deploys and rollbacks,
backups, logs, admins and secrets. Production is docker compose on one server
([0005](decisions/0005-compose-on-one-server.md)), and the commands below run
there, in `~/fullmono`, unless they say otherwise.

## The first deploy

The deploy job is skipped until there is a server. To add one:

1. On the server, with Docker installed and ports 80 and 443 open, make the
   folder and its backups' folder, `mkdir -p ~/fullmono/backups`, and write
   `~/fullmono/.env`, readable by you alone (`chmod 600 .env`):

    ```sh
    POSTGRES_USER=fullmono
    POSTGRES_PASSWORD=<openssl rand -hex 24>
    POSTGRES_DB=fullmono
    SITE_ADDRESS=links.example.com
    JWT_SECRET=<openssl rand -base64 32>
    ```

    `SITE_ADDRESS` is your domain, whose DNS points at the server: Caddy gets
    its HTTPS certificate on the first start. End the file with a newline, as
    an editor does: each deploy adds a `TAG=` line to it.
2. Make the two GHCR packages public, or `docker login ghcr.io` on the server
   with a token that can read packages.
3. In the repository's settings, add the variable `DEPLOY_HOST` (`user@host`)
   and the secrets `DEPLOY_SSH_KEY` (a private key the server accepts) and
   `DEPLOY_KNOWN_HOSTS` (`ssh-keyscan <host>`).
4. Push to `main`, or run the Deploy workflow with `main`'s sha (below).

## Deploys

Every push to `main` is a release ([release.yml](../.github/workflows/release.yml)):
CI again, then both images pushed, tagged with the commit's sha, then
[deploy.yml](../.github/workflows/deploy.yml), which:

1. copies that commit's [deploy/compose.yaml](../deploy/compose.yaml) to the
   server;
2. records the sha as `TAG` in `.env`, so `docker compose` there always runs
   what's deployed;
3. pulls the images and runs `docker compose up --detach --wait`: `migrate`
   applies the new migrations, then the api starts, then Caddy, and the job
   fails if one of them isn't healthy.

The site is down for the few seconds the api and Caddy take to restart. To see
what's running: `docker compose ps`, whose images end in the sha.

## Rolling back

A release that breaks something is undone by deploying the one before it:

1. Find the last good commit's full sha: `git log main`, or the Release
   workflow's runs.
2. Run the Deploy workflow with it: on GitHub, Actions, Deploy, "Run workflow";
   or from a terminal, `gh workflow run deploy.yml -f tag=<sha>`.

Only a commit that a Release run built has images: one of `main`'s, not a
branch's. The schema stays as the newer release left it: the older api's
`migrate` leaves migrations it doesn't know, and [the migrations'
rules](../apps/api/migrations/README.md) keep each schema working with the code
before it. A migration that damaged data is another matter: restore a backup.

Then fix `main`, by a revert or a fix: its next push deploys again.

## Backups

The `backup` container dumps the database when it starts and then every day,
into `~/fullmono/backups/<date>.dump` (pg_dump's custom format), and deletes
those older than two weeks.

- They share the database's disk, so a lost server loses both. Copy them off
  it, from another machine, on a schedule (cron):
  `rsync -a user@host:fullmono/backups/ fullmono-backups/`.
- A backup nobody has restored isn't known to work: try one now and then, on
  your machine.

### Restoring

On the server, with the api stopped so that nothing writes meanwhile:

```sh
docker compose stop api
docker compose exec -T db sh -c 'pg_restore --clean --if-exists --single-transaction --no-owner -U "$POSTGRES_USER" -d "$POSTGRES_DB"' < backups/2026-11-27.dump
docker compose up --detach --wait
```

`--clean --if-exists` drops each table before restoring it, and
`--single-transaction` makes it all or nothing: a restore that fails leaves the
database as it was. `up` runs `migrate` before it starts the api, so a dump
from before a migration catches up.

### A drill, on your machine

```sh
just up && just seed
just backup                    # prints backups/<time>.dump
just reset && just up          # an empty database
just restore backups/<time>.dump
```

The seeded links are back at http://localhost:8000.

## Logs

```sh
docker compose logs api --since 1h   # or -f to follow, or web for Caddy's
```

Docker keeps each container's last 30 MB (three files of 10 MB). The api logs
one JSON line per request, and every line a request logs carries its
`request_id`, which the answer's `X-Request-Id` header holds and a 500's
message names. A visitor who reports "something went wrong (request X)" leads
to what happened:

```sh
docker compose logs api | grep X
```

Failed admin logins are `admin login failed` lines, with the name tried and
the address; an address with 10 failures in 15 minutes gets
`admin login refused` until the oldest is 15 minutes old.

## Admins

To add one, or change their password:

```sh
read -rsp 'Password: ' p && echo && printf '%s\n' "$p" | docker compose run --rm -T api set-admin alice
```

To remove one, in `docker compose exec db sh -c 'psql -U "$POSTGRES_USER" "$POSTGRES_DB"'`:
`DELETE FROM admins WHERE username = 'alice';`. Their login lasts until it
expires, up to 8 hours, unless `JWT_SECRET` changes
([0004](decisions/0004-a-jwt-in-a-cookie.md)).

The login limit counts in the api's memory: `docker compose restart api`
clears it, for an admin locked out by their own typos.

## Rotating secrets

- **`JWT_SECRET`**: put a new one in `.env`, then `docker compose up --detach --wait`.
  Every admin is logged out.
- **`POSTGRES_PASSWORD`**: Postgres reads it only when it creates its volume,
  so a new value in `.env` alone locks the api out. Change it in Postgres
  first: in psql (as above), `\password`, and the new password twice. Then
  `.env`, then `docker compose up --detach --wait`.
- **`DEPLOY_SSH_KEY`**: make a new key pair, add its public key to the
  server's `~/.ssh/authorized_keys`, replace the secret, then remove the old
  key from the server.
