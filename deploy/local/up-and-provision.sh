#!/usr/bin/env bash
# Bring up local DIGIT stack (docker compose) and run full tenant provision + one complaint.
# Requires: Docker, jq, curl, python3, Go (for digit-cli via go run in provision scripts).
#
# Usage: ./up-and-provision.sh
# From repo: digit3/deploy/local/

set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

command -v docker >/dev/null || { echo "docker required" >&2; exit 1; }
command -v jq >/dev/null || { echo "jq required" >&2; exit 1; }

echo "=== docker compose up -d ==="
docker compose up -d

echo "=== wait for postgres ==="
for _ in $(seq 1 90); do
  if docker compose exec -T postgres pg_isready -U postgres >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

echo "=== wait for keycloak (8080) ==="
for _ in $(seq 1 90); do
  code=$(curl -sS -o /dev/null -w "%{http_code}" "http://127.0.0.1:8080/" 2>/dev/null || true)
  if [[ "$code" =~ ^(200|302|303|401|403)$ ]]; then
    break
  fi
  sleep 2
done

echo "=== provision new tenant + platform + services ==="
"$ROOT/provision/run-full-provision.sh"

echo "=== complaint (workflow + registry) ==="
set -a
# shellcheck source=/dev/null
source "$ROOT/provision/env.provision"
set +a
"$ROOT/provision/create-complaint.sh" "CS-DEPLOY-$(date +%s)"

echo "=== done ==="
