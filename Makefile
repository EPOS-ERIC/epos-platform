up:
	docker compose up -d --build

down:
	docker compose down

populate:
	./populatescript.sh

.PHONY: up down
