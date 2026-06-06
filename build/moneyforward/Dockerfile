# syntax=docker/dockerfile:1.7

FROM golang:1.25-bookworm AS builder
WORKDIR /src
COPY myscraper/go.mod myscraper/go.sum ./
RUN go mod download
COPY myscraper/ ./
RUN CGO_ENABLED=0 go build -o /out/myscraper ./cmd/myscraper

FROM debian:bookworm-slim

# Install Chrome (matches the prior Python image so Playwright can drive
# the same binary; the dev shell uses `chromium` from Nix but the
# container image installs google-chrome-stable so the same Selenium
# tooling continues to work).
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      gnupg \
      libglib2.0-0 \
      libnss3 \
      libgconf-2-4 \
      libfontconfig1 \
      wget && \
    rm -rf /var/lib/apt/lists/*
RUN wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add - && \
    echo "deb http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends google-chrome-stable && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/myscraper /usr/local/bin/myscraper
RUN mkdir -p /data
ENTRYPOINT ["myscraper"]
CMD ["moneyforward", "--fetch", "--s3-upload"]
