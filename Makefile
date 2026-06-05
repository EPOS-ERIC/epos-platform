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
	docker volume rm -f docker_neo4j-data docker_user-group-db-data docker_data-retriever-db-data docker_oss-aai-data

test:
	$(MAKE) -C apps/platform test
	$(MAKE) -C services/gateway test
	$(MAKE) -C services/user-group test
	$(MAKE) -C services/data-retriever test
	$(MAKE) -C services/metadata-lifecycle test
	$(MAKE) -C services/oss-aai-service test

fmt:
	$(MAKE) -C apps/platform fmt
	$(MAKE) -C services/gateway fmt
	$(MAKE) -C services/user-group fmt
	$(MAKE) -C services/data-retriever fmt
	$(MAKE) -C services/metadata-lifecycle fmt
	$(MAKE) -C services/oss-aai-service fmt

lint:
	$(MAKE) -C apps/platform lint
	$(MAKE) -C services/gateway lint
	$(MAKE) -C services/user-group lint
	$(MAKE) -C services/data-retriever lint
	$(MAKE) -C services/metadata-lifecycle lint
	$(MAKE) -C services/oss-aai-service lint

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

test-integration: test-schemathesis test-hurl

test-schemathesis:
	SCHEMATHESIS_HOOKS=tests/schemathesis/hooks.py schemathesis run \
		--url "http://localhost:32000/v1" \
		--suppress-health-check=filter_too_much \
		--exclude-checks unsupported_method \
		services/gateway/docs/openapi.yaml

test-hurl:
	hurl --variable baseUrl=http://localhost:32000 --test tests/hurl

.PHONY: up down populate populate-metadatone clean-db-volumes test test-hurl fmt lint helm-lint helm-template helm-deploy helm-status helm-uninstall test-schemathesis
