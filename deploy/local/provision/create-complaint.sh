#!/usr/bin/env bash
# Create a complaint workflow instance (APPLY from INIT) for process code PGR67.
#
# After a successful workflow transition, also POSTs a registry row when REGISTRY_BASE_URL is set
# (schema REGISTRY_CASE_SCHEMA, default complaints.case). Set COMPLAINT_SKIP_REGISTRY=1 to skip.
#
# Usage:
#   set -a && source env.provision && set +a
#   ./create-complaint.sh [optional-entity-id]

set -euo pipefail
PROVISION_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$PROVISION_ROOT/_common.sh"
PROVISION_ROOT="$PROVISION_ROOT" provision_load_env

: "${KEYCLOAK_ORIGIN:?}" "${KEYCLOAK_REALM:?}" "${WORKFLOW_BASE_URL:?}"
: "${OAUTH_CLIENT_ID:?}" "${OAUTH_CLIENT_SECRET:?}" "${OAUTH_USERNAME:?}" "${OAUTH_PASSWORD:?}"

REGISTRY_CASE_SCHEMA="${REGISTRY_CASE_SCHEMA:-complaints.case}"

ENTITY_ID="${1:-CS-$(date +%s)}"
WORKFLOW_CODE="${WORKFLOW_CODE:-PGR67}"

tok=$(provision_get_token)
provision_extract_jwt "$tok" || exit 1

SUB=$(python3 -c "import sys,json,base64; p=sys.argv[1].split('.')[1]; p+=('='*((4-len(p)%4)%4)); print(json.loads(base64.urlsafe_b64decode(p.encode())).get('sub',''))" "$JWT")

PROC_ID=$(curl -sS "${WORKFLOW_BASE_URL}/workflow/v1/process?code=${WORKFLOW_CODE}" \
  -H "X-Tenant-ID: ${KEYCLOAK_REALM}" \
  -H "Authorization: Bearer ${JWT}" | jq -r '.[0].id // empty')
if [[ -z "$PROC_ID" || "$PROC_ID" == "null" ]]; then
  echo "No workflow process with code=${WORKFLOW_CODE}" >&2
  exit 1
fi

echo "Creating complaint entityId=$ENTITY_ID process=$WORKFLOW_CODE ($PROC_ID)"

resp=$(curl -sS -w "\n%{http_code}" -X POST "${WORKFLOW_BASE_URL}/workflow/v1/transition" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: ${KEYCLOAK_REALM}" \
  -H "X-Client-Id: ${SUB}" \
  -H "Authorization: Bearer ${JWT}" \
  -d "$(jq -n \
    --arg pid "$PROC_ID" \
    --arg eid "$ENTITY_ID" \
    '{processId:$pid,entityId:$eid,action:"APPLY",init:true,comment:"Complaint submission",attributes:{roles:["CITIZEN"]}}')")

code=$(echo "$resp" | tail -n1)
body=$(echo "$resp" | sed '$d')
echo "$body" | jq . 2>/dev/null || echo "$body"
[[ "$code" =~ ^2 ]] || exit 1

WINST=$(echo "$body" | jq -r '.id // empty')
if [[ -n "${REGISTRY_BASE_URL:-}" && "${COMPLAINT_SKIP_REGISTRY:-0}" != "1" ]]; then
  reg_payload=$(jq -n \
    --arg sid "$ENTITY_ID" \
    --arg tid "$KEYCLOAK_REALM" \
    --arg sc "$WORKFLOW_CODE" \
    --arg pid "$PROC_ID" \
    --arg wid "$WINST" \
    '{data:{serviceRequestId:$sid,tenantId:$tid,serviceCode:$sc,processId:$pid,workflowInstanceId:$wid,description:"Complaint submission",applicationStatus:"SUBMITTED"}}')
  reg_resp=$(curl -sS -w "\n%{http_code}" -X POST "${REGISTRY_BASE_URL%/}/registry/v1/schema/${REGISTRY_CASE_SCHEMA}/data" \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: ${KEYCLOAK_REALM}" \
    -H "X-Client-ID: ${SUB}" \
    -H "Authorization: Bearer ${JWT}" \
    -d "$reg_payload")
  reg_code=$(echo "$reg_resp" | tail -n1)
  reg_body=$(echo "$reg_resp" | sed '$d')
  echo "$reg_body" | jq . 2>/dev/null || echo "$reg_body"
  [[ "$reg_code" =~ ^2 ]] || {
    echo "Registry persist failed HTTP $reg_code" >&2
    exit 1
  }
  echo "OK workflow + registry (HTTP $code / $reg_code)"
else
  echo "OK workflow (HTTP $code)"
  if [[ "${COMPLAINT_SKIP_REGISTRY:-0}" == "1" ]]; then
    echo "(registry skipped: COMPLAINT_SKIP_REGISTRY=1)"
  elif [[ -z "${REGISTRY_BASE_URL:-}" ]]; then
    echo "(set REGISTRY_BASE_URL to persist complaint to registry)"
  fi
fi
