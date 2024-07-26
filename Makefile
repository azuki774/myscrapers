SHELL=/bin/bash
VERSION=latest
container_name=myscrapers

.PHONY: bin-linux-amd64

bin-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags "netgo" -installsuffix netgo  -ldflags="-s -w -extldflags \"-static\" \
	-X main.version=$(git describe --tag --abbrev=0) \
	-X main.revision=$(git rev-list -1 HEAD) \
	-X main.build=$(git describe --tags)" \
	-o build/bin/ ./...
