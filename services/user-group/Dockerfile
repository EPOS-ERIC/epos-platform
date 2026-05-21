FROM alpine:3.22.1

WORKDIR /app

COPY services/user-group/user-group /usr/local/bin/user-group

EXPOSE 8090

ENTRYPOINT ["/usr/local/bin/user-group"]
