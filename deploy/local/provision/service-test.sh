#!/usr/bin/env bash
# Service testing: smoke checks after account-setup → service-setup → onboarding.
#
# Usage:
#   set -a && source env.provision && set +a
#   ./service-test.sh
#
# Exits non-zero if a critical check fails.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT/env.provision}"
if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck source=/dev/null
  source "$ENV_FILE"
  set +a
fi

: "${KEYCLOAK_ORIGIN:?Set KEYCLOAK_ORIGIN}"
: "${KEYCLOAK_REALM:?Set KEYCLOAK_REALM}"
: "${OAUTH_CLIENT_ID:?Set OAUTH_CLIENT_ID}"
: "${OAUTH_CLIENT_SECRET:?Set OAUTH_CLIENT_SECRET}"
: "${OAUTH_USERNAME:?Set OAUTH_USERNAME}"
: "${OAUTH_PASSWORD:?Set OAUTH_PASSWORD}"

FILESTORE_BASE_URL="${FILESTORE_BASE_URL:-http://localhost:8102}"
WORKFLOW_BASE_URL="${WORKFLOW_BASE_URL:-http://localhost:8085}"
BOUNDARY_BASE_URL="${BOUNDARY_BASE_URL:-http://localhost:8093}"
REGISTRY_BASE_URL="${REGISTRY_BASE_URL:-http://localhost:8104}"
MDMS_BASE_URL="${MDMS_BASE_URL:-http://localhost:8099}"
IDGEN_BASE_URL="${IDGEN_BASE_URL:-http://localhost:8100}"

fail=0
ok() { echo "OK  $*"; }
bad() { echo "FAIL $*" >&2; fail=1; }

# --- No auth: filestore health (Java service exposes /filestore/health on container) ---
code=$(curl -sS -o /dev/null -w "%{http_code}" "${FILESTORE_BASE_URL%/}/filestore/health" || true)
if [[ "$code" == "200" ]]; then
  ok "filestore health ($FILESTORE_BASE_URL/filestore/health)"
else
  bad "filestore health HTTP $code"
fi

tok_json=$(curl -sS -X POST "${KEYCLOAK_ORIGIN%/}/keycloak/realms/${KEYCLOAK_REALM}/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=password&client_id=${OAUTH_CLIENT_ID}&client_secret=${OAUTH_CLIENT_SECRET}&username=${OAUTH_USERNAME}&password=${OAUTH_PASSWORD}")

JWT=""
if command -v jq >/dev/null 2>&1; then
  JWT=$(echo "$tok_json" | jq -r '.access_token // empty')
fi
if [[ -z "$JWT" ]]; then
  bad "could not parse JWT (install jq or fix Keycloak credentials). Body: $tok_json"
  exit 1
fi

# Workflow audit user (any stable string is fine for smoke tests)
CLIENT_HDR="${OAUTH_USERNAME:-provision-tester}"

# Registry service expects X-Client-ID = Keycloak subject (UUID)
JWT_SUB=""
if command -v python3 >/dev/null 2>&1; then
  JWT_SUB=$(python3 -c "import sys,json,base64; t=sys.argv[1]; p=t.split('.')[1]; p+='='*((4-len(p)%4)%4); print(json.loads(base64.urlsafe_b64decode(p.encode())).get('sub',''))" "$JWT" 2>/dev/null) || true
fi
[[ -z "$JWT_SUB" ]] && JWT_SUB="$CLIENT_HDR"

# --- Workflow: list processes (empty list is success) ---
code=$(curl -sS -o /tmp/wf.json -w "%{http_code}" "${WORKFLOW_BASE_URL%/}/workflow/v1/process" \
  -H "X-Tenant-ID: ${KEYCLOAK_REALM}" \
  -H "X-Client-Id: ${CLIENT_HDR}" \
  -H "Authorization: Bearer ${JWT}" || true)
if [[ "$code" == "200" ]]; then
  ok "workflow GET /workflow/v1/process"
else
  bad "workflow list HTTP $code $(cat /tmp/wf.json 2>/dev/null || true)"
fi

# --- Boundary: search (may return 400 without query — tolerate 200/400) ---
code=$(curl -sS -o /tmp/b.json -w "%{http_code}" "${BOUNDARY_BASE_URL%/}/boundary/v1" \
  -H "X-Tenant-ID: ${KEYCLOAK_REALM}" \
  -H "X-Client-Id: ${CLIENT_HDR}" \
  -H "Authorization: Bearer ${JWT}" || true)
