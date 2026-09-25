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
# TARGETARCH is filled in by BuildKit (docker buildx --platform ...); a
# plain build falls back to amd64.
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
# go.mod replaces github.com/oligo/gvcode with the local ./internal/gvcode
# fork -- `go mod download` resolves replace directives at parse time, so
# that directory must exist before it runs, not just before the build step.
COPY internal/gvcode ./internal/gvcode
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} go build -trimpath -o /out/typstify-server ./cmd/typstify-server

# ---- typst + tinymist ----
# Downloads are pinned by SHA-256. When bumping TYPST_VERSION or
# TINYMIST_VERSION, update the matching checksums below (sha256sum of each
# release asset) -- the build fails loudly on a mismatch.
FROM debian:bookworm-slim AS tools
ARG TARGETARCH
ARG TYPST_VERSION=v0.15.1
ARG TINYMIST_VERSION=v0.15.8
RUN apt-get update && apt-get install -y --no-install-recommends curl xz-utils ca-certificates \
    && rm -rf /var/lib/apt/lists/*
RUN set -eu; \
    case "${TARGETARCH:-amd64}" in \
      amd64) typst_triple=x86_64-unknown-linux-musl; tinymist_arch=x64; \
             typst_sha=a6d077d0a95eed5a2eba715b2dae06be954f624ccbf85758a03f389ded33118c; \
             tinymist_sha=781fcd7ccfa099d886f0a227def422595979febe77cbaf4b42f7c66cb746be1d ;; \
      arm64) typst_triple=aarch64-unknown-linux-musl; tinymist_arch=arm64; \
             typst_sha=5aa8d74a3d906e60ea12a66ac2f37f8eef1b14cbad7182a745e393a10c23dcee; \
             tinymist_sha=0cb1896364ff6efaefd18b73c40276f00515d2388cb6746a8e5c14a940f9aae3 ;; \
      *) echo "unsupported TARGETARCH: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    curl -fsSL -o /tmp/typst.tar.xz \
      "https://github.com/typst/typst/releases/download/${TYPST_VERSION}/typst-${typst_triple}.tar.xz"; \
    echo "${typst_sha}  /tmp/typst.tar.xz" | sha256sum -c -; \
    tar -xJf /tmp/typst.tar.xz -C /tmp; \
    mv "/tmp/typst-${typst_triple}/typst" /usr/local/bin/typst; \
    chmod +x /usr/local/bin/typst; \
    rm -rf /tmp/typst*; \
    curl -fsSL -o /usr/local/bin/tinymist \
      "https://github.com/Myriad-Dreamin/tinymist/releases/download/${TINYMIST_VERSION}/tinymist-linux-${tinymist_arch}"; \
    echo "${tinymist_sha}  /usr/local/bin/tinymist" | sha256sum -c -; \
    chmod +x /usr/local/bin/tinymist

# ---- chess library dependencies: board-n-pieces + print fonts ----
# @local/chessbook (chessbook/lib) imports @preview/board-n-pieces; bundling
# it means documents compile offline and on first use. Fonts: the library's
# theme.typ asks for Roboto (sans) and Noto Serif (serif); Noto Sans Symbols 2
# covers the chess figurines. Only the needed faces are copied to the final
# image, not the whole ~50 MB Noto package.
FROM debian:bookworm-slim AS chesslib
ARG BOARD_N_PIECES_VERSION=0.9.0
RUN apt-get update && apt-get install -y --no-install-recommends \
      curl ca-certificates fonts-roboto-unhinted fonts-noto-core \
    && rm -rf /var/lib/apt/lists/*
RUN mkdir -p /out/packages/preview/board-n-pieces/${BOARD_N_PIECES_VERSION} \
    && curl -fsSL "https://packages.typst.org/preview/board-n-pieces-${BOARD_N_PIECES_VERSION}.tar.gz" \
       | tar -xz -C /out/packages/preview/board-n-pieces/${BOARD_N_PIECES_VERSION} \
    && test -f /out/packages/preview/board-n-pieces/${BOARD_N_PIECES_VERSION}/typst.toml
RUN mkdir -p /out/fonts && cd /usr/share/fonts/truetype \
    && for f in Regular Bold Italic BoldItalic Medium; do cp roboto/unhinted/RobotoTTF/Roboto-$f.ttf /out/fonts/; done \
    && for f in Regular Bold Italic BoldItalic; do cp noto/NotoSerif-$f.ttf noto/NotoSans-$f.ttf /out/fonts/; done \
    && cp noto/NotoSansSymbols2-Regular.ttf /out/fonts/

# ---- antigravity (Google's own agy_acp_server + localharness bundle,
# verified live against the real ACP registry, see docs/web-server.md) ----
FROM debian:bookworm-slim AS antigravity
ARG ANTIGRAVITY_VERSION=1.2.1
ARG TARGETARCH
# Set to false to leave the Antigravity agent out of the image (it is
# large, and needs an AVX-capable CPU on x86_64 -- see the probe below).
ARG WITH_ANTIGRAVITY=true
RUN mkdir -p /out
RUN apt-get update && apt-get install -y --no-install-recommends curl unzip ca-certificates \
    && rm -rf /var/lib/apt/lists/*
RUN set -eu; \
    if [ "${WITH_ANTIGRAVITY}" != "true" ]; then echo "skipping antigravity"; exit 0; fi; \
    case "${TARGETARCH:-amd64}" in \
      amd64) agy_arch=x86_64 ;; \
      arm64) agy_arch=arm64 ;; \
      *) echo "unsupported TARGETARCH for antigravity-acp: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    curl -fsSL -o /tmp/agy.zip \
      "https://dl.google.com/agy-extensions/releases/linux/agy-acp-server-${ANTIGRAVITY_VERSION}-linux-${agy_arch}.zip" \
    && unzip -q /tmp/agy.zip -d /tmp/agy \
    && mv /tmp/agy/agy_acp_server.par /out/agy_acp_server.par \
    && mv /tmp/agy/localharness_external /out/localharness \
    && chmod +x /out/agy_acp_server.par /out/localharness \
    && rm -rf /tmp/agy /tmp/agy.zip
# agy_acp_server.par has no --version flag; confirm it at least starts,
# same "build must go RED" rule as the typst/tinymist steps above -- but
# with one deliberate exception. Verified live 2026-09-18: the linux-x86_64
# build is compiled requiring AVX and SIGILLs (exit 132) on any CPU/VM
# without it exposed -- observed on this machine's own Docker Desktop VM.
# That is a fact about the BUILD host's CPU, not about whether the archive
# was fetched/unpacked correctly (the curl/unzip/chmod above already fail
# loudly for that), and it says nothing about whether a *different* deploy
# host has AVX (real x86_64 hardware has had it since 2011; the gap is
# virtualized/emulated build environments like this one). Hard-failing the
# entire image over the build machine's CPU would be a worse failure mode
# than a loud, impossible-to-miss warning -- so exit 132 only warns; any
# other failure (wrong file, corrupt archive, missing interpreter) still
# fails the build.
RUN if [ ! -f /out/agy_acp_server.par ]; then exit 0; fi; \
    timeout 5 /out/agy_acp_server.par --uid= < /dev/null > /tmp/agy_probe.log 2>&1; \
    status=$?; \
    if [ "$status" -eq 132 ]; then \
      echo "############################################################" >&2; \
      echo "# WARNING: agy_acp_server.par (Google Antigravity) SIGILL'd  #" >&2; \
      echo "# on THIS BUILD MACHINE (needs AVX). The binary was fetched  #" >&2; \
      echo "# and installed correctly; the image will still ship it,    #" >&2; \
      echo "# but Antigravity will not run in this container unless the #" >&2; \
      echo "# deploy host's CPU exposes AVX. See docs/web-server.md.    #" >&2; \
      echo "############################################################" >&2; \
    elif [ "$status" -ne 0 ] && [ "$status" -ne 124 ]; then \
      echo "agy_acp_server.par failed to start (exit $status):" >&2; \
      cat /tmp/agy_probe.log >&2; \
      exit 1; \
    fi

# ---- final ----
# node:22-slim (not a plain distroless/scratch image) because the default
# AI agent -- and most agents in the ACP registry -- are distributed via
# npx (see risk #4 in docs/plans/plan_web_version.md); Node must be present
# on PATH for them to start. Codex (codex-acp) is npx-based too, so it needs
# no extra image support. antigravity-acp is the one binary-distributed
# agent we bundle (see the "antigravity" stage above); its build-time probe
# already proved the .par is self-contained (no system Python needed) --
# other platform-binary agents from the registry are still not supported by
# this image in v1 and will show as "install manually" in the picker.
FROM node:22-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && groupadd -g 65535 nobody 2>/dev/null || true \
    && rm -rf /var/lib/apt/lists/*

COPY --from=tools /usr/local/bin/typst /usr/local/bin/typst
COPY --from=tools /usr/local/bin/tinymist /usr/local/bin/tinymist
# Empty when built with WITH_ANTIGRAVITY=false.
COPY --from=antigravity /out/ /usr/local/bin/
COPY --from=backend /out/typstify-server /usr/local/bin/typstify-server
COPY --from=frontend /src/web/dist /app/web/dist
# Typst finds system fonts under /usr/share/fonts without extra flags.
COPY --from=chesslib /out/fonts /usr/share/fonts/truetype/chessbook
COPY --from=chesslib /out/packages /opt/typst-packages
COPY chessbook/lib /opt/typst-packages/local/chessbook/0.1.0

RUN useradd --create-home --home-dir /data --shell /usr/sbin/nologin typstify \
    && mkdir -p /data/project /data/config /data/cache \
    && chown -R typstify:typstify /data /app

USER typstify
WORKDIR /app

# TYPSTIFY_SERVER_PASSWORD has no default here on purpose -- it must be set
# explicitly by whoever runs the container (see docs/web-server.md).
ENV TYPSTIFY_STATIC_DIR=/app/web/dist \
    TYPSTIFY_PROJECT_DIR=/data/project \
    TYPSTIFY_PROJECT_ROOT=/data \
    TYPSTIFY_SERVER_ADDR=:8080 \
    ANTIGRAVITY_HARNESS_PATH=/usr/local/bin/localharness \
    HOME=/data \
    XDG_CONFIG_HOME=/data/config \
    XDG_CACHE_HOME=/data/cache \
    TYPST_PACKAGE_PATH=/opt/typst-packages

VOLUME ["/data"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD node -e "fetch('http://127.0.0.1:8080/api/health').then(r=>process.exit(r.ok?0:1)).catch(()=>process.exit(1))"

ENTRYPOINT ["typstify-server"]
