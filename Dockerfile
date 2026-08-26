# syntax=docker/dockerfile:1

# ─────────────────────────────────────────────────────────────────────────────
# Base images, pinned by digest.
#
# A tag is mutable: `ubuntu:26.04` is rebuilt with new package versions, so a
# build that succeeded yesterday can produce a different image today. The digest
# makes the build reproducible and makes a base-image change an explicit commit.
#
# To move to a newer base:
#   docker buildx imagetools inspect ubuntu:26.04
#   docker buildx imagetools inspect golang:1.27-bookworm
# and paste the top-level (index) digest here.
#
# golang 1.27 matches the `go 1.27` directive in go.mod. This previously pinned
# Go 1.24, which cannot compile a module requiring 1.27 — the build was broken.
# ─────────────────────────────────────────────────────────────────────────────
ARG GO_IMAGE=golang:1.27-bookworm@sha256:ded31c68586d2e49e760acc2e65a884b23d032e9bbbed0ae0c55abd3fcaf4452
ARG RUNTIME_IMAGE=ubuntu:26.04@sha256:2260313b31c8c011cd2eebe728008efac1b3982be73eb71348ea2648d2c0e09b


# ─────────────────────────────────────────────────────────────────────────────
# Stage: builder — compiles the binary and nothing else.
# ─────────────────────────────────────────────────────────────────────────────
FROM ${GO_IMAGE} AS builder

WORKDIR /src

# Dependencies before source, so editing a .go file does not invalidate the
# module download layer.
COPY go.mod go.sum ./

# `go mod download` rather than the `go mod tidy` this used to run at build time.
# tidy rewrites go.mod and go.sum from whatever the network currently resolves,
# which makes the build non-hermetic and can silently change the dependency graph
# between two builds of the same commit.
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# CGO off for a static binary the runtime stage needs no libc matching for.
# -trimpath strips local filesystem paths out of the binary; -s -w drops the
# symbol table and DWARF data. -mod=readonly fails the build if the source needs
# a dependency go.mod does not already record.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
        -mod=readonly \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/web-app \
        ./main.go


# ─────────────────────────────────────────────────────────────────────────────
# Stage: runtime-base — the OS layer both final stages share.
# ─────────────────────────────────────────────────────────────────────────────
FROM ${RUNTIME_IMAGE} AS runtime-base

ARG WWWGROUP=1000
ARG WWWUSER=1000
ARG TZ=UTC

ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=${TZ}
ENV SUPERVISOR_GO_COMMAND="/opt/go-bins/web-app http"
ENV SUPERVISOR_GO_USER="app"
ENV PGSSLCERT=/tmp/postgresql.crt

# Only what the running service needs: supervisor to manage it, gosu to drop
# privileges, ca-certificates for outbound TLS, tzdata for the timezone,
# postgresql-client and curl for operational access and the healthcheck.
#
# postgresql-client comes from Ubuntu's own archive rather than the PGDG repo the
# previous version added. PGDG suites are keyed to a release codename, so pointing
# at it hardcodes an Ubuntu version into an unrelated line; the client is only
# used for manual inspection, and schema changes run through the Go binary's
# migrate command.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        gosu \
        postgresql-client \
        supervisor \
        tzdata \
    && ln -snf /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo ${TZ} > /etc/timezone \
    && apt-get -y autoremove \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# The default unprivileged account Ubuntu ships would collide on uid 1000.
RUN userdel -r ubuntu 2>/dev/null || true

# uid matches WWWUSER, so start-container.sh does not have to usermod at boot and
# bind-mounted files keep the host owner's identity. The old image created this
# user as uid 1337 while compose passed WWWUSER=1000, so every start rewrote it.
RUN groupadd --force -g ${WWWGROUP} app \
    && useradd -ms /bin/bash --no-user-group -g ${WWWGROUP} -u ${WWWUSER} app

COPY start-container.sh /usr/local/bin/start-container.sh
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

RUN chmod +x /usr/local/bin/start-container.sh \
    && mkdir -p /opt/go-bins \
    && chown -R app:app /opt/go-bins

WORKDIR /web-app

EXPOSE 8000/tcp

ENTRYPOINT ["start-container.sh"]


# ─────────────────────────────────────────────────────────────────────────────
# Stage: development — adds the Go toolchain for the compose workflow.
#
# docker-compose bind-mounts the source over /web-app and caches modules in a
# volume, so `go run main.go test` and rebuilds happen inside the container. That
# needs a toolchain, which the production image deliberately does not ship.
# ─────────────────────────────────────────────────────────────────────────────
FROM runtime-base AS development

# The whole toolchain, copied rather than downloaded: the version is then
# guaranteed to match the one the binary was built with.
COPY --from=builder /usr/local/go /usr/local/go

ENV PATH=$PATH:/usr/local/go/bin
ENV GOCACHE=/var/tmp/go-cache
ENV GOPATH=/web-app/go
ENV GOMODCACHE=/web-app/go/pkg/mod
ENV GOBIN=/web-app/go/bin

RUN apt-get update \
    && apt-get install -y --no-install-recommends git \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /var/tmp/go-cache /web-app/go/pkg/mod \
    && chown -R app:app /var/tmp/go-cache /web-app

COPY --from=builder /out/web-app /opt/go-bins/web-app


# ─────────────────────────────────────────────────────────────────────────────
# Stage: production — the default target. Binary only.
#
# Last stage on purpose, so a plain `docker build` produces the deployable image.
# No toolchain, no module cache, no source tree: the previous single-stage image
# shipped all three, along with anything else `COPY . .` picked up.
# ─────────────────────────────────────────────────────────────────────────────
FROM runtime-base AS production

LABEL maintainer="Islam Samy"

# Build metadata. Supplied by CI; --build-arg so the labels are accurate rather
# than hardcoded to a stale value.
ARG IMAGE_VERSION="dev"
ARG IMAGE_REVISION="unknown"
ARG IMAGE_CREATED="unknown"

LABEL org.opencontainers.image.title="web-app" \
      org.opencontainers.image.description="Gin web application with Laravel-style structure" \
      org.opencontainers.image.source="https://github.com/islamsamy214/gin-di" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${IMAGE_VERSION}" \
      org.opencontainers.image.revision="${IMAGE_REVISION}" \
      org.opencontainers.image.created="${IMAGE_CREATED}"

COPY --from=builder /out/web-app /opt/go-bins/web-app

# The application writes its log file relative to the working directory, and runs
# as an unprivileged user, so the directory has to exist and be writable.
RUN mkdir -p /web-app/storage/logs /web-app/storage/app \
    && chown -R app:app /web-app

# GET / is unversioned and needs no authentication, which is what makes it usable
# here. start-period covers the database ping in Boot.
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -fsS http://127.0.0.1:8000/ >/dev/null || exit 1

# No USER directive, deliberately.
#
# The entrypoint has to start as root: it reconciles the app uid with the host's
# and then drops privileges itself, either through gosu or through supervisor's
# `user=` directive. Setting USER app here would leave it unable to do either and
# supervisord unable to start. The served process still runs as app — see
# SUPERVISOR_GO_USER and supervisord.conf.
