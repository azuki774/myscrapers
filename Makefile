SHELL=/bin/bash
VERSION=latest
container_name=myscrapers

.PHONY: build bin-linux-amd64 start stop debug setup lint test

bin-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags "netgo" -installsuffix netgo  -ldflags="-s -w -extldflags \"-static\" \
	-X main.version=$(git describe --tag --abbrev=0) \
	-X main.revision=$(git rev-list -1 HEAD) \
	-X main.build=$(git describe --tags)" \
	-o build/bin/ ./...

build:
	docker build -t $(container_name):$(VERSION) -f build/sbi/Dockerfile .

start:
	docker compose -f deployment/compose.yml up -d

stop:
	docker compose -f deployment/compose.yml down

debug:
	docker compose -f deployment/compose.yml up

clean:
	sudo rm -rf deployment/browser/*
	touch deployment/browser/.gitkeep

setup:
	go mod download
	go mod tidy
	go install honnef.co/go/tools/cmd/staticcheck@latest

lint:
	(! gofmt -s -d . | grep '^')
	go vet ./...
	staticcheck ./...

test:
	go test -v ./...
