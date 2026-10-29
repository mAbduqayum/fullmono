# syntax=docker/dockerfile:1

# Build stage: bun and node_modules, never shipped.
FROM oven/bun:1.4.3-alpine AS build
WORKDIR /app

# Dependencies first: this layer is reused until package.json or bun.lock change.
COPY package.json bun.lock bunfig.toml ./
RUN --mount=type=cache,target=/root/.bun/install/cache \
    bun install --frozen-lockfile

COPY . .
RUN bun run build

# Runtime stage: Caddy serving the built files, which are all the web app is.
FROM caddy:2.11.6-alpine
COPY Caddyfile /etc/caddy/Caddyfile
COPY --from=build /app/dist /srv
