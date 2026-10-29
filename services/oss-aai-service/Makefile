IMAGE ?= epos-oss-aai-service:latest

BIN = bin/oss-aai-service
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILDFLAGS = -trimpath
LDFLAGS = -s -w -X main.version=$(VERSION)

build:
	CGO_ENABLED=0 go build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) .

docker: build
	docker build -t $(IMAGE) -f Dockerfile ../..

run:
	go run .

test:
	go test ./...

fmt:
	golangci-lint fmt

lint: fmt
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin

.PHONY: build docker run test fmt lint tidy clean
