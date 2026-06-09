# syntax=docker/dockerfile:1.7

# Base images are pinned to immutable digests for reproducible builds.
# Refresh via:
#   token=$(curl -fsS "https://auth.docker.io/token?service=registry.docker.io&scope=repository:library/<image>:pull" | jq -r .token)
#   curl -fsS -D - -o /dev/null \
#     "https://registry-1.docker.io/v2/library/<image>/manifests/<tag>" \
#     -H "Authorization: Bearer ${token}" \
#     -H 'Accept: application/vnd.docker.distribution.manifest.list.v2+json' \
#     | grep -i docker-content-digest
ARG GO_BASE=golang:1.25-bookworm@sha256:a1ae6b6c564f3e0072d70081036827a2705dbcf6b38aaa6d97f5de97fe9abdb4
ARG RUNTIME_BASE=debian:bookworm-slim@sha256:0104b334637a5f19aa9c983a91b54c89887c0984081f2068983107a6f6c21eeb

FROM ${GO_BASE} AS builder
WORKDIR /src

# Cache module download layer separately from source.
COPY myscraper/go.mod myscraper/go.sum ./
RUN go mod download

COPY myscraper/ ./
# Reproducible Go binary: strip host paths and embed no VCS info.
# Override with --build-arg SOURCE_DATE_EPOCH=$(git log -1 --format=%ct)
# at build time for bit-for-bit identical image layers.
ARG SOURCE_DATE_EPOCH=0
ENV SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH}
RUN CGO_ENABLED=0 \
    go build \
      -trimpath \
      -buildvcs=false \
      -ldflags=-buildid= \
      -o /out/myscraper ./cmd/myscraper

# Pre-download the Playwright Node driver so the runtime image does not
# need Go or network access to install it. The driver version must match
# playwright-go's expectation (v1.52.0 for playwright-go v0.5200.1).
ENV PLAYWRIGHT_DRIVER_PATH=/out/playwright-driver
RUN go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1 install

FROM ${RUNTIME_BASE}

# Debian's `chromium` is on PATH and is the first match in
# myscraper/internal/chromium/exec.go's lookup, so no third-party
# apt source or apt-key is required. The apt version is intentionally
# NOT pinned so `apt-get upgrade` can roll forward with Debian
# security updates; only the base image is digest-pinned.
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      ca-certificates \
      chromium \
      fonts-noto-cjk \
      libglib2.0-0 \
      libnss3 \
      libfontconfig1 && \
    rm -rf /var/lib/apt/lists/*

ENV PLAYWRIGHT_DRIVER_PATH=/playwright/.driver
COPY --from=builder /out/playwright-driver ${PLAYWRIGHT_DRIVER_PATH}
RUN chmod +x ${PLAYWRIGHT_DRIVER_PATH}/node

RUN useradd --system --uid 10001 --create-home --home-dir /home/myscraper myscraper
WORKDIR /home/myscraper

COPY --from=builder /out/myscraper /usr/local/bin/myscraper
RUN mkdir -p /data && chown -R myscraper:myscraper /data
VOLUME ["/data"]
ENV TZ=Asia/Tokyo

USER myscraper
ENTRYPOINT ["myscraper"]
CMD ["moneyforward", "--fetch", "--s3-upload"]
