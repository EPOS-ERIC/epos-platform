BIN = gateway
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILDFLAGS=-trimpath

# TODO
# LDFLAGS=-s -w -X github.com/EPOS-ERIC//common.Version=$(VERSION)

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) .

test:
	go test ./...

fmt:
	golangci-lint fmt

lint: fmt
	golangci-lint run ./...

GATEWAY_SPEC := swagger/openapi.yaml
SERVER_GEN_DIR := gen/gatewayserver
METADATA_SPEC := ../metadata-lifecycle/openapi.yaml
METADATA_CLIENT_GEN_DIR := gen/metadataclient
generate: clean
	openapi-generator generate -g go-server \
		-i $(GATEWAY_SPEC) \
		-o $(SERVER_GEN_DIR) \
		-p packageName=gatewayserver,router=chi,outputAsLibrary=true,onlyInterfaces=true,hideGenerationTimestamp=true,sourceFolder=.
	openapi-generator generate -g go \
		-i $(METADATA_SPEC) \
		-o $(METADATA_CLIENT_GEN_DIR) \
		--global-property apiDocs=false,modelDocs=false,apiTests=false,modelTests=false \
		-p packageName=metadataclient,withGoMod=false,generateInterfaces=true,hideGenerationTimestamp=true,useOneOfDiscriminatorLookup=true
    # Clean up wrong generated imports and format
	(cd $(SERVER_GEN_DIR) && golangci-lint fmt -c ../../.golangci.yaml)
	(cd $(METADATA_CLIENT_GEN_DIR) && golangci-lint fmt -c ../../.golangci.yaml)

clean:
	rm -rf $(BIN) $(SERVER_GEN_DIR) $(METADATA_CLIENT_GEN_DIR)

.PHONY: build generate test fmt lint clean
