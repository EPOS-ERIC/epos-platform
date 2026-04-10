build:
	./gradlew --no-daemon clean shadowJar

test:
	./gradlew --no-daemon test

fmt:
	./gradlew --no-daemon spotlessApply

lint: fmt
	./gradlew --no-daemon spotlessCheck

.PHONY: build test fmt lint
