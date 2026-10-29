BIN = epos-cli
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
EXT ?=
PREFIX ?= /usr/local

BUILDFLAGS = -trimpath
LDFLAGS = -s -w -X github.com/EPOS-ERIC/epos-platform/cli/common.Version=$(VERSION)

build: generate
	CGO_ENABLED=0 go build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) .

# Build for specific platform (used by CI)
build-release:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN)-$(GOOS)-$(GOARCH)$(EXT) .

install: build
	install -d $(PREFIX)/bin
	install -m 755 $(BIN) $(PREFIX)/bin/$(BIN)

clean:
	rm -f $(BIN) $(BIN)-* || true
	rm log.log || true

generate:
	go generate ./...

fmt:
	golangci-lint fmt

lint: fix fmt
	helm lint ./pkg/k8s/config/helm
	golangci-lint run ./...

fix:
	go mod tidy
	go fix ./...

test: vet
	go test ./...

run: clean build
	./$(BIN)

.PHONY: build build-release install clean generate fmt lint vet test run
