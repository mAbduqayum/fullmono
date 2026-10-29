# syntax=docker/dockerfile:1

# Playwright's image: Chromium and every library it needs, already installed.
# Its version must be package.json's @playwright/test version, or the tests
# look for a browser build the image doesn't have.
FROM mcr.microsoft.com/playwright:v1.64.1-noble

# The image has Node; bun only installs the packages.
COPY --from=oven/bun:1.4.3 /usr/local/bin/bun /usr/local/bin/bun

WORKDIR /e2e
COPY package.json bun.lock bunfig.toml ./
RUN --mount=type=cache,target=/root/.bun/install/cache \
    bun install --frozen-lockfile

COPY . .
CMD ["bun", "run", "test"]
