#!/bin/bash

set -euo pipefail

AAI_URL=${HURL_AAI_URL:-http://localhost:8110}
GATEWAY_API_URL=${HURL_GATEWAY_API_URL:-http://localhost:32000/v1}
AUTH_USERNAME=${HURL_AUTH_USERNAME:-admin@admin.org}
AUTH_PASSWORD=${HURL_AUTH_PASSWORD:-adminadmin}
AUTH_CLIENT_ID=${HURL_AUTH_CLIENT_ID:-local-dev-client}
AUTH_CLIENT_SECRET=${HURL_AUTH_CLIENT_SECRET:-dev-secret}
RUN_ID=${HURL_AUTHZ_RUN_ID:-$(date +%s)-$$}

USER_PASSWORD=${HURL_AUTHZ_USER_PASSWORD:-hurlpass123}
GROUP_ADMIN_PASSWORD=${HURL_AUTHZ_GROUP_ADMIN_PASSWORD:-hurlpass123}

USER_EMAIL="hurl-user-${RUN_ID}@example.org"
GROUP_ADMIN_EMAIL="hurl-group-admin-${RUN_ID}@example.org"
GROUP_A_SLUG="hurl-authz-a-${RUN_ID}"
GROUP_B_SLUG="hurl-authz-b-${RUN_ID}"
ADMIN_LIFECYCLE_GROUP_SLUG="hurl-admin-lifecycle-${RUN_ID}"

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

fetch_token() {
  local username=$1
  local password=$2

  local token_response
  token_response=$(curl \
    --silent \
    --show-error \
    --fail-with-body \
    --request POST \
    "$AAI_URL/oauth/token" \
    --user "$AUTH_CLIENT_ID:$AUTH_CLIENT_SECRET" \
    --data-urlencode "grant_type=password" \
    --data-urlencode "username=$username" \
    --data-urlencode "password=$password")

  local token
  token=$(printf '%s' "$token_response" | jq -r '.access_token // empty')
  if [ -z "$token" ]; then
    log "OSS AAI token response missing access_token for $username"
    exit 1
  fi

  printf '%s' "$token"
}

resolve_self_user_id() {
  local token=$1
  local self_response
  self_response=$(fetch_json GET "$GATEWAY_API_URL/users/self" "" --header "Authorization: Bearer $token")

  local user_id
  user_id=$(printf '%s' "$self_response" | jq -r '.id // empty')
  if [ -z "$user_id" ]; then
    log "Gateway self user response missing id"
    exit 1
  fi

  printf '%s' "$user_id"
}

register_user() {
  local name=$1
  local surname=$2
  local email=$3
  local password=$4

  local cookie_jar
  cookie_jar=$(mktemp)

  local register_page
  register_page=$(curl --silent --show-error --fail-with-body --cookie-jar "$cookie_jar" "$AAI_URL/register")

  local csrf_token
  csrf_token=$(printf '%s' "$register_page" | sed -n 's/.*name="csrf_token" value="\([^"]*\)".*/\1/p' | head -n 1)
  if [ -z "$csrf_token" ]; then
    rm -f "$cookie_jar"
    log "Register page missing csrf_token"
    exit 1
  fi

  curl \
    --silent \
    --show-error \
    --fail-with-body \
    --request POST \
    --cookie "$cookie_jar" \
    --header "Content-Type: application/x-www-form-urlencoded" \
    --data-urlencode "csrf_token=$csrf_token" \
    --data-urlencode "continue=" \
    --data-urlencode "name=$name" \
    --data-urlencode "surname=$surname" \
    --data-urlencode "email=$email" \
    --data-urlencode "password=$password" \
    "$AAI_URL/register" \
    >/dev/null

  rm -f "$cookie_jar"
}

create_group() {
  local token=$1
  local slug=$2
  local name=$3
  local description=$4

  local response
  response=$(fetch_json \
    POST \
    "$GATEWAY_API_URL/groups" \
    "{\"slug\":\"$slug\",\"name\":\"$name\",\"description\":\"$description\",\"is_system\":false}" \
    --header "Authorization: Bearer $token" \
    --header "Content-Type: application/json")

  local group_id
  group_id=$(printf '%s' "$response" | jq -r '.id // empty')
  if [ -z "$group_id" ]; then
    log "Create group response missing id for slug $slug"
    exit 1
  fi

  printf '%s' "$group_id"
}

request_membership() {
  local token=$1
  local group_id=$2
  local role=$3

  fetch_json \
    POST \
    "$GATEWAY_API_URL/groups/$group_id/memberships" \
    "{\"role\":\"$role\"}" \
    --header "Authorization: Bearer $token" \
    --header "Content-Type: application/json" \
    >/dev/null
}

patch_membership() {
  local token=$1
  local group_id=$2
  local user_id=$3
  local role=$4
  local status=$5

  fetch_json \
    PATCH \
    "$GATEWAY_API_URL/groups/$group_id/memberships/$user_id" \
    "{\"role\":\"$role\",\"status\":\"$status\"}" \
    --header "Authorization: Bearer $token" \
    --header "Content-Type: application/json" \
    >/dev/null
}

require_command curl
require_command jq
require_command sed

log "Fetching admin token"
admin_token=$(fetch_token "$AUTH_USERNAME" "$AUTH_PASSWORD")
admin_user_id=$(resolve_self_user_id "$admin_token")

log "Registering regular user"
register_user "Hurl" "User" "$USER_EMAIL" "$USER_PASSWORD"
user_token=$(fetch_token "$USER_EMAIL" "$USER_PASSWORD")
user_id=$(resolve_self_user_id "$user_token")

log "Registering group admin user"
register_user "Hurl" "GroupAdmin" "$GROUP_ADMIN_EMAIL" "$GROUP_ADMIN_PASSWORD"
group_admin_token=$(fetch_token "$GROUP_ADMIN_EMAIL" "$GROUP_ADMIN_PASSWORD")
group_admin_user_id=$(resolve_self_user_id "$group_admin_token")

log "Creating authz groups"
group_a_id=$(create_group "$admin_token" "$GROUP_A_SLUG" "Hurl Authz Group A" "Group admin allowed scope")
group_b_id=$(create_group "$admin_token" "$GROUP_B_SLUG" "Hurl Authz Group B" "Group admin forbidden scope")

log "Promoting group admin on group A"
request_membership "$group_admin_token" "$group_a_id" "VIEWER"
patch_membership "$admin_token" "$group_a_id" "$group_admin_user_id" "ADMIN" "ACTIVE"

printf 'HURL_ADMIN_BEARER_TOKEN=%s\n' "$admin_token"
printf 'HURL_ADMIN_USER_ID=%s\n' "$admin_user_id"
printf 'HURL_USER_BEARER_TOKEN=%s\n' "$user_token"
printf 'HURL_USER_ID=%s\n' "$user_id"
printf 'HURL_GROUP_ADMIN_BEARER_TOKEN=%s\n' "$group_admin_token"
printf 'HURL_GROUP_ADMIN_USER_ID=%s\n' "$group_admin_user_id"
printf 'HURL_GROUP_A_ID=%s\n' "$group_a_id"
printf 'HURL_GROUP_A_SLUG=%s\n' "$GROUP_A_SLUG"
printf 'HURL_GROUP_B_ID=%s\n' "$group_b_id"
printf 'HURL_GROUP_B_SLUG=%s\n' "$GROUP_B_SLUG"
printf 'HURL_ADMIN_LIFECYCLE_GROUP_SLUG=%s\n' "$ADMIN_LIFECYCLE_GROUP_SLUG"
