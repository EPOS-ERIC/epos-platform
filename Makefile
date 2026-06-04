up: 
	tilt up

down:
	tilt down

populate:
	./packages/scripts/populatescript.sh

populate-metadatone:
	./packages/scripts/metadatone.sh

clean: down
	docker volume rm -f docker_neo4j-data docker_user-group-db-data docker_data-retriever-db-data

test:
	$(MAKE) -C services/gateway test
	$(MAKE) -C services/user-group test
	$(MAKE) -C services/data-retriever test
	$(MAKE) -C services/metadata-lifecycle test

test-hurl:
	$(MAKE) -C services/gateway test-hurl

fmt:
	$(MAKE) -C services/gateway fmt
	$(MAKE) -C services/user-group fmt
	$(MAKE) -C services/data-retriever fmt
	$(MAKE) -C services/metadata-lifecycle fmt

lint:
	$(MAKE) -C services/gateway lint
	$(MAKE) -C services/user-group lint
	$(MAKE) -C services/data-retriever lint
	$(MAKE) -C services/metadata-lifecycle lint

test-schemathesis:
	SCHEMATHESIS_HOOKS=tests/schemathesis/hooks.py schemathesis run \
		--url "http://localhost:32000/v1" \
		--suppress-health-check=filter_too_much \
		--exclude-checks unsupported_method \
		services/gateway/docs/openapi.yaml

.PHONY: up down populate populate-metadatone clean-db-volumes test test-hurl fmt lint test-schemathesis
