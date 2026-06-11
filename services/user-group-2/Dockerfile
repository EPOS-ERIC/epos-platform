FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

RUN apk upgrade --no-cache

WORKDIR /app

COPY services/gateway/gateway /usr/local/bin/gateway
COPY docker/gateway-config.yaml /etc/gateway/config.yaml

EXPOSE 5000

ENTRYPOINT ["/usr/local/bin/gateway"]
