#!/bin/bash

set -euo pipefail

AAI_URL=${SCHEMATHESIS_AAI_URL:-http://localhost:8110}
GATEWAY_API_URL=${SCHEMATHESIS_GATEWAY_API_URL:-http://localhost:32000/v1}
AUTH_USERNAME=${SCHEMATHESIS_AUTH_USERNAME:-admin@admin.org}
AUTH_PASSWORD=${SCHEMATHESIS_AUTH_PASSWORD:-adminadmin}
AUTH_CLIENT_ID=${SCHEMATHESIS_AUTH_CLIENT_ID:-local-dev-client}
AUTH_CLIENT_SECRET=${SCHEMATHESIS_AUTH_CLIENT_SECRET:-dev-secret}
TARGET_GROUP_SLUG=${SCHEMATHESIS_TARGET_GROUP_SLUG:-all}
TARGET_MEMBERSHIP_ROLE=${SCHEMATHESIS_TARGET_MEMBERSHIP_ROLE:-VIEWER}

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
  else
    response=$(curl --silent --show-error --fail-with-body -X "$method" "$url" "${curl_args[@]}")
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

log "Resolving authenticated user"
self_response=$(fetch_json GET "$GATEWAY_API_URL/users/self" "" --header "$auth_header")
self_user_id=$(printf '%s' "$self_response" | jq -r '.id // empty')
if [ -z "$self_user_id" ]; then
  log "Gateway self user response missing id"
  exit 1
fi

log "Resolving target group '$TARGET_GROUP_SLUG'"
groups_response=$(fetch_json GET "$GATEWAY_API_URL/groups" "" --header "$auth_header")
group_id=$(printf '%s' "$groups_response" | jq -r --arg slug "$TARGET_GROUP_SLUG" 'map(select(.slug == $slug)) | .[0].id // empty')
if [ -z "$group_id" ]; then
  log "Target group '$TARGET_GROUP_SLUG' not found"
  exit 1
fi

log "Ensuring membership for authenticated user"
fetch_json \
  POST \
  "$GATEWAY_API_URL/groups/$group_id/memberships" \
  "{\"role\":\"$TARGET_MEMBERSHIP_ROLE\"}" \
  --header "$auth_header" \
  --header "Content-Type: application/json" \
  >/dev/null

printf 'SCHEMATHESIS_BEARER_TOKEN=%s\n' "$token"
printf 'SCHEMATHESIS_GROUP_ID=%s\n' "$group_id"
printf 'SCHEMATHESIS_MEMBERSHIP_USER_ID=%s\n' "$self_user_id"
