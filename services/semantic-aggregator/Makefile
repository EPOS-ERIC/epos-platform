IMAGE ?= epos-semantic-aggregator:latest
build: generate
	mvn -B package

dev:
	mvn -B package -DskipTests
	java -jar target/semantic-aggregator.jar

docker: build
	docker build -t $(IMAGE) -f Dockerfile ../..

test: generate
	mvn -B test

lint: generate
	mvn -B clean compile -Dmaven.compiler.showWarnings=true

generate:
	docker run --rm --user "$$(id -u):$$(id -g)" -e HOME=/tmp \
		-v "$(abspath ../../packages/protobuf):/workspace" -w /workspace \
		--entrypoint sh bufbuild/buf:1.75.0 -c 'buf generate'

clean:
	mvn -B clean

.PHONY: build dev docker test lint clean generate
