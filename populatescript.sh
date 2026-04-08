#!/bin/bash

get_host_ip() {
  case "$(uname -s)" in
    Darwin)
      ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null
      ;;
    Linux)
      ip route get 1.1.1.1 2>/dev/null | awk '/src/ {print $7}'
      ;;
    MINGW*|CYGWIN*|MSYS*)
      ipconfig | grep -A1 "Ethernet\|Wi-Fi" | grep "IPv4" | head -1 | awk '{print $NF}' | tr -d '\r'
      ;;
    *)
      echo ""
      ;;
  esac
}

HOST_IP=$(get_host_ip)

if [ -z "$HOST_IP" ]; then
  echo "❌ Impossible get the host IP"
  exit 1
fi

echo "🌐 IP: $HOST_IP"

# Start container
docker run -idt -p 5050:80 --name cache epos-ci.brgm.fr:5005/epos/metadata-cache:test-ref

# Wait for container to be ready
echo "Waiting for the container to be ready.."
sleep 2

# Sed inside the container
echo "Sed metadata-cache with $HOST_IP:5050 in index.txt..."
docker exec cache sed -i "s|metadata-cache|$HOST_IP:5050|g" /opt/metadata-cache/index.txt

echo "✅ Done.:"
docker exec cache cat /opt/metadata-cache/index.txt

curl -X 'POST' \
  'http://localhost:8080/api/v2/admin/populate/from-url' \
  -H 'accept: */*' \
  -H 'Content-Type: application/json' \
  -d '{
  "indexUrl": "http://'$HOST_IP':5050/index.txt"
}'