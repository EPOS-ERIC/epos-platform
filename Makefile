HELM_RELEASE ?= epos-platform
HELM_CHART ?= helm/epos-platform
HELM_NAMESPACE ?= epos-platform
HELM_VALUES ?= $(HELM_CHART)/values.yaml
HELM_SET_ARGS ?=
KUBE_CONTEXT ?=

HELM_CONTEXT_ARG := $(if $(KUBE_CONTEXT),--kube-context $(KUBE_CONTEXT),)
HELM_VALUES_ARG := $(if $(HELM_VALUES),--values $(HELM_VALUES),)
TILT_CONTEXT_ARG := $(if $(KUBE_CONTEXT),--context $(KUBE_CONTEXT),)

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
	$(MAKE) -C apps/platform test
	$(MAKE) -C services/gateway test
	$(MAKE) -C services/metadata-lifecycle test
	$(MAKE) -C services/data-retriever test
	$(MAKE) -C services/oss-aai-service test

fmt:
	$(MAKE) -C apps/platform fmt
	$(MAKE) -C services/gateway fmt
	$(MAKE) -C services/data-retriever fmt
	$(MAKE) -C services/oss-aai-service fmt
	$(MAKE) -C services/metadata-lifecycle fmt

lint:
	$(MAKE) -C apps/platform lint
	$(MAKE) -C services/gateway lint
	$(MAKE) -C services/data-retriever lint
	$(MAKE) -C services/oss-aai-service lint
	$(MAKE) -C services/metadata-lifecycle lint

helm-lint:
	helm lint $(HELM_CHART) $(HELM_VALUES_ARG)

helm-template:
	helm template $(HELM_RELEASE) $(HELM_CHART) --namespace $(HELM_NAMESPACE) $(HELM_VALUES_ARG) $(HELM_SET_ARGS)

helm-deploy:
	helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) --namespace $(HELM_NAMESPACE) --create-namespace $(HELM_CONTEXT_ARG) $(HELM_VALUES_ARG) $(HELM_SET_ARGS)

helm-status:
	helm status $(HELM_RELEASE) --namespace $(HELM_NAMESPACE) $(HELM_CONTEXT_ARG)

helm-uninstall:
	helm uninstall $(HELM_RELEASE) --namespace $(HELM_NAMESPACE) $(HELM_CONTEXT_ARG)

test-schemathesis:
# TODO: remove the exclude method flags when the metadata-lifecycle endpoints are implemente
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
		ghcr.io/schemathesis/schemathesis:latest \
		run \
		$$SCHEMATHESIS_EXCLUDE_RESOURCE_PATH \
		--url "http://host.docker.internal:32000/v1" \
		--suppress-health-check=filter_too_much \
		--exclude-checks unsupported_method,missing_required_header,ignored_auth,negative_data_rejection \
		--exclude-method POST \
		--exclude-method PUT \
		--exclude-method PATCH \
		--exclude-method DELETE \
		/work/services/gateway/docs/openapi.yaml

test-hurl:
	docker run --rm \
		--add-host host.docker.internal:host-gateway \
		-v "$(CURDIR)/tests/hurl:/tests:ro" \
		ghcr.io/orange-opensource/hurl:latest \
		--variable baseUrl=http://host.docker.internal:32000 \
		--test /tests

get-dev-token:
	curl -s -X POST http://localhost:8110/oauth/token \
		-u local-dev-client:dev-secret \
		-d grant_type=password \
		-d username=admin@admin.org \
		-d password=adminadmin

.PHONY: up down populate populate-metadatone clean-db-volumes test test-hurl fmt lint helm-lint helm-template helm-deploy helm-status helm-uninstall test-schemathesis get-dev-token
