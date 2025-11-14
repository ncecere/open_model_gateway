# syntax=docker/dockerfile:1.7

###############################
# Stage 1: Build frontend assets
###############################
# Enable access to BuildKit platforms for multi-arch builds
ARG BUILDPLATFORM=linux/amd64
ARG TARGETPLATFORM=linux/amd64
ARG TARGETOS=linux
ARG TARGETARCH=amd64

FROM --platform=$BUILDPLATFORM oven/bun:1 AS frontend-builder
WORKDIR /src
COPY backend/frontend ./backend/frontend
WORKDIR /src/backend/frontend
RUN bun install --frozen-lockfile \
    && bun run build

###############################
# Stage 2: Build Go backend
###############################
FROM --platform=$BUILDPLATFORM golang:1.25 AS backend-builder
ARG TARGETOS
ARG TARGETARCH
ENV GOTOOLCHAIN=auto
WORKDIR /src
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download
COPY backend ./backend
RUN mkdir -p /src/backend/internal/httpserver/ui/dist
COPY --from=frontend-builder /src/backend/frontend/dist /src/backend/internal/httpserver/ui/dist
RUN cd backend && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /router ./cmd/routerd

###############################
# Stage 3: Runtime image
###############################
FROM --platform=$TARGETPLATFORM debian:bookworm-slim
ENV ROUTER_CONFIG_FILE=/config/router.yaml \
    ROUTER_DB_URL=postgres://open_gateway:open_gateway@postgres:5432/open_gateway?sslmode=disable \
    ROUTER_DATABASE_URL=postgres://open_gateway:open_gateway@postgres:5432/open_gateway?sslmode=disable \
    ROUTER_REDIS_URL=redis://redis:6379/0
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --create-home --home /home/router --shell /usr/sbin/nologin router
WORKDIR /home/router
COPY --from=backend-builder /router /usr/local/bin/router
COPY backend/migrations /home/router/migrations
RUN mkdir -p /config && chown -R router:router /home/router /config
USER router
EXPOSE 8090
ENTRYPOINT ["/usr/local/bin/router"]
