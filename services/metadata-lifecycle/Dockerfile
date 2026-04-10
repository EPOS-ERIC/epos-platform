# Multi-stage build: Build stage
FROM amazoncorretto:17 AS build

WORKDIR /app

# Copy Gradle wrapper and build scripts first for layer caching
COPY gradle ./gradle
COPY gradlew gradlew
COPY settings.gradle.kts build.gradle.kts gradle.properties ./

RUN chmod +x gradlew

# Copy source code
COPY src ./src

# Build the fat jar
RUN ./gradlew --no-daemon shadowJar

# Runtime stage
FROM amazoncorretto:17

WORKDIR /app

# Copy the fat jar from build stage
COPY --from=build /app/build/libs/metadata-service-*.jar /app/app.jar

# Expose API port
EXPOSE 8080

# Default: run the API server (or pass args like "populate")
ENTRYPOINT ["java", "-jar", "/app/app.jar"]
