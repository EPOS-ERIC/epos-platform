FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

RUN apk upgrade --no-cache

WORKDIR /app

COPY services/user-group/user-group /usr/local/bin/user-group
COPY packages/scripts/epos-banner.sh /usr/local/bin/epos-banner.sh

ENV EPOS_SERVICE_NAME="User Group" \
    EPOS_PORT="8090"

EXPOSE 8090

ENTRYPOINT ["/bin/sh", "/usr/local/bin/epos-banner.sh", "/usr/local/bin/user-group"]
