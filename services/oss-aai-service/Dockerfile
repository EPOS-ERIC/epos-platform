FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY services/oss-aai-service/go.mod services/oss-aai-service/go.sum ./
RUN go mod download

COPY services/oss-aai-service/. .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/oss-aai-service .

FROM alpine:3.22

WORKDIR /app

RUN adduser -D -g '' appuser

COPY --from=builder /out/oss-aai-service /app/oss-aai-service
COPY --from=builder /app/templates /app/templates
COPY packages/scripts/epos-banner.sh /usr/local/bin/epos-banner.sh

RUN mkdir -p /app/data && chown -R appuser:appuser /app

USER appuser

ENV EPOS_SERVICE_NAME="OSS AAI" \
    EPOS_PORT="8080"

EXPOSE 8080

ENTRYPOINT ["/bin/sh", "/usr/local/bin/epos-banner.sh", "/app/oss-aai-service"]