if [[ "$code" == "200" ]] || [[ "$code" == "400" ]]; then
  ok "boundary GET /boundary/v1 (HTTP $code)"
else
  bad "boundary GET HTTP $code $(cat /tmp/b.json 2>/dev/null || true)"
fi

# --- MDMS: actuator or root — many images expose /actuator/health ---
code=$(curl -sS -o /dev/null -w "%{http_code}" "${MDMS_BASE_URL%/}/actuator/health" || true)
if [[ "$code" == "200" ]]; then
  ok "mdms actuator health"
else
  # fallback: schema endpoint may require auth — just check TCP target responds something
  code2=$(curl -sS -o /dev/null -w "%{http_code}" "${MDMS_BASE_URL%/}/mdms-v2/v1/schema" -H "Authorization: Bearer ${JWT}" || true)
  # 400 = common for GET without required params; still means service responded
  if [[ "$code2" =~ ^(200|401|403|404|405|400)$ ]]; then
    ok "mdms reachable (HTTP $code2 on /mdms-v2/v1/schema)"
  else
    bad "mdms health unclear (actuator $code, schema $code2)"
  fi
fi

# --- IdGen: liveness (no auth) + template search GET ---
code=$(curl -sS -o /dev/null -w "%{http_code}" "${IDGEN_BASE_URL%/}/idgen/health" || true)
if [[ "$code" == "200" ]]; then
  ok "idgen GET /idgen/health"
else
  bad "idgen health HTTP $code (rebuild idgen image if /idgen/health is missing)"
fi

code=$(curl -sS -o /tmp/ig.json -w "%{http_code}" \
  "${IDGEN_BASE_URL%/}/idgen/v1/template?templateCode=registryId" \
  -H "X-Tenant-ID: ${KEYCLOAK_REALM}" \
  -H "X-Client-ID: ${JWT_SUB}" \
  -H "Authorization: Bearer ${JWT}" || true)
if [[ "$code" == "200" ]]; then
  ok "idgen GET /idgen/v1/template?templateCode=registryId"
else
  bad "idgen template search HTTP $code"
fi

# --- Registry: schema probe + optional complaint row (needs JWT sub as X-Client-ID) ---
if [[ -n "${REGISTRY_TEST_SCHEMA:-}" ]]; then
  code=$(curl -sS -o /dev/null -w "%{http_code}" \
    "${REGISTRY_BASE_URL%/}/registry/v1/schema/${REGISTRY_TEST_SCHEMA}" \
    -H "X-Tenant-ID: ${KEYCLOAK_REALM}" \
    -H "X-Client-ID: ${JWT_SUB}" \
    -H "Authorization: Bearer ${JWT}" || true)
  if [[ "$code" == "200" ]]; then
    ok "registry schema $REGISTRY_TEST_SCHEMA"
    if [[ "${REGISTRY_WRITE_SMOKE:-1}" == "1" ]]; then
      smoke_id="SR-SMOKE-$(date +%s)"
      wcode=$(curl -sS -o /tmp/regw.json -w "%{http_code}" -X POST \
        "${REGISTRY_BASE_URL%/}/registry/v1/schema/${REGISTRY_TEST_SCHEMA}/data" \
        -H "Content-Type: application/json" \
        -H "X-Tenant-ID: ${KEYCLOAK_REALM}" \
        -H "X-Client-ID: ${JWT_SUB}" \
        -H "Authorization: Bearer ${JWT}" \
        -d "$(jq -n --arg s "$smoke_id" --arg t "$KEYCLOAK_REALM" '{data:{serviceRequestId:$s,tenantId:$t,serviceCode:"PGR67",description:"provision smoke"}}')" || true)
      if [[ "$wcode" =~ ^2 ]]; then
        ok "registry POST case row ($smoke_id)"
      else
        bad "registry POST smoke HTTP $wcode $(cat /tmp/regw.json 2>/dev/null || true)"
      fi
    fi
  else
    bad "registry schema $REGISTRY_TEST_SCHEMA HTTP $code"
  fi
else
  ok "registry (skipped; set REGISTRY_TEST_SCHEMA in env to verify a schema code)"
fi

if [[ "$fail" -ne 0 ]]; then
  echo "One or more checks failed." >&2
  exit 1
fi
echo "All runnable checks passed."
