FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

RUN apk upgrade --no-cache

WORKDIR /
COPY services/data-retriever/data-retriever /data-retriever
COPY services/data-retriever/migrations /migrations
COPY packages/scripts/epos-banner.sh /usr/local/bin/epos-banner.sh

ENV EPOS_SERVICE_NAME="Data Retriever" \
    EPOS_PORT="8100"

EXPOSE 8100
ENTRYPOINT ["/bin/sh", "/usr/local/bin/epos-banner.sh", "/data-retriever"]
