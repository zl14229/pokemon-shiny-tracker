.PHONY: deps fmt test build check smoke run

deps:
	go mod download
	go mod verify

fmt:
	gofmt -w cmd internal

test:
	go test ./...

build:
	go build ./...
	go build -o bin/shiny-tracker ./cmd/server

check:
	test -z "$$(gofmt -l cmd internal)"
	go mod tidy -diff
	go mod verify
	go vet ./...
	go test -race ./...
	go build ./...

smoke: build
	python3 scripts/smoke_backend.py
	python3 scripts/smoke_records.py

run:
	go run ./cmd/server
