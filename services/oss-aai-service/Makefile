IMAGE ?= epos-oss-aai-service:latest

BIN = bin/oss-aai-service
BUILDFLAGS = -trimpath

build:
	CGO_ENABLED=0 go build $(BUILDFLAGS) -o $(BIN) .

build-docker:
	docker build -t $(IMAGE) .

run:
	go run .

test:
	go test ./...

fmt:
	gofmt -w .

lint: fmt
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin

.PHONY: build build-docker run test fmt lint tidy clean
