FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

RUN apk upgrade --no-cache && apk add --no-cache ca-certificates git openjdk21-jre python3 py3-pip
RUN adduser -D -u 1000 appuser && mkdir -p /data/plugins && chown 1000:1000 /data/plugins

WORKDIR /
COPY services/converter/converter /converter
COPY packages/scripts/epos-banner.sh /usr/local/bin/epos-banner.sh

ENV EPOS_SERVICE_NAME="Converter" \
    EPOS_PORT="8200" \
    PLUGINS_PATH="/data/plugins"

USER 1000
EXPOSE 8200
ENTRYPOINT ["/bin/sh", "/usr/local/bin/epos-banner.sh", "/converter"]
