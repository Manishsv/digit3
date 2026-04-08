#!/usr/bin/env bash
# Account setup: tenant/realm + platform resources (boundaries, core registry, idgen for registry).
#
# Phases:
#   1) Create tenant (POST /account/v1) — no JWT.
#   2) Platform (needs Keycloak OAuth client + user): boundaries, core registry schema, registryId idgen template.
#
# Usage:
#   ./account-setup.sh                    # tenant only (or tenant + platform if OAUTH_* already set)
#   ./account-setup.sh --platform-only    # skip tenant; run platform steps only (after Keycloak bootstrap)
#
# After first run: set KEYCLOAK_REALM from printed tenant code, configure OAuth client in Keycloak, fill OAUTH_*,
# then: ./account-setup.sh --platform-only

set -euo pipefail

PROVISION_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$PROVISION_ROOT/_common.sh"
provision_load_env

PLATFORM_ONLY=0
for arg in "$@"; do
  if [[ "$arg" == "--platform-only" ]]; then PLATFORM_ONLY=1; fi
done

create_tenant() {
  : "${ACCOUNT_BASE_URL:?Set ACCOUNT_BASE_URL}"
  : "${ACCOUNT_X_CLIENT_ID:?Set ACCOUNT_X_CLIENT_ID}"
  : "${TENANT_NAME:?Set TENANT_NAME}"
  : "${TENANT_EMAIL:?Set TENANT_EMAIL}"

  local body
  body=$(jq -n \
    --arg name "$TENANT_NAME" \
    --arg email "$TENANT_EMAIL" \
    '{ tenant: { name: $name, email: $email, isActive: true, additionalAttributes: {} } }')

  if [[ -n "${TENANT_CODE:-}" ]]; then
    body=$(echo "$body" | jq --arg c "$TENANT_CODE" '.tenant += {code: $c}')
  fi

  local resp code json
  resp=$(curl -sS -w "\n%{http_code}" -X POST "${ACCOUNT_BASE_URL%/}/account/v1" \
    -H "Content-Type: application/json" \
    -H "X-Client-Id: ${ACCOUNT_X_CLIENT_ID}" \
    -d "$body")

  code=$(echo "$resp" | tail -n1)
  json=$(echo "$resp" | sed '$d')

  if [[ "$code" != "201" ]]; then
    echo "Account API failed (HTTP $code): $json" >&2
    exit 1
  fi

  echo "$json" | jq . 2>/dev/null || echo "$json"

  local tenant_code=""
  if command -v jq >/dev/null 2>&1; then
    tenant_code=$(echo "$json" | jq -r '.tenants[0].code // empty')
  fi
  if [[ -n "$tenant_code" ]]; then
    echo >&2
    echo "TENANT_CODE / realm: $tenant_code" >&2
    echo "Set KEYCLOAK_REALM=$tenant_code in env.provision, configure OAuth client + user in Keycloak, then:" >&2
    echo "  ./account-setup.sh --platform-only" >&2
  fi
}

run_platform() {
  : "${KEYCLOAK_REALM:?Set KEYCLOAK_REALM (tenant code)}"
  : "${BOUNDARY_BASE_URL:?Set BOUNDARY_BASE_URL}"
  : "${REGISTRY_BASE_URL:?Set REGISTRY_BASE_URL}"
  : "${IDGEN_BASE_URL:?Set IDGEN_BASE_URL}"

  local tok
  tok=$(provision_get_token)
  provision_extract_jwt "$tok" || exit 1
  echo "Obtained JWT for platform setup (length ${#JWT})."

  EXAMPLES="$(provision_examples_dir)"
  local boundaries_file core_registry
  boundaries_file="${CORE_BOUNDARIES_FILE:-${BOUNDARIES_FILE:-$EXAMPLES/example-boundaries.yaml}}"
  core_registry="${CORE_REGISTRY_SCHEMA_FILE:-$PROVISION_ROOT/examples/core-registry-schema.yaml}"

  if [[ -f "$boundaries_file" ]]; then
    echo "=== Boundaries (platform) ==="
    provision_run_digit create-boundaries --file "$boundaries_file" --server "$BOUNDARY_BASE_URL" --jwt-token "$JWT"
  else
    echo "Skip boundaries (missing): $boundaries_file"
  fi

  if [[ -f "$core_registry" ]]; then
    echo "=== Core registry schema ==="
    provision_run_digit create-registry-schema --file "$core_registry" --server "$REGISTRY_BASE_URL" --jwt-token "$JWT"
  else
    echo "Skip core registry (missing): $core_registry"
  fi

  echo "=== IdGen template registryId (registry dependency) ==="
  provision_run_digit create-idgen-template --default --template-code registryId --server "$IDGEN_BASE_URL" --jwt-token "$JWT" || true

  echo "=== Account / platform setup complete ==="
}

if [[ "$PLATFORM_ONLY" -eq 1 ]]; then
  run_platform
  exit 0
fi

create_tenant

# If operator already configured Keycloak + env, run platform in same invocation
if [[ -n "${OAUTH_CLIENT_ID:-}" && -n "${KEYCLOAK_REALM:-}" ]]; then
  echo >&2
  echo "OAUTH_* and KEYCLOAK_REALM set — running platform steps..." >&2
  run_platform
else
  echo >&2
  echo "Skipping platform steps (set KEYCLOAK_REALM + OAUTH_* then run ./account-setup.sh --platform-only)" >&2
fi
