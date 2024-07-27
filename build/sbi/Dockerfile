FROM golang:1.21.0 as builder

COPY . /app/
WORKDIR /app
RUN go mod download
RUN make bin-linux-amd64

FROM busybox:latest as runner
RUN mkdir -p /usr/local/bin && mkdir -p /data/
COPY --from=builder /app/build/bin/myscrapers /usr/local/bin/myscrapers
COPY --chmod=755 build/sbi/main.sh /usr/local/bin/main.sh
ENTRYPOINT ["/usr/local/bin/main.sh"]
