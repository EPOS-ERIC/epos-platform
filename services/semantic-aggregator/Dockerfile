FROM eclipse-temurin:25-jre-alpine
RUN adduser -D -u 1000 appuser && mkdir -p /state && chown 1000:1000 /state
COPY services/semantic-aggregator/target/semantic-aggregator.jar /semantic-aggregator.jar
COPY packages/scripts/epos-banner.sh /usr/local/bin/epos-banner.sh
ENV EPOS_SERVICE_NAME="Semantic Aggregator" EPOS_PORT="8300"
USER 1000
EXPOSE 8300
ENTRYPOINT ["/bin/sh", "/usr/local/bin/epos-banner.sh", "java", "-jar", "/semantic-aggregator.jar"]
