IMAGE ?= epos-platform:latest

build:
	pnpm run build

docker:
	docker build -t $(IMAGE) -f Dockerfile .

test:
	pnpm run test

fmt:
	pnpm run fmt

lint: fmt
	pnpm run lint

codegen:
	pnpm run codegen

.PHONY: build docker test fmt lint codegen
