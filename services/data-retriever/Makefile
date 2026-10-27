IMAGE ?= epos-data-retriever:latest

BIN = data-retriever
BUILDFLAGS = -trimpath

build: generate
	CGO_ENABLED=0 go build $(BUILDFLAGS) -o $(BIN) .

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
