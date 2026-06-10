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
	go test ./...

test-fixtures-refresh:
	go run ./scripts/refresh_test_fixtures.go

fmt:
	golangci-lint fmt

lint: fmt
	golangci-lint run ./...

METADATA_SPEC := ../metadata-lifecycle/openapi.yaml
METADATA_CLIENT_GEN_DIR := gen/metadataclient

USERGROUP_SPEC := ../user-group/swagger/openapi.yaml
USERGROUP_CLIENT_GEN_DIR := gen/usergroupclient
generate: clean
	openapi-generator generate -g go \
		-i $(METADATA_SPEC) \
		-o $(METADATA_CLIENT_GEN_DIR) \
		--global-property apiDocs=false,modelDocs=false,apiTests=false,modelTests=false \
		-p packageName=metadataclient,withGoMod=false,generateInterfaces=true,hideGenerationTimestamp=true,useOneOfDiscriminatorLookup=true
	# Format generated metadata client
	(cd $(METADATA_CLIENT_GEN_DIR) && golangci-lint fmt -c ../../.golangci.yaml)

	openapi-generator generate -g go \
		-i $(USERGROUP_SPEC) \
		-o $(USERGROUP_CLIENT_GEN_DIR) \
		--global-property apiDocs=false,modelDocs=false,apiTests=false,modelTests=false \
		-p packageName=usergroupclient,withGoMod=false,generateInterfaces=true,hideGenerationTimestamp=true,useOneOfDiscriminatorLookup=true
	(cd $(USERGROUP_CLIENT_GEN_DIR) && golangci-lint fmt -c ../../.golangci.yaml)

clean:
	rm -rf $(BIN) $(METADATA_CLIENT_GEN_DIR)

.PHONY: build build-docker generate test test-hurl test-fixtures-refresh fmt lint clean
