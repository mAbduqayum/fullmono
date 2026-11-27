# 0005. Production is docker compose on one server

Status: accepted

## Context

The stack is four containers and a database, with little traffic. The ways
to run it:

- **A platform** (Fly.io, Render, Railway): deploys, TLS and often Postgres
  taken care of, priced per service, and each one's own config.
- **Kubernetes**: replicas, rolling updates and self-healing, and a cluster to
  run and learn, which pays off at many services or many machines.
- **One server with docker compose**: the same compose file shape as local
  development, on a VM that costs a few dollars a month.

## Decision

One server, running [deploy/compose.yaml](../../deploy/compose.yaml): the
images CI built, Caddy in front, `restart: unless-stopped`, a daily backup,
and log files of capped size.

## Consequences

- Production is the stack `just up` runs, with released images in place of
  builds: little to learn, and nothing to pay for but the VM.
- Each deploy recreates the api and Caddy, so the site is down for a few
  seconds.
- The server is a single point of failure: when it's down, so is the site.
- Running it is ours to do: backups, OS updates, disk space.
- There are logs, but no metrics or alerts: nothing tells anyone the site is
  down, unless an outside check (an uptime monitor) does.
- One api process, so state in its memory, such as the login limit's counts,
  is the whole state.

## Revisit when

A few seconds of downtime per deploy matter, the traffic outgrows one machine,
or the database outgrows backups once a day. Managed Postgres would come
first, then more api replicas behind a load balancer, on a platform or
Kubernetes.
