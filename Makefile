SHELL=/bin/bash
VERSION=latest
container_name=myscrapers

.PHONY: build start stop debug

build:
	docker build -t $(container_name):$(VERSION) -f build/Dockerfile .

start:
	docker compose -f deployment/compose.yml up -d

stop:
	docker compose -f deployment/compose.yml down

debug:
	docker compose -f deployment/compose.yml up

clean:
	sudo rm -rf deployment/browser/*
	touch deployment/browser/.gitkeep
