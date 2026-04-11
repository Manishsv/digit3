#!/usr/bin/env bash
# Create a Registry schema only if allowed by a facts contract (MDMS).
#
# Usage:
#   set -a && source env.provision && set +a
#   ./create-registry-schema-with-contract.sh --file examples/case-registry-schema.yaml --contract SBL_DEFAULT --version 1
#
# Notes:
# - This validates schemaCode against Governance `allowedRegistrySchemaCodes` (if the contract defines a non-empty list).
# - If the contract list is empty, the validation endpoint returns OK and this becomes a no-op gate.

set -euo pipefail

PROVISION_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$PROVISION_ROOT/_common.sh"
provision_load_env

: "${KEYCLOAK_ORIGIN:?Set KEYCLOAK_ORIGIN}"
: "${KEYCLOAK_REALM:?Set KEYCLOAK_REALM}"
: "${REGISTRY_BASE_URL:?Set REGISTRY_BASE_URL}"

GOVERNANCE_BASE_URL="${GOVERNANCE_BASE_URL:-http://localhost:8098}"

FILE=""
CONTRACT_CODE=""
CONTRACT_VERSION="1"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --file) FILE="$2"; shift 2 ;;
    --contract) CONTRACT_CODE="$2"; shift 2 ;;
    --version) CONTRACT_VERSION="$2"; shift 2 ;;
    *) echo "Unknown arg: $1" >&2; exit 2 ;;
  esac
done

if [[ -z "$FILE" || ! -f "$FILE" ]]; then
  echo "--file is required and must exist" >&2
  exit 2
fi
if [[ -z "$CONTRACT_CODE" ]]; then
  echo "--contract is required" >&2
  exit 2
fi

tok=$(provision_get_token)
provision_extract_jwt "$tok" || exit 1
echo "Obtained JWT (length ${#JWT})."

SCHEMA_CODE="$(awk -F: '/^[[:space:]]*schemaCode[[:space:]]*:/{v=$2; gsub(/^[[:space:]]+|[[:space:]]+$/,"",v); gsub(/["'\'']/, "", v); print v; exit}' "$FILE")"
if [[ -z "$SCHEMA_CODE" ]]; then
  echo "schemaCode missing in $FILE" >&2
  exit 2
fi

echo "=== Contract gate (governance) ==="
curl -sS -X POST "${GOVERNANCE_BASE_URL%/}/governance/v1/contracts:validateRegistrySchema" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${JWT}" \
  -H "X-Tenant-ID: ${KEYCLOAK_REALM}" \
  -H "X-Client-ID: provision" \
  -d "$(jq -n --arg c "$CONTRACT_CODE" --arg v "$CONTRACT_VERSION" --arg s "$SCHEMA_CODE" \
        '{factsContractCode:$c,factsContractVersion:$v,registrySchemaCode:$s}')" >/dev/null

echo "=== Create registry schema ==="
provision_run_digit create-registry-schema --file "$FILE" --server "$REGISTRY_BASE_URL" --jwt-token "$JWT"

echo "=== Done ==="

