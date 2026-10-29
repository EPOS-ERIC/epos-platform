FROM alpine:3.22

WORKDIR /app

RUN adduser -D -g '' appuser

COPY services/oss-aai-service/bin/oss-aai-service /app/oss-aai-service
COPY services/oss-aai-service/templates /app/templates
COPY packages/scripts/epos-banner.sh /usr/local/bin/epos-banner.sh

RUN mkdir -p /app/data && chown -R appuser:appuser /app

USER appuser

ENV EPOS_SERVICE_NAME="OSS AAI" \
    EPOS_PORT="8080"

EXPOSE 8080

ENTRYPOINT ["/bin/sh", "/usr/local/bin/epos-banner.sh", "/app/oss-aai-service"]
