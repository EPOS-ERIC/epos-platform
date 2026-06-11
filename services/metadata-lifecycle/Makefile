IMAGE ?= epos-metadata-lifecycle:latest

build:
	./gradlew --no-daemon clean shadowJar

build-docker: build
	docker build -t $(IMAGE) -f Dockerfile ../..

test:
	./gradlew --no-daemon test

test-integration:
	./gradlew --no-daemon test --tests "*IntegrationTest"

fmt:
	./gradlew --no-daemon spotlessApply

lint: fmt
	./gradlew --no-daemon spotlessCheck

.PHONY: build build-docker test test-integration fmt lint
