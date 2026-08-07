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

test-fixtures-refresh:
	go run ./scripts/refresh_test_fixtures.go

fmt:
	golangci-lint fmt

lint: fmt
	golangci-lint run ./...

METADATA_SPEC := ../metadata-lifecycle/openapi.yaml
METADATA_CLIENT_GEN_DIR := gen/metadataclient

BACKOFFICE_SPEC := ../backoffice/docs/openapi.yaml
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
		-i $(BACKOFFICE_SPEC) \
		-o $(USERGROUP_CLIENT_GEN_DIR) \
		--global-property apiDocs=false,modelDocs=false,apiTests=false,modelTests=false \
		-p packageName=usergroupclient,withGoMod=false,generateInterfaces=true,hideGenerationTimestamp=true,useOneOfDiscriminatorLookup=true
	(cd $(USERGROUP_CLIENT_GEN_DIR) && golangci-lint fmt -c ../../.golangci.yaml)

clean:
	rm -rf $(BIN) $(METADATA_CLIENT_GEN_DIR) $(USERGROUP_CLIENT_GEN_DIR)

get-token:
	curl -s -X POST http://localhost:8110/oauth/token \
		-u local-dev-client:dev-secret \
		-d grant_type=password \
		-d username=admin@admin.org \
		-d password=adminadmin


.PHONY: build build-docker generate test test-hurl test-fixtures-refresh fmt lint clean
