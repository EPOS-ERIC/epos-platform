#!/bin/bash

set -euo pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
TTL_FILE="$SCRIPT_DIR/../assets/metadatone.ttl"
SOURCE_NAME="metadatone.ttl"
API_URL="http://localhost:8080/api/v2/admin/populate/content"

escape_json_string() {
  awk '
    BEGIN {
      first = 1
    }
    {
      gsub(/\\/, "\\\\")
      gsub(/"/, "\\\"")
      gsub(/\r/, "\\r")
      gsub(/\t/, "\\t")

      if (!first) {
        printf "\\n"
      }

      printf "%s", $0
      first = 0
    }
  ' "$1"
}

TTL_CONTENT=$(escape_json_string "$TTL_FILE")
JSON_PAYLOAD=$(printf '{\n  "content": "%s",\n  "sourceName": "%s"\n}\n' \
  "$TTL_CONTENT" \
  "$SOURCE_NAME")

echo "Posting $TTL_FILE to $API_URL"

curl --fail-with-body -X POST \
  "$API_URL" \
  -H 'accept: */*' \
  -H 'X-Agent: http://test/agent' \
  -H 'X-Group: http://test/group' \
  -H 'Content-Type: application/json' \
  --data-binary "$JSON_PAYLOAD"
