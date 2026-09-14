# This suite fuzzes schema contracts only.
SCHEMATHESIS_EXCLUDE_PATHS := \
	--exclude-path /search \
	--exclude-path /imports/turtle \
	--exclude-path /groups/{groupId}/membership-requests \
	--exclude-path /groups/{groupId}/memberships/{userId} \
	--exclude-path /groups/{groupId}/memberships/{userId}/roles/{role} \
	--exclude-path /groups/{groupId}/memberships/{userId}/permissions/{permission} \
	--exclude-path /groups/{groupId}/role-requests \
	--exclude-path /groups/{groupId}/role-requests/{requestId} \
	--exclude-path /groups/{groupId}/resources \
	--exclude-path /groups/{groupId}/resources/{resourceType}/{id}/copy \
	--exclude-path /resources/{resourceType}/{id}/versions/{versionId}/submit \
	--exclude-path /resources/{resourceType}/{id}/versions/{versionId}/withdraw \
	--exclude-path /resources/{resourceType}/{id}/versions/{versionId}/reject \
	--exclude-path /resources/{resourceType}/{id}/versions/{versionId}/reopen \
	--exclude-path /resources/{resourceType}/{id}/versions/{versionId}/publish \
	--exclude-path /resources/{resourceType}/{id}/versions/{versionId}/unpublish \
	--exclude-path /resources/{resourceType}/{id}/draft-from-publication \
	--exclude-path /resources/{resourceType}/{id}/draft-from-archive \
	--exclude-path /users/{id}/block \
	--exclude-path /users/{id}/unblock \
	--exclude-path /users/{id}/global-roles/{role} \
	--exclude-path /users/{id}/global-permissions/{permission}

up: 
	tilt up $(TILT_CONTEXT_ARG)

down:
	tilt down $(TILT_CONTEXT_ARG)

populate:
	./packages/scripts/populatescript.sh

populate-metadatone:
	./packages/scripts/metadatone.sh

clean: down
	docker volume rm -f docker_neo4j-data docker_data-retriever-db-data docker_oss-aai-data docker_metadata-lifecycle-db-data

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

test-schemathesis:
	schemathesis_env="$$(./packages/scripts/schemathesis-env.sh)" && eval "$$schemathesis_env" && docker run --rm \
		--add-host host.docker.internal:host-gateway \
		-v "$(CURDIR):/work:ro" \
		-w /tmp \
		-e HYPOTHESIS_HOME=/tmp/.hypothesis \
		-e SCHEMATHESIS_HOOKS=/work/tests/schemathesis/hooks.py \
		-e SCHEMATHESIS_BEARER_TOKEN="$$SCHEMATHESIS_BEARER_TOKEN" \
		-e SCHEMATHESIS_GROUP_ID="$$SCHEMATHESIS_GROUP_ID" \
		-e SCHEMATHESIS_CATEGORY_ID="$$SCHEMATHESIS_CATEGORY_ID" \
		-e SCHEMATHESIS_RESOURCE_ID="$$SCHEMATHESIS_RESOURCE_ID" \
		-e SCHEMATHESIS_RESOURCE_VERSION_ID="$$SCHEMATHESIS_RESOURCE_VERSION_ID" \
		-e SCHEMATHESIS_PROVIDER_ID="$$SCHEMATHESIS_PROVIDER_ID" \
		-e SCHEMATHESIS_CONTACT_POINT_ID="$$SCHEMATHESIS_CONTACT_POINT_ID" \
		-e SCHEMATHESIS_ORGANIZATION_ID="$$SCHEMATHESIS_ORGANIZATION_ID" \
		-e SCHEMATHESIS_USER_ID="$$SCHEMATHESIS_USER_ID" \
		-e SCHEMATHESIS_MEMBERSHIP_USER_ID="$$SCHEMATHESIS_MEMBERSHIP_USER_ID" \
		-e SCHEMATHESIS_ROLE_REQUEST_ID="$$SCHEMATHESIS_ROLE_REQUEST_ID" \
		-e SCHEMATHESIS_DRAFT_RESOURCE_ID="$$SCHEMATHESIS_DRAFT_RESOURCE_ID" \
		-e SCHEMATHESIS_DRAFT_RESOURCE_VERSION_ID="$$SCHEMATHESIS_DRAFT_RESOURCE_VERSION_ID" \
		-e SCHEMATHESIS_PUBLISHED_RESOURCE_ID="$$SCHEMATHESIS_PUBLISHED_RESOURCE_ID" \
		-e SCHEMATHESIS_PUBLISHED_RESOURCE_VERSION_ID="$$SCHEMATHESIS_PUBLISHED_RESOURCE_VERSION_ID" \
		-e SCHEMATHESIS_ARCHIVED_RESOURCE_ID="$$SCHEMATHESIS_ARCHIVED_RESOURCE_ID" \
		-e SCHEMATHESIS_ARCHIVED_RESOURCE_VERSION_ID="$$SCHEMATHESIS_ARCHIVED_RESOURCE_VERSION_ID" \
		ghcr.io/schemathesis/schemathesis:latest \
		run \
		$$SCHEMATHESIS_EXCLUDE_RESOURCE_PATH \
		$(SCHEMATHESIS_EXCLUDE_PATHS) \
		--exclude-method DELETE \
		--exclude-method PUT \
		--url "http://host.docker.internal:32000/v1" \
		--phases examples,coverage,fuzzing \
		--suppress-health-check=filter_too_much \
		--exclude-checks unsupported_method,missing_required_header,ignored_auth,negative_data_rejection \
		--generation-deterministic \
		--generation-allow-x00 false \
		/work/services/gateway/docs/openapi.yaml

test-contract:
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

.PHONY: up down populate populate-metadatone clean test test-gateway test-metadata-lifecycle test-data-retriever test-oss-aai-service lint lint-gateway lint-data-retriever lint-oss-aai-service lint-metadata-lifecycle test-schemathesis test-contract get-dev-token
