GATEWAY_DIR := services/gateway
METADATA_LIFECYCLE_DIR := services/metadata-lifecycle
DATA_RETRIEVER_DIR := services/data-retriever
OSS_AAI_SERVICE_DIR := services/oss-aai-service

up: 
	tilt up $(TILT_CONTEXT_ARG)

down:
	tilt down $(TILT_CONTEXT_ARG)

clean: down
	docker volume rm -f docker_neo4j-data docker_data-retriever-db-data docker_oss-aai-data docker_metadata-lifecycle-db-data

populate:
	./packages/scripts/populatescript.sh

populate-metadatone:
	./packages/scripts/metadatone.sh

ci:
	+$(MAKE) -j ci-gateway-image ci-metadata-lifecycle-image ci-data-retriever-image ci-oss-aai-image
	docker compose -f docker/compose.ci.yaml up -d --wait gateway-service metadata-lifecycle data-retriever oss-aai-service
ci-gateway-image:
	+$(MAKE) -C $(GATEWAY_DIR) docker
ci-metadata-lifecycle-image:
	+$(MAKE) -C $(METADATA_LIFECYCLE_DIR) docker
ci-data-retriever-image:
	+$(MAKE) -C $(DATA_RETRIEVER_DIR) docker
ci-oss-aai-image:
	+$(MAKE) -C $(OSS_AAI_SERVICE_DIR) docker

test:
	+$(MAKE) -j test-gateway test-metadata-lifecycle test-data-retriever test-oss-aai-service
test-gateway:
	$(MAKE) -C $(GATEWAY_DIR) test
test-metadata-lifecycle:
	$(MAKE) -C $(METADATA_LIFECYCLE_DIR) test
test-data-retriever:
	$(MAKE) -C $(DATA_RETRIEVER_DIR) test
test-oss-aai-service:
	$(MAKE) -C $(OSS_AAI_SERVICE_DIR) test

lint:
	+$(MAKE) -j lint-gateway lint-data-retriever lint-oss-aai-service lint-metadata-lifecycle
lint-gateway:
	$(MAKE) -C $(GATEWAY_DIR) lint
lint-data-retriever:
	$(MAKE) -C $(DATA_RETRIEVER_DIR) lint
lint-oss-aai-service:
	$(MAKE) -C $(OSS_AAI_SERVICE_DIR) lint
lint-metadata-lifecycle:
	$(MAKE) -C $(METADATA_LIFECYCLE_DIR) lint

generate:
	+$(MAKE) -j generate-gateway generate-metadata-lifecycle generate-protobuf
generate-gateway:
	$(MAKE) -C $(GATEWAY_DIR) generate
generate-metadata-lifecycle:
	$(MAKE) -C $(METADATA_LIFECYCLE_DIR) generate
generate-protobuf:
	$(MAKE) -C packages/protobuf generate

test-integration:
	docker run --rm \
		--add-host host.docker.internal:host-gateway \
		-v "$(CURDIR):/work:ro" \
		-w /work \
		-e GATEWAY_API_URL=http://host.docker.internal:32000/v1 \
		-e AAI_URL=http://host.docker.internal:8110 \
		--entrypoint python \
		ghcr.io/schemathesis/schemathesis:latest \
		-m unittest discover -s /work/tests/contract -p 'test_*.py' -v

get-dev-token:
	curl -s -X POST http://localhost:8110/oauth/token \
		-u local-dev-client:dev-secret \
		-d grant_type=password \
		-d username=admin@admin.org \
		-d password=adminadmin

.PHONY: up ci ci-gateway-image ci-metadata-lifecycle-image ci-data-retriever-image ci-oss-aai-image down populate populate-metadatone clean test test-gateway test-metadata-lifecycle test-data-retriever test-oss-aai-service lint lint-gateway lint-data-retriever lint-oss-aai-service lint-metadata-lifecycle test-integration get-dev-token generate generate-gateway generate-metadata-lifecycle generate-protobuf
