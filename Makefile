SHELL=/bin/bash
VERSION=latest
container_name_sbi=myscrapers-sbi
container_name_mf=myscrapers-mf

.PHONY: build start stop debug

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
