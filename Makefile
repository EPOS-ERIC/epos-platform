up:
	docker compose up -f ./docker/docker-compose.yaml -d --build

down:
	docker compose down -f ./docker/docker-compose.yaml

populate:
	./packages/scripts/populatescript.sh

.PHONY: up down populate
