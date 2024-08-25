FROM golang:1.21.0 as builder

COPY . /app/
WORKDIR /app

# Go build
RUN go mod download
RUN make bin-linux-amd64

FROM debian:bookworm-slim as runner
# Required Packages
RUN apt-get update && \
    apt-get install -y curl unzip && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# AWS Setup
RUN curl -o /var/tmp/awscli.zip https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip && \
    unzip -d /usr/local/bin/ /var/tmp/awscli.zip

RUN mkdir -p /usr/local/bin && mkdir -p /data/
COPY --from=builder /app/build/bin/myscrapers /usr/local/bin/myscrapers
COPY --chmod=755 build/moneyforward/main.sh /usr/local/bin/main.sh
ENTRYPOINT ["/usr/local/bin/main.sh"]
