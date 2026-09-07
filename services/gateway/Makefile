IMAGE ?= epos-gateway:latest

BIN = gateway
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILDFLAGS=-trimpath

# TODO
# LDFLAGS=-s -w -X github.com/EPOS-ERIC//common.Version=$(VERSION)

build:
	CGO_ENABLED=0 go build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) .

build-docker: build
	docker build -t $(IMAGE) -f Dockerfile ../..

test:
	go test ./... -race -count=1

fmt:
	golangci-lint fmt

lint: fix fmt
	golangci-lint run ./...

fix:
	go mod tidy
	go fix ./...

generate:
	go generate .

GATEWAY_SERVER_GEN_DIR := gen/gatewayapi
clean:
	rm -rf $(BIN) $(GATEWAY_SERVER_GEN_DIR)

.PHONY: build build-docker generate test fmt lint clean fix
