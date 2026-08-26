FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

RUN apk upgrade --no-cache

WORKDIR /app

COPY services/metadata-lifecycle/metadata-lifecycle /usr/local/bin/metadata-lifecycle
COPY packages/scripts/epos-banner.sh /usr/local/bin/epos-banner.sh

ENV EPOS_SERVICE_NAME="Metadata Lifecycle" \
    EPOS_PORT="5000"

EXPOSE 5000

ENTRYPOINT ["/bin/sh", "/usr/local/bin/epos-banner.sh", "/usr/local/bin/metadata-lifecycle"]
