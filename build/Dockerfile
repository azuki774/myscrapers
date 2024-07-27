FROM golang:1.21.0 as builder

COPY . /app/
WORKDIR /app
RUN go mod tidy && make bin-linux-amd64

FROM busybox:latest as runner
RUN mkdir -p /usr/local/bin && mkdir -p /data/
COPY --from=builder /app/build/bin/myscrapers /usr/local/bin/myscrapers
CMD [ "/usr/local/bin/myscrapers", "download" ,"test-github"]
