# syntax=docker/dockerfile:1
# Build context is the repository root:
#   docker build -f docker/Dockerfile -t factorio-server-manager .

FROM node:26-bookworm-slim AS frontend
WORKDIR /build
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY webpack.config.js tailwind.config.js ./
COPY ui ./ui
COPY app ./app
RUN npm run build

FROM golang:1.27 AS backend
WORKDIR /build/src
COPY src/go.mod src/go.sum ./
RUN go mod download
COPY src/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/factorio-server-manager .

# Glibc is required for the Factorio headless server binaries.
FROM ubuntu:24.04

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl jq xz-utils \
    && rm -rf /var/lib/apt/lists/*

ENV FACTORIO_VERSION=stable \
    FACTORIO_ALLOW_MAJOR_UPGRADE=false \
    FSM_SECURE_COOKIES=false \
    RCON_PASS=""

WORKDIR /opt/fsm
COPY --from=backend /out/factorio-server-manager ./factorio-server-manager
COPY --from=frontend /build/app ./app
COPY conf.json.example ./conf.json
COPY --chmod=0755 docker/entrypoint.sh /opt/entrypoint.sh
RUN mkdir -p /opt/fsm-data /opt/fsm/mod_packs \
             /opt/factorio/saves /opt/factorio/mods /opt/factorio/config

VOLUME /opt/fsm-data /opt/factorio/saves /opt/factorio/mods /opt/factorio/config

EXPOSE 80/tcp 34197/udp

# The first start downloads Factorio before the web UI comes up, hence the long start period.
HEALTHCHECK --interval=30s --timeout=5s --start-period=5m --retries=3 \
    CMD curl -fsS -o /dev/null http://127.0.0.1/ || exit 1

ENTRYPOINT ["/opt/entrypoint.sh"]
