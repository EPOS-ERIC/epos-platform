IMAGE ?= epos-data-retriever:latest

BIN = data-retriever
BUILDFLAGS = -trimpath

build:
	CGO_ENABLED=0 go build $(BUILDFLAGS) -o $(BIN) .

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

.PHONY: build docker test fmt lint clean
