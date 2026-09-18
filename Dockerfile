# Self-hosted "web mode" image for Typstify (docs/plans/plan_web_version.md,
# Giai doan 4). Runs cmd/typstify-server (the headless backend) plus the
# built web frontend behind a single HTTP port. See docs/web-server.md.

# ---- frontend ----
FROM node:22-slim AS frontend
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---- backend ----
FROM golang:1.25-bookworm AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/typstify-server ./cmd/typstify-server

# ---- typst + tinymist ----
FROM debian:bookworm-slim AS tools
ARG TYPST_VERSION=v0.15.1
ARG TINYMIST_VERSION=v0.15.8
RUN apt-get update && apt-get install -y --no-install-recommends curl xz-utils ca-certificates \
    && rm -rf /var/lib/apt/lists/*
RUN curl -fsSL -o /tmp/typst.tar.xz \
      "https://github.com/typst/typst/releases/download/${TYPST_VERSION}/typst-x86_64-unknown-linux-musl.tar.xz" \
    && tar -xJf /tmp/typst.tar.xz -C /tmp \
    && mv "/tmp/typst-x86_64-unknown-linux-musl/typst" /usr/local/bin/typst \
    && chmod +x /usr/local/bin/typst \
    && rm -rf /tmp/typst*
RUN curl -fsSL -o /usr/local/bin/tinymist \
      "https://github.com/Myriad-Dreamin/tinymist/releases/download/${TINYMIST_VERSION}/tinymist-linux-x64" \
    && chmod +x /usr/local/bin/tinymist

# ---- final ----
# node:22-slim (not a plain distroless/scratch image) because the default
# AI agent -- and most agents in the ACP registry -- are distributed via
# npx (see risk #4 in docs/plans/plan_web_version.md); Node must be present
# on PATH for them to start. uvx-based or platform-binary agents are not
# supported by this image in v1.
FROM node:22-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=tools /usr/local/bin/typst /usr/local/bin/typst
COPY --from=tools /usr/local/bin/tinymist /usr/local/bin/tinymist
COPY --from=backend /out/typstify-server /usr/local/bin/typstify-server
COPY --from=frontend /src/web/dist /app/web/dist

RUN useradd --create-home --home-dir /data --shell /usr/sbin/nologin typstify \
    && mkdir -p /data/project /data/config /data/cache \
    && chown -R typstify:typstify /data /app

USER typstify
WORKDIR /app

# TYPSTIFY_SERVER_PASSWORD has no default here on purpose -- it must be set
# explicitly by whoever runs the container (see docs/web-server.md).
ENV TYPSTIFY_STATIC_DIR=/app/web/dist \
    TYPSTIFY_PROJECT_DIR=/data/project \
    TYPSTIFY_SERVER_ADDR=:8080 \
    HOME=/data \
    XDG_CONFIG_HOME=/data/config \
    XDG_CACHE_HOME=/data/cache

VOLUME ["/data"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD node -e "fetch('http://127.0.0.1:8080/api/health').then(r=>process.exit(r.ok?0:1)).catch(()=>process.exit(1))"

ENTRYPOINT ["typstify-server"]
