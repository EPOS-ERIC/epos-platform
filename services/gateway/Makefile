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
	go fix ./...

METADATA_LIFECYCLE_SPEC := ../metadata-lifecycle/docs/openapi.yaml
METADATA_LIFECYCLE_CLIENT_GEN_DIR := gen/metadata-lifecycle
GATEWAY_SERVER_GEN_DIR := gen/gatewayapi

generate: generate-metadata-lifecycle generate-gateway-server

generate-metadata-lifecycle:
	openapi-generator generate -g go \
		-i $(METADATA_LIFECYCLE_SPEC) \
		-o $(METADATA_LIFECYCLE_CLIENT_GEN_DIR) \
		--global-property apiDocs=false,modelDocs=false,apiTests=false,modelTests=false \
		-p packageName=metadatalifecycleclient,withGoMod=false,generateInterfaces=true,hideGenerationTimestamp=true,useOneOfDiscriminatorLookup=true
	(cd $(METADATA_LIFECYCLE_CLIENT_GEN_DIR) && golangci-lint fmt -c ../../.golangci.yaml)

generate-gateway-server:
	go generate .
	(cd $(GATEWAY_SERVER_GEN_DIR) && golangci-lint fmt -c ../../.golangci.yaml)

clean:
	rm -rf $(BIN) $(METADATA_LIFECYCLE_CLIENT_GEN_DIR) $(GATEWAY_SERVER_GEN_DIR)

.PHONY: build build-docker generate generate-metadata-lifecycle generate-gateway-server test fmt lint clean fix
