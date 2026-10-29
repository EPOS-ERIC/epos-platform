IMAGE ?= epos-data-retriever:latest

BIN = data-retriever
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILDFLAGS = -trimpath
LDFLAGS = -s -w -X main.version=$(VERSION)

build: generate
	CGO_ENABLED=0 go build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) .

generate:
	go generate ./...

dev: generate
	go run .

docker: build
	docker build -t $(IMAGE) -f Dockerfile ../..

test:
	go test ./...

fmt:
	golangci-lint fmt

lint: fmt
	golangci-lint run ./...

clean:
	rm -rf $(BIN)

.PHONY: build docker generate dev test fmt lint clean
