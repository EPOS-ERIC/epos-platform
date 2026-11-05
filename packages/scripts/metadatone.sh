#!/bin/bash

set -euo pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
TTL_FILE=${TTL_FILE:-"$SCRIPT_DIR/../assets/metadatone.ttl"}
GATEWAY_API_URL=${GATEWAY_API_URL:-http://localhost:33000/v1}
AAI_URL=${AAI_URL:-http://localhost:35000}
AUTH_CLIENT_ID=${AUTH_CLIENT_ID:-local-dev-client}
AUTH_CLIENT_SECRET=${AUTH_CLIENT_SECRET:-dev-secret}
AUTH_USERNAME=${AUTH_USERNAME:-admin@admin.org}
AUTH_PASSWORD=${AUTH_PASSWORD:-adminadmin}
IMPORT_BASE_URI=${IMPORT_BASE_URI:-https://www.epos-eu.org/epos-dcat-ap/}
IMPORT_MAPPING_ID=${IMPORT_MAPPING_ID:-epos-dcat-ap}
IMPORT_MAPPING_VERSION=${IMPORT_MAPPING_VERSION:-1}

if [ ! -f "$TTL_FILE" ]; then
  printf 'Turtle file not found: %s\n' "$TTL_FILE" >&2
  exit 1
fi

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

IMPORT_URL="$GATEWAY_API_URL/imports/turtle"
printf 'Importing %s through %s\n' "$TTL_FILE" "$IMPORT_URL"

jq -n \
  --rawfile turtle "$TTL_FILE" \
  --arg base_uri "$IMPORT_BASE_URI" \
  --arg mapping_id "$IMPORT_MAPPING_ID" \
  --argjson mapping_version "$IMPORT_MAPPING_VERSION" \
  '{turtle: $turtle, baseUri: $base_uri, mappingId: $mapping_id, mappingVersion: $mapping_version}' | \
curl \
  --silent \
  --show-error \
  --fail-with-body \
  --request POST \
  "$IMPORT_URL" \
  --header "Authorization: Bearer $TOKEN" \
  --header 'Content-Type: application/json' \
  --data-binary @-
printf '\n'
