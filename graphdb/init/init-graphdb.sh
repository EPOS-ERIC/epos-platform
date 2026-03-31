#!/bin/sh
set -e

GRAPHDB_URL="${GRAPHDB_URL:-http://graphdb:7200}"
REPO_ID="${REPO_ID:-epos}"
CONFIG_FILE="${CONFIG_FILE:-/config/repo-config.ttl}"
SHACL_URL="${SHACL_URL:-https://raw.githubusercontent.com/epos-eu/EPOS-DCAT-AP/refs/heads/EPOS-DCAT-AP-shapes/epos-dcat-ap_shapes.ttl}"
SHACL_GRAPH="http://rdf4j.org/schema/rdf4j#SHACLShapeGraph"
SHACL_GRAPH_ENCODED="%3Chttp%3A%2F%2Frdf4j.org%2Fschema%2Frdf4j%23SHACLShapeGraph%3E"
MAX_RETRIES=30
RETRY_INTERVAL=5

echo "==> Waiting for GraphDB to become available at ${GRAPHDB_URL}..."

i=0
until curl -sf "${GRAPHDB_URL}/rest/repositories" > /dev/null 2>&1; do
  i=$((i + 1))
  if [ "$i" -ge "$MAX_RETRIES" ]; then
    echo "ERROR: GraphDB did not become available after $((MAX_RETRIES * RETRY_INTERVAL))s. Giving up."
    exit 1
  fi
  echo "  Attempt $i/$MAX_RETRIES — retrying in ${RETRY_INTERVAL}s..."
  sleep "$RETRY_INTERVAL"
done

echo "==> GraphDB is up."

# --- Create repository (idempotent) ---
if curl -sf "${GRAPHDB_URL}/rest/repositories/${REPO_ID}" > /dev/null 2>&1; then
  echo "==> Repository '${REPO_ID}' already exists — skipping creation."
else
  echo "==> Creating repository '${REPO_ID}'..."
  HTTP_STATUS=$(curl -s -o /tmp/curl_response.txt -w "%{http_code}" \
    -X POST "${GRAPHDB_URL}/rest/repositories" \
    -H "Content-Type: multipart/form-data" \
    -F "config=@${CONFIG_FILE};type=text/turtle")

  if [ "$HTTP_STATUS" -eq 201 ]; then
    echo "==> Repository '${REPO_ID}' created successfully."
  else
    echo "ERROR: Failed to create repository. HTTP status: ${HTTP_STATUS}"
    echo "Response: $(cat /tmp/curl_response.txt)"
    exit 1
  fi
fi

# --- Load EPOS SHACL shapes into the SHACL shapes graph ---
echo "==> Loading EPOS DCAT-AP SHACL shapes from: ${SHACL_URL}"

# Download shapes file
curl -sf -o /tmp/epos-shacl-shapes.ttl "${SHACL_URL}" || {
  echo "ERROR: Failed to download SHACL shapes from ${SHACL_URL}"
  exit 1
}

echo "==> Uploading shapes to SHACL graph <${SHACL_GRAPH}>..."

HTTP_STATUS=$(curl -s -o /tmp/shacl_response.txt -w "%{http_code}" \
  -X PUT "${GRAPHDB_URL}/repositories/${REPO_ID}/statements?context=${SHACL_GRAPH_ENCODED}" \
  -H "Content-Type: text/turtle" \
  --data-binary @/tmp/epos-shacl-shapes.ttl) || {
  CURL_EXIT=$?
  echo "ERROR: curl failed while uploading SHACL shapes (exit code: ${CURL_EXIT})"
  exit 1
}

if [ "$HTTP_STATUS" -eq 204 ]; then
  echo "==> EPOS DCAT-AP SHACL shapes loaded successfully."
else
  echo "ERROR: Failed to upload SHACL shapes. HTTP status: ${HTTP_STATUS}"
  echo "Response: $(cat /tmp/shacl_response.txt)"
  exit 1
fi
