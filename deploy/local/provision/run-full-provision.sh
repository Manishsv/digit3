#!/usr/bin/env bash
# One-shot local provision: tenant (curl) → platform → service → onboarding → service-test.
# Uses realm template defaults: client auth-server / changeme, superuser password default.

set -euo pipefail
PROVISION_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROVISION_ROOT"

command -v jq >/dev/null || { echo "jq required" >&2; exit 1; }

STAMP=$(date +%s)
TENANT_NAME="ProvLocal${STAMP}"
TENANT_EMAIL="admin${STAMP}@provision.local"

echo "=== Create tenant: $TENANT_NAME ==="
body=$(jq -n --arg name "$TENANT_NAME" --arg email "$TENANT_EMAIL" \
  '{tenant:{name:$name,email:$email,isActive:true,additionalAttributes:{}}}')
resp=$(curl -sS -w "\n%{http_code}" -X POST "http://localhost:8094/account/v1" \
  -H "Content-Type: application/json" \
  -H "X-Client-Id: test-client" \
  -d "$body")
code=$(echo "$resp" | tail -n1)
json=$(echo "$resp" | sed '$d')
if [[ "$code" != "201" ]]; then
  echo "Account create failed HTTP $code: $json" >&2
  exit 1
fi
KEYCLOAK_REALM=$(echo "$json" | jq -r '.tenants[0].code // empty')
if [[ -z "$KEYCLOAK_REALM" ]]; then
  echo "No tenant code in response: $json" >&2
  exit 1
fi
echo "KEYCLOAK_REALM=$KEYCLOAK_REALM (superuser $TENANT_EMAIL / default)"

cat > "$PROVISION_ROOT/env.provision" <<EOF
KEYCLOAK_ORIGIN=http://localhost:8080
KEYCLOAK_REALM=${KEYCLOAK_REALM}
OAUTH_CLIENT_ID=auth-server
OAUTH_CLIENT_SECRET=changeme
OAUTH_USERNAME=${TENANT_EMAIL}
OAUTH_PASSWORD=default
ACCOUNT_BASE_URL=http://localhost:8094
ACCOUNT_X_CLIENT_ID=test-client
TENANT_NAME=${TENANT_NAME}
TENANT_EMAIL=${TENANT_EMAIL}
WORKFLOW_BASE_URL=http://localhost:8085
MDMS_BASE_URL=http://localhost:8099
BOUNDARY_BASE_URL=http://localhost:8093
REGISTRY_BASE_URL=http://localhost:8104
FILESTORE_BASE_URL=http://localhost:8102
IDGEN_BASE_URL=http://localhost:8100
REGISTRY_TEST_SCHEMA=complaints.case
EOF

set -a
# shellcheck source=/dev/null
source "$PROVISION_ROOT/env.provision"
set +a

echo "=== Platform (boundaries, core registry, idgen) ==="
./account-setup.sh --platform-only

echo "=== Service setup ==="
./service-setup.sh

echo "=== Onboarding (roles) ==="
./onboarding.sh

echo "=== Service test ==="
./service-test.sh

echo "=== Done. Realm=$KEYCLOAK_REALM user=$TENANT_EMAIL password=default ==="
