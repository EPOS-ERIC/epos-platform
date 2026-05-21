FROM amazoncorretto:21

WORKDIR /app

# Copy the fat jar built on host via `make build`
COPY build/libs/metadata-service-*.jar /app/app.jar

# Expose API port
EXPOSE 8080

# Default: run the API server (or pass args like "populate")
ENTRYPOINT ["java", "-jar", "/app/app.jar"]
