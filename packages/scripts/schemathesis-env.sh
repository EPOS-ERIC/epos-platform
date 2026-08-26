#!/bin/bash

set -euo pipefail

AAI_URL=${SCHEMATHESIS_AAI_URL:-http://localhost:8110}
GATEWAY_API_URL=${SCHEMATHESIS_GATEWAY_API_URL:-http://localhost:32000/v1}
AUTH_USERNAME=${SCHEMATHESIS_AUTH_USERNAME:-admin@admin.org}
AUTH_PASSWORD=${SCHEMATHESIS_AUTH_PASSWORD:-adminadmin}
AUTH_CLIENT_ID=${SCHEMATHESIS_AUTH_CLIENT_ID:-local-dev-client}
AUTH_CLIENT_SECRET=${SCHEMATHESIS_AUTH_CLIENT_SECRET:-dev-secret}
TARGET_GROUP_ID=${SCHEMATHESIS_TARGET_GROUP_ID:-}
TARGET_GROUP_NAME=${SCHEMATHESIS_TARGET_GROUP_NAME:-Seismology}

log() {
  printf '%s\n' "$*" >&2
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    log "Missing required command: $1"
    exit 1
  fi
}

fetch_json() {
  local method=$1
  local url=$2
  local body=${3:-}
  shift 3 || true

  local curl_args=()
  while [ "$#" -gt 0 ]; do
    curl_args+=("$1")
    shift
  done

  local response
  if [ -n "$body" ]; then
    response=$(curl --silent --show-error --fail-with-body -X "$method" "$url" "${curl_args[@]}" --data "$body")
  elif [ "${#curl_args[@]}" -gt 0 ]; then
    response=$(curl --silent --show-error --fail-with-body -X "$method" "$url" "${curl_args[@]}")
  else
    response=$(curl --silent --show-error --fail-with-body -X "$method" "$url")
  fi

  printf '%s' "$response"
}

require_command curl
require_command jq

log "Fetching Schemathesis auth token"
token_response=$(curl \
  --silent \
  --show-error \
  --fail-with-body \
  --request POST \
  "$AAI_URL/oauth/token" \
  --user "$AUTH_CLIENT_ID:$AUTH_CLIENT_SECRET" \
  --data-urlencode "grant_type=password" \
  --data-urlencode "username=$AUTH_USERNAME" \
  --data-urlencode "password=$AUTH_PASSWORD")

token=$(printf '%s' "$token_response" | jq -r '.access_token // empty')
if [ -z "$token" ]; then
  log "OSS AAI token response missing access_token"
  exit 1
fi

auth_header="Authorization: Bearer $token"

if [ -n "$TARGET_GROUP_ID" ]; then
  group_id=$TARGET_GROUP_ID
  target_group_label=$TARGET_GROUP_ID
else
  log "Resolving target group '$TARGET_GROUP_NAME'"
  groups_response=$(fetch_json GET "$GATEWAY_API_URL/groups" "" --header "$auth_header")
  group_id=$(printf '%s' "$groups_response" | jq -r --arg name "$TARGET_GROUP_NAME" 'map(select(.name == $name)) | .[0].id // empty')
  target_group_label=$TARGET_GROUP_NAME
fi
if [ -z "$group_id" ]; then
  log "Target group '$target_group_label' not found"
  exit 1
fi

log "Resolving target category"
categories_response=$(fetch_json GET "$GATEWAY_API_URL/groups/$group_id/categories" "" --header "$auth_header")
category_id=$(printf '%s' "$categories_response" | jq -r '.[0].id // empty')
if [ -z "$category_id" ]; then
  log "Target group '$target_group_label' has no categories"
  exit 1
fi

log "Resolving target draft resource"
resources_response=$(fetch_json GET "$GATEWAY_API_URL/search?type=dataset&versioningStatus=DRAFT" "" --header "$auth_header")
resource_id=$(printf '%s' "$resources_response" | jq -r '.[0].resource_id // empty')
resource_version_id=$(printf '%s' "$resources_response" | jq -r '.[0].version_id // empty')
provider_id=$(printf '%s' "$resources_response" | jq -r '.[0].providers[0].id // empty')
if [ -z "$resource_id" ] || [ -z "$resource_version_id" ]; then
  log "No draft dataset found; skipping version-read target"
  printf 'SCHEMATHESIS_EXCLUDE_RESOURCE_PATH=--exclude-path\ /resources/{resourceType}/{id}/versions/{versionId}\n'
else
  printf 'SCHEMATHESIS_RESOURCE_ID=%s\n' "$resource_id"
  printf 'SCHEMATHESIS_RESOURCE_VERSION_ID=%s\n' "$resource_version_id"
  if [ -n "$provider_id" ]; then
    printf 'SCHEMATHESIS_PROVIDER_ID=%s\n' "$provider_id"
  fi
  printf 'SCHEMATHESIS_EXCLUDE_RESOURCE_PATH=\n'
fi

printf 'SCHEMATHESIS_BEARER_TOKEN=%s\n' "$token"
printf 'SCHEMATHESIS_GROUP_ID=%s\n' "$group_id"
printf 'SCHEMATHESIS_CATEGORY_ID=%s\n' "$category_id"
