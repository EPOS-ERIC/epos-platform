.PHONY: install dev dev-oss dev-tilt build build-oss lint test cypress

install:
	npm ci

dev:
	npm run dev

dev-oss:
	npm run dev-oss

dev-tilt:
	npm install --no-audit --no-fund
	npm run dev-oss -- --host 0.0.0.0 --proxy-config proxy.tilt.conf.json

build:
	npm run build

build-oss:
	npm run build-oss

lint:
	npm run lint

test:
	npm test

cypress:
	npm run cypress
