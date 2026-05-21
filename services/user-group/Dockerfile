FROM alpine:3.22.1

WORKDIR /app

COPY services/user-group/user-group /usr/local/bin/user-group
COPY services/user-group/openapi.yaml /app/openapi.yaml

EXPOSE 8090

ENTRYPOINT ["/usr/local/bin/user-group"]
