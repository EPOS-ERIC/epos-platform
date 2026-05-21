#!/bin/bash

set -euo pipefail

get_host_alias() {
  case "$(uname -s)" in
    Darwin|MINGW*|CYGWIN*|MSYS*)
      # Docker Desktop reliably exposes host via this DNS name
      echo "host.docker.internal"
      ;;
    Linux)
      # Linux Docker may not support host.docker.internal by default
      ip route | awk '/default/ {print $3; exit}'
      ;;
    *)
      echo "host.docker.internal"
      ;;
  esac
}

HOST_ALIAS=${HOST_ALIAS:-$(get_host_alias)}

if [ -z "$HOST_ALIAS" ]; then
  echo "❌ Impossible to resolve host alias"
  exit 1
fi

HOST_ENDPOINT="${HOST_ALIAS}:5050"
echo "🌐 Host endpoint for metadata-lifecycle: $HOST_ENDPOINT"

docker rm -f cache || true

# Start container
docker run -idt -p 5050:80 --name cache epos-ci.brgm.fr:5005/epos/metadata-cache:test-ref

# Wait for container to be ready
echo "Waiting for the container to be ready.."
sleep 2

# Sed inside the container
echo "Rewriting all index URLs to http://$HOST_ENDPOINT/..."
docker exec cache sed -E -i "s|https?://[^/]+/|http://$HOST_ENDPOINT/|g" /opt/metadata-cache/index.txt

echo "✅ Done.:"
docker exec cache cat /opt/metadata-cache/index.txt

curl -X 'POST' \
  'http://localhost:8080/api/v2/admin/populate/from-url' \
  -H 'accept: */*' \
  -H 'X-Agent: http://test/agent' \
  -H 'X-Group: http://test/group' \
  -H 'Content-Type: application/json' \
  -d '{
  "indexUrl": "http://'$HOST_ENDPOINT'/index.txt"
}'
