IMAGE ?= backoffice:latest

BIN = backoffice
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILDFLAGS=-trimpath

# TODO
# LDFLAGS=-s -w -X github.com/EPOS-ERIC//common.Version=$(VERSION)

build: generate
	CGO_ENABLED=0 go build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) .

build-docker: build
	docker build -t $(IMAGE) -f Dockerfile ../..

test:
	go test ./...

generate:
	go generate ./...

fmt:
	golangci-lint fmt

lint: fmt
	golangci-lint run ./...

.PHONY: build build-docker generate test fmt lint
