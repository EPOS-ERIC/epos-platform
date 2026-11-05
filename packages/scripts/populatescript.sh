#!/bin/bash

set -euo pipefail

MERGED_TTL_FILE=$(mktemp)
IMPORT_RESPONSE_FILE=$(mktemp)

cleanup() {
  docker rm -f cache >/dev/null 2>&1 || true
  rm -f "$MERGED_TTL_FILE" "$IMPORT_RESPONSE_FILE"
}

trap cleanup EXIT

HOST_ALIAS=${HOST_ALIAS:-localhost}
GATEWAY_API_URL=${GATEWAY_API_URL:-http://localhost:33000/v1}
AAI_URL=${AAI_URL:-http://localhost:35000}
AUTH_CLIENT_ID=${AUTH_CLIENT_ID:-local-dev-client}
AUTH_CLIENT_SECRET=${AUTH_CLIENT_SECRET:-dev-secret}
AUTH_USERNAME=${AUTH_USERNAME:-admin@admin.org}
AUTH_PASSWORD=${AUTH_PASSWORD:-adminadmin}
IMPORT_BASE_URI=${IMPORT_BASE_URI:-https://www.epos-eu.org/epos-dcat-ap/}
IMPORT_MAPPING_ID=${IMPORT_MAPPING_ID:-epos-dcat-ap}
IMPORT_MAPPING_VERSION=${IMPORT_MAPPING_VERSION:-1}

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

SEEN_SUBJECTS_FILE=$(mktemp)
TAIL_INDEX=0

# Disambiguate subject IRIs redefined multiple times (across files or within
# the same file): the first definition keeps its URI; every later definition
# gets a file-scoped suffix so the merged graph keeps every definition intact.
disambiguate_file() {
  local file=$1 index=$2
  local subjects
  subjects=$(grep -o '^<[^>]*>' "$file" | sort -u || true)
  # Phase 1: subjects defined in earlier files get a file-scoped suffix.
  local subject iri
  while IFS= read -r subject; do
    [ -z "$subject" ] && continue
    if grep -qFx "$subject" "$SEEN_SUBJECTS_FILE"; then
      iri=${subject#<}
      iri=${iri%>}
      perl -i -pe 's{\Q<'"$iri"'>\E}{<'"$iri"_"$index"'>}g' "$file"
    else
      printf '%s\n' "$subject" >> "$SEEN_SUBJECTS_FILE"
    fi
  done <<< "$subjects"
  # Phase 2: subjects defined more than once within this file: keep the first
  # occurrence, suffix the following ones.
  while IFS= read -r subject; do
    [ -z "$subject" ] && continue
    iri=${subject#<}
    iri=${iri%>}
    awk -v old="<${iri}>" -v suffix="_${index}" \
      'BEGIN{count=0} { if (substr($0, 1, length(old)) == old) { count++; if (count>1) { $0 = substr(old, 1, length(old)-1) suffix "_" (count-1) ">" substr($0, length(old)+1) } } print }' \
      "$file" > "$file.tmp" && mv "$file.tmp" "$file"
  done < <(grep -o '^<[^>]*>' "$file" | sort | uniq -d)
}

INDEX_CONTENT=$(curl --silent --show-error --fail-with-body "$INDEX_URL")
while IFS= read -r TTL_URL; do
  case "$TTL_URL" in
    ""|'#'*)
      continue
      ;;
  esac

  printf 'Fetching %s\n' "$TTL_URL"
  local_tar=$(mktemp)
  # Repair legacy forms rejected by the gateway Turtle parser.
  curl --silent --show-error --fail-with-body "$TTL_URL" | \
    sed -E \
      -e 's/^(@prefix[[:space:]]+[^:[:space:]]+:)</\1 </' \
      -e 's|<environment_type:|<environment-type:|g' > "$local_tar"
  TAIL_INDEX=$((TAIL_INDEX + 1))
  disambiguate_file "$local_tar" "$TAIL_INDEX"
  cat "$local_tar" >> "$MERGED_TTL_FILE"
  rm -f "$local_tar"
  printf '\n' >> "$MERGED_TTL_FILE"
done <<< "$INDEX_CONTENT"
rm -f "$SEEN_SUBJECTS_FILE"

printf 'Importing merged metadata through %s\n' "$IMPORT_URL"
jq -n \
  --rawfile turtle "$MERGED_TTL_FILE" \
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
  --data-binary @- > "$IMPORT_RESPONSE_FILE"
jq . "$IMPORT_RESPONSE_FILE"
if jq -e '.failure? != null' "$IMPORT_RESPONSE_FILE" >/dev/null; then
  printf 'Metadata population failed\n' >&2
  exit 1
fi

warning_count=$(jq '.warnings | length' "$IMPORT_RESPONSE_FILE")
if [ "$warning_count" -gt 0 ]; then
  printf 'Metadata population completed with %s warning(s):\n' "$warning_count" >&2
  jq -r '.warnings[] | "WARNING: \(.subject): [\(.code)] \(.message // "Import skipped")"' "$IMPORT_RESPONSE_FILE" >&2
fi

printf 'Metadata population completed\n'
