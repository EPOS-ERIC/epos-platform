.PHONY: install dev dev-tilt build test generate

PNPM ?= pnpm

install:
	$(PNPM) install --frozen-lockfile

dev:
	$(PNPM) dev

dev-tilt: install
	$(PNPM) start --host 0.0.0.0 --proxy-config proxy.tilt.conf.json

build:
	$(PNPM) build

test:
	$(PNPM) exec ng test --watch=false

generate:
	$(PNPM) codegen
