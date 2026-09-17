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
	+$(MAKE) -C services/gateway docker
	+$(MAKE) -C services/metadata-lifecycle docker
	+$(MAKE) -C services/oss-aai-service docker
	docker compose -f docker/compose.ci.yaml up -d --wait gateway-service metadata-lifecycle oss-aai-service

test:
	+$(MAKE) -j test-gateway test-metadata-lifecycle test-data-retriever test-oss-aai-service
test-gateway:
	$(MAKE) -C services/gateway test
test-metadata-lifecycle:
	$(MAKE) -C services/metadata-lifecycle test
test-data-retriever:
	$(MAKE) -C services/data-retriever test
test-oss-aai-service:
	$(MAKE) -C services/oss-aai-service test

lint:
	+$(MAKE) -j lint-gateway lint-data-retriever lint-oss-aai-service lint-metadata-lifecycle
lint-gateway:
	$(MAKE) -C services/gateway lint
lint-data-retriever:
	$(MAKE) -C services/data-retriever lint
lint-oss-aai-service:
	$(MAKE) -C services/oss-aai-service lint
lint-metadata-lifecycle:
	$(MAKE) -C services/metadata-lifecycle lint

generate:
	+$(MAKE) -j generate-gateway generate-metadata-lifecycle
generate-gateway:
	$(MAKE) -C services/gateway generate
generate-metadata-lifecycle:
	$(MAKE) -C services/metadata-lifecycle generate

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

.PHONY: up ci down populate populate-metadatone clean test test-gateway test-metadata-lifecycle test-data-retriever test-oss-aai-service lint lint-gateway lint-data-retriever lint-oss-aai-service lint-metadata-lifecycle test-integration get-dev-token generate generate-gateway generate-metadata-lifecycle
