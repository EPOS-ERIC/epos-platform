up:
	docker compose -f ./docker/docker-compose.yaml up -d --build

down:
	docker compose -f ./docker/docker-compose.yaml down

populate:
	./packages/scripts/populatescript.sh

build:
	cd services/gateway && $(MAKE) build
	cd services/metadata-lifecycle && $(MAKE) build

test:
	@echo "==> services/gateway: skipping test (target missing)"
	cd services/metadata-lifecycle && $(MAKE) test

fmt:
	@echo "==> services/gateway: skipping fmt (target missing)"
	cd services/metadata-lifecycle && $(MAKE) fmt

lint:
	@echo "==> services/gateway: skipping lint (target missing)"
	cd services/metadata-lifecycle && $(MAKE) lint

.PHONY: up down populate build test fmt lint
