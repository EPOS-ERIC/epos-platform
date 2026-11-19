FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

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
