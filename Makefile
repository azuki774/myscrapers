SHELL=/bin/bash
VERSION=latest
container_name_sbi=myscrapers-sbi
container_name_mf=myscrapers-mf

.PHONY: build bin-linux-amd64 start stop debug setup lint test

bin-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags "netgo" -installsuffix netgo  -ldflags="-s -w -extldflags \"-static\" \
	-X main.version=$(git describe --tag --abbrev=0) \
	-X main.revision=$(git rev-list -1 HEAD) \
	-X main.build=$(git describe --tags)" \
	-o build/bin/ ./...

build:
	docker build -t $(container_name_sbi):$(VERSION) -f build/sbi/Dockerfile .
	docker build -t $(container_name_mf):$(VERSION) -f build/moneyforward/Dockerfile .

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
