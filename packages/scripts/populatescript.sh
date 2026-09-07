#!/bin/bash

set -euo pipefail

MERGED_TTL_FILE=$(mktemp)

cleanup() {
  docker rm -f cache >/dev/null 2>&1 || true
  rm -f "$MERGED_TTL_FILE"
}

trap cleanup EXIT

HOST_ALIAS=${HOST_ALIAS:-localhost}
GATEWAY_API_URL=${GATEWAY_API_URL:-http://localhost:32000/v1}
AAI_URL=${AAI_URL:-http://localhost:8110}
AUTH_CLIENT_ID=${AUTH_CLIENT_ID:-local-dev-client}
AUTH_CLIENT_SECRET=${AUTH_CLIENT_SECRET:-dev-secret}
AUTH_USERNAME=${AUTH_USERNAME:-admin@admin.org}
AUTH_PASSWORD=${AUTH_PASSWORD:-adminadmin}

if [ -z "$HOST_ALIAS" ]; then
  printf 'Unable to resolve host alias\n' >&2
  exit 1
fi

HOST_ENDPOINT="${HOST_ALIAS}:5050"
INDEX_URL="http://${HOST_ENDPOINT}/index.txt"
IMPORT_URL="$GATEWAY_API_URL/imports/turtle"

if [ -n "${EPOS_BEARER_TOKEN:-}" ]; then
  TOKEN=$EPOS_BEARER_TOKEN
else
  printf 'Fetching gateway import token\n'
  TOKEN_RESPONSE=$(curl \
    --silent \
    --show-error \
    --fail-with-body \
    --request POST \
    "$AAI_URL/oauth/token" \
    --user "$AUTH_CLIENT_ID:$AUTH_CLIENT_SECRET" \
    --data-urlencode 'grant_type=password' \
    --data-urlencode "username=$AUTH_USERNAME" \
    --data-urlencode "password=$AUTH_PASSWORD")

  TOKEN=$(printf '%s' "$TOKEN_RESPONSE" | jq -r '.access_token // empty')
  if [ -z "$TOKEN" ]; then
    printf 'Token response did not contain access_token\n' >&2
    exit 1
  fi
fi

printf 'Starting metadata-cache; host endpoint: %s\n' "$HOST_ENDPOINT"
docker rm -f cache >/dev/null 2>&1 || true
docker run -idt -p 5050:80 --name cache epos-ci.brgm.fr:5005/epos/metadata-cache:development >/dev/null

printf 'Waiting for metadata-cache\n'
sleep 2

printf 'Rewriting metadata-cache index URLs to http://%s/\n' "$HOST_ENDPOINT"
docker exec cache sed -E -i "s|https?://[^/]+/|http://$HOST_ENDPOINT/|g" /opt/metadata-cache/index.txt

INDEX_CONTENT=$(curl --silent --show-error --fail-with-body "$INDEX_URL")
while IFS= read -r TTL_URL; do
  case "$TTL_URL" in
    ""|'#'*)
      continue
      ;;
  esac

  printf 'Fetching %s\n' "$TTL_URL"
  # Repair legacy forms rejected by the gateway Turtle parser.
  curl --silent --show-error --fail-with-body "$TTL_URL" | \
    sed -E \
      -e 's/^(@prefix[[:space:]]+[^:[:space:]]+:)</\1 </' \
      -e 's|<environment_type:|<environment-type:|g' >> "$MERGED_TTL_FILE"
  printf '\n' >> "$MERGED_TTL_FILE"
done <<< "$INDEX_CONTENT"

printf 'Importing merged metadata through %s\n' "$IMPORT_URL"
curl \
  --silent \
  --show-error \
  --fail-with-body \
  --request POST \
  "$IMPORT_URL" \
  --header "Authorization: Bearer $TOKEN" \
  --header 'Content-Type: text/turtle' \
  --data-binary "@$MERGED_TTL_FILE"
printf '\n'

printf 'Metadata population completed\n'
