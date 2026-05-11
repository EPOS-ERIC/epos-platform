IMAGE ?= epos-metadata-lifecycle:latest

build:
	./gradlew --no-daemon clean shadowJar

build-docker: build
	docker build -t $(IMAGE) .

test:
	./gradlew --no-daemon test

fmt:
	./gradlew --no-daemon spotlessApply

lint: fmt
	./gradlew --no-daemon spotlessCheck

.PHONY: build build-docker test fmt lint
