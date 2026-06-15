FROM amazoncorretto:21

WORKDIR /app

# Copy the fat jar built on host via `make build`
COPY services/metadata-lifecycle/build/libs/metadata-service-*.jar /app/app.jar
COPY packages/scripts/epos-banner.sh /usr/local/bin/epos-banner.sh

ENV EPOS_SERVICE_NAME="Metadata Lifecycle" \
    EPOS_PORT="8080"

# Expose API port
EXPOSE 8080

# Default: run the API server (or pass args like "populate")
ENTRYPOINT ["/bin/sh", "/usr/local/bin/epos-banner.sh", "java", "-jar", "/app/app.jar"]
