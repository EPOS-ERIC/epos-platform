IMAGE ?= epos-converter:latest

BIN = converter
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILDFLAGS = -trimpath
LDFLAGS = -s -w -X main.version=$(VERSION)

build: generate
	CGO_ENABLED=0 go build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) .

docker: build
	docker build -t $(IMAGE) -f Dockerfile ../..

test:
	go test ./...

dev: generate
	go run .

fix:
	go mod tidy
	go fix ./...

generate:
	go generate ./...

fmt:
	golangci-lint fmt

lint: fix fmt
	golangci-lint run ./...

.PHONY: build docker generate test fmt lint fix dev
