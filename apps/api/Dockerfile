# syntax=docker/dockerfile:1

# Build stage: the Go toolchain, ~300 MB, never shipped.
FROM golang:1.27-alpine AS build
WORKDIR /src

# Dependencies first: this layer is reused until go.mod or go.sum change.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Runtime stage: one static binary on distroless, ~15 MB, no shell, not root.
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/api /api
USER nonroot:nonroot
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/api", "healthcheck"]
ENTRYPOINT ["/api"]
