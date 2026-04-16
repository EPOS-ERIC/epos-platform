FROM alpine:3.22.1

WORKDIR /app

COPY services/gateway/gateway /usr/local/bin/gateway
COPY docker/gateway-config.yaml /etc/gateway/config.yaml

EXPOSE 5000

ENTRYPOINT ["/usr/local/bin/gateway"]
