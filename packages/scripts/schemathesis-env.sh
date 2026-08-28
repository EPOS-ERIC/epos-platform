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

fetch_optional_json() {
  local method=$1
  local url=$2
  shift 2

  curl --silent --show-error -X "$method" "$url" "$@" || true
}

json_value() {
  local json=$1
  shift

  printf '%s' "$json" | jq -r "$@" 2>/dev/null || true
}

emit_env() {
  local name=$1
  local value=$2

  if [ -n "$value" ]; then
    printf '%s=%s\n' "$name" "$value"
  fi
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

log "Resolving canonical metadata targets"
contact_points_response=$(fetch_optional_json GET "$GATEWAY_API_URL/contact-points" --header "$auth_header")
contact_point_id=$(json_value "$contact_points_response" 'if type == "array" then .[0].id // empty else empty end')
organizations_response=$(fetch_optional_json GET "$GATEWAY_API_URL/organizations" --header "$auth_header")
organization_id=$(json_value "$organizations_response" 'if type == "array" then .[0].id // empty else empty end')
users_response=$(fetch_optional_json GET "$GATEWAY_API_URL/users" --header "$auth_header")
user_id=$(json_value "$users_response" --arg email "$AUTH_USERNAME" 'if type == "array" then map(select(.email != $email)) | .[0].id // empty else empty end')

log "Resolving membership targets"
memberships_response=$(fetch_optional_json GET "$GATEWAY_API_URL/groups/$group_id/memberships" --header "$auth_header")
membership_user_id=$(json_value "$memberships_response" 'if type == "array" then .[0].user.id // empty else empty end')
role_request_id=$(json_value "$memberships_response" 'if type == "array" then .[0].role_requests[0].id // empty else empty end')

log "Resolving lifecycle resource targets"
draft_resources_response=$(fetch_json GET "$GATEWAY_API_URL/search?type=dataset&versioningStatus=DRAFT" "" --header "$auth_header")
draft_resource_id=$(json_value "$draft_resources_response" '.[0].resource_id // empty')
draft_resource_version_id=$(json_value "$draft_resources_response" '.[0].version_id // empty')
provider_id=$(json_value "$draft_resources_response" '.[0].providers[0].id // empty')
published_resources_response=$(fetch_json GET "$GATEWAY_API_URL/search?type=dataset&versioningStatus=PUBLISHED" "" --header "$auth_header")
published_resource_id=$(json_value "$published_resources_response" '.[0].resource_id // empty')
published_resource_version_id=$(json_value "$published_resources_response" '.[0].version_id // empty')
archived_resources_response=$(fetch_json GET "$GATEWAY_API_URL/search?type=dataset&versioningStatus=ARCHIVED" "" --header "$auth_header")
archived_resource_id=$(json_value "$archived_resources_response" '.[0].resource_id // empty')
archived_resource_version_id=$(json_value "$archived_resources_response" '.[0].version_id // empty')

if [ -z "$draft_resource_id" ] || [ -z "$draft_resource_version_id" ]; then
  log "No draft dataset found; skipping version-read target"
  printf 'SCHEMATHESIS_EXCLUDE_RESOURCE_PATH=--exclude-path\ /resources/{resourceType}/{id}/versions/{versionId}\n'
else
  printf 'SCHEMATHESIS_EXCLUDE_RESOURCE_PATH=\n'
fi

printf 'SCHEMATHESIS_BEARER_TOKEN=%s\n' "$token"
printf 'SCHEMATHESIS_GROUP_ID=%s\n' "$group_id"
printf 'SCHEMATHESIS_CATEGORY_ID=%s\n' "$category_id"
emit_env SCHEMATHESIS_CONTACT_POINT_ID "$contact_point_id"
emit_env SCHEMATHESIS_ORGANIZATION_ID "$organization_id"
emit_env SCHEMATHESIS_USER_ID "$user_id"
emit_env SCHEMATHESIS_MEMBERSHIP_USER_ID "$membership_user_id"
emit_env SCHEMATHESIS_ROLE_REQUEST_ID "$role_request_id"
emit_env SCHEMATHESIS_RESOURCE_ID "$draft_resource_id"
emit_env SCHEMATHESIS_RESOURCE_VERSION_ID "$draft_resource_version_id"
emit_env SCHEMATHESIS_DRAFT_RESOURCE_ID "$draft_resource_id"
emit_env SCHEMATHESIS_DRAFT_RESOURCE_VERSION_ID "$draft_resource_version_id"
emit_env SCHEMATHESIS_PUBLISHED_RESOURCE_ID "$published_resource_id"
emit_env SCHEMATHESIS_PUBLISHED_RESOURCE_VERSION_ID "$published_resource_version_id"
emit_env SCHEMATHESIS_ARCHIVED_RESOURCE_ID "$archived_resource_id"
emit_env SCHEMATHESIS_ARCHIVED_RESOURCE_VERSION_ID "$archived_resource_version_id"
emit_env SCHEMATHESIS_PROVIDER_ID "$provider_id"
