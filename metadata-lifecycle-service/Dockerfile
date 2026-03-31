# Multi-stage build: Build stage
FROM maven:3.9-eclipse-temurin-17 AS build

WORKDIR /app

# Copy pom.xml and download dependencies (layer caching)
COPY pom.xml .
RUN mvn dependency:go-offline -B

# Copy source code
COPY src ./src

# Build the application and copy runtime dependencies
RUN mvn clean package -DskipTests && mvn dependency:copy-dependencies -DincludeScope=runtime -DskipTests

# Runtime stage
FROM eclipse-temurin:17-jre-alpine

WORKDIR /app

# Copy the built JAR and runtime dependencies from build stage
COPY --from=build /app/target/metadata-service-*.jar /app/app.jar
COPY --from=build /app/target/dependency /app/lib

# Expose API port
EXPOSE 8080

# Default: run the API server (or pass args like "populate")
ENTRYPOINT ["java", "-cp", "/app/app.jar:/app/lib/*", "org.epos.ApplicationKt"]
