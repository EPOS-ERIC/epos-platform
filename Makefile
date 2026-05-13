up: 
	docker compose -f ./docker/docker-compose.yaml up -d

down:
	docker compose -f ./docker/docker-compose.yaml down

populate:
	./packages/scripts/populatescript.sh

build:
	cd services/gateway && $(MAKE) build
	cd services/metadata-lifecycle && $(MAKE) build

build-docker:
	cd services/gateway && $(MAKE) build-docker
	cd services/metadata-lifecycle && $(MAKE) build-docker

generate:
	cd services/gateway && $(MAKE) generate

test:
	cd services/gateway && $(MAKE) test
	cd services/metadata-lifecycle && $(MAKE) test

fmt:
	cd services/gateway && $(MAKE) fmt
	cd services/metadata-lifecycle && $(MAKE) fmt

lint:
	cd services/gateway && $(MAKE) lint
	cd services/metadata-lifecycle && $(MAKE) lint

.PHONY: up down populate build build-docker generate test fmt lint
