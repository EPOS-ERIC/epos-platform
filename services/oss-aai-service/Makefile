IMAGE ?= epos-oss-aai-service:latest

BIN = bin/oss-aai-service
BUILDFLAGS = -trimpath

build:
	CGO_ENABLED=0 go build $(BUILDFLAGS) -o $(BIN) .

docker:
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
