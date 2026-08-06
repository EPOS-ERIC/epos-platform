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
	docker volume rm -f docker_neo4j-data docker_data-retriever-db-data docker_oss-aai-data

test:
	$(MAKE) -C apps/platform test
	$(MAKE) -C services/gateway test
	$(MAKE) -C services/backoffice test
	$(MAKE) -C services/data-retriever test
	$(MAKE) -C services/metadata-lifecycle test
	$(MAKE) -C services/oss-aai-service test

fmt:
	$(MAKE) -C apps/platform fmt
	$(MAKE) -C services/gateway fmt
	$(MAKE) -C services/data-retriever fmt
	$(MAKE) -C services/metadata-lifecycle fmt
	$(MAKE) -C services/oss-aai-service fmt
	$(MAKE) -C services/backoffice fmt

lint:
	$(MAKE) -C apps/platform lint
	$(MAKE) -C services/gateway lint
	$(MAKE) -C services/data-retriever lint
	$(MAKE) -C services/metadata-lifecycle lint
	$(MAKE) -C services/oss-aai-service lint
	$(MAKE) -C services/backoffice lint

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
# TODO: remove the exclude method flags when the backoffice endpoints are implemente
	eval "$$(./packages/scripts/schemathesis-env.sh)" && docker run --rm \
		--add-host host.docker.internal:host-gateway \
		-v "$(CURDIR):/work:ro" \
		-w /tmp \
		-e HYPOTHESIS_HOME=/tmp/.hypothesis \
		-e SCHEMATHESIS_HOOKS=/work/tests/schemathesis/hooks.py \
		-e SCHEMATHESIS_BEARER_TOKEN="$$SCHEMATHESIS_BEARER_TOKEN" \
		-e SCHEMATHESIS_GROUP_ID="$$SCHEMATHESIS_GROUP_ID" \
		-e SCHEMATHESIS_MEMBERSHIP_USER_ID="$$SCHEMATHESIS_MEMBERSHIP_USER_ID" \
		ghcr.io/schemathesis/schemathesis:latest \
		run \
		--url "http://host.docker.internal:32000/v1" \
		--suppress-health-check=filter_too_much \
		--exclude-checks unsupported_method,missing_required_header,ignored_auth \
		--exclude-method POST \
		--exclude-method PUT \
		--exclude-method DELETE \
		/work/services/gateway/docs/openapi.yaml

test-hurl:
	docker run --rm \
		--add-host host.docker.internal:host-gateway \
		-v "$(CURDIR)/tests/hurl:/tests:ro" \
		ghcr.io/orange-opensource/hurl:latest \
		--variable baseUrl=http://host.docker.internal:32000 \
		--test /tests

test-hurl-groups:
	eval "$$(./packages/scripts/hurl-groups-authz-env.sh)" && docker run --rm \
		--add-host host.docker.internal:host-gateway \
		-v "$(CURDIR)/tests/hurl-groups-authz:/tests:ro" \
		ghcr.io/orange-opensource/hurl:latest \
		--variable baseUrl=http://host.docker.internal:32000 \
		--variable adminBearerToken="$$HURL_ADMIN_BEARER_TOKEN" \
		--variable adminUserId="$$HURL_ADMIN_USER_ID" \
		--variable userBearerToken="$$HURL_USER_BEARER_TOKEN" \
		--variable userId="$$HURL_USER_ID" \
		--variable groupAdminBearerToken="$$HURL_GROUP_ADMIN_BEARER_TOKEN" \
		--variable groupAdminUserId="$$HURL_GROUP_ADMIN_USER_ID" \
		--variable groupAId="$$HURL_GROUP_A_ID" \
		--variable groupASlug="$$HURL_GROUP_A_SLUG" \
		--variable groupBId="$$HURL_GROUP_B_ID" \
		--variable groupBSlug="$$HURL_GROUP_B_SLUG" \
		--variable adminLifecycleGroupSlug="$$HURL_ADMIN_LIFECYCLE_GROUP_SLUG" \
		--test /tests

.PHONY: up down populate populate-metadatone clean-db-volumes test test-hurl test-hurl-groups fmt lint helm-lint helm-template helm-deploy helm-status helm-uninstall test-schemathesis
