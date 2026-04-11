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
    echo "Set KEYCLOAK_REALM=$tenant_code in env.provision; realm import includes public client demo-ui (Vite dev URIs). Add a test user in Keycloak, then:" >&2
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
  local studio_service_schema studio_bundle_schema
  studio_service_schema="$PROVISION_ROOT/examples/studio-service.schema.yaml"
  studio_bundle_schema="$PROVISION_ROOT/examples/studio-bundle.schema.yaml"
  local gov_ruleset_schema gov_trace_schema gov_receipt_schema gov_order_schema gov_appeal_schema
  gov_ruleset_schema="$PROVISION_ROOT/examples/governance-ruleset.registry-schema.yaml"
  gov_trace_schema="$PROVISION_ROOT/examples/governance-decision-trace.registry-schema.yaml"
  gov_receipt_schema="$PROVISION_ROOT/examples/governance-decision-receipt.registry-schema.yaml"
  gov_order_schema="$PROVISION_ROOT/examples/governance-order.registry-schema.yaml"
  gov_appeal_schema="$PROVISION_ROOT/examples/governance-appeal.registry-schema.yaml"

  if [[ -f "$boundaries_file" ]]; then
    echo "=== Boundaries (platform) ==="
    provision_run_digit create-boundaries --file "$boundaries_file" --server "$BOUNDARY_BASE_URL" --jwt-token "$JWT" || true
  else
    echo "Skip boundaries (missing): $boundaries_file"
  fi

  if [[ -f "$core_registry" ]]; then
    echo "=== Core registry schema ==="
    provision_run_digit create-registry-schema --file "$core_registry" --server "$REGISTRY_BASE_URL" --jwt-token "$JWT" || true
  else
    echo "Skip core registry (missing): $core_registry"
  fi

  if [[ -f "$studio_service_schema" ]]; then
    echo "=== Studio registry schemas ==="
    provision_run_digit create-registry-schema --file "$studio_service_schema" --server "$REGISTRY_BASE_URL" --jwt-token "$JWT" || true
  fi
  if [[ -f "$studio_bundle_schema" ]]; then
    provision_run_digit create-registry-schema --file "$studio_bundle_schema" --server "$REGISTRY_BASE_URL" --jwt-token "$JWT" || true
  fi

  if [[ -f "$gov_ruleset_schema" ]]; then
    echo "=== Registry schemas (governance) ==="
    provision_run_digit create-registry-schema --file "$gov_ruleset_schema" --server "$REGISTRY_BASE_URL" --jwt-token "$JWT" || true
  fi
  if [[ -f "$gov_trace_schema" ]]; then
    provision_run_digit create-registry-schema --file "$gov_trace_schema" --server "$REGISTRY_BASE_URL" --jwt-token "$JWT" || true
  fi
  if [[ -f "$gov_receipt_schema" ]]; then
    provision_run_digit create-registry-schema --file "$gov_receipt_schema" --server "$REGISTRY_BASE_URL" --jwt-token "$JWT" || true
  fi
  if [[ -f "$gov_order_schema" ]]; then
    provision_run_digit create-registry-schema --file "$gov_order_schema" --server "$REGISTRY_BASE_URL" --jwt-token "$JWT" || true
  fi
  if [[ -f "$gov_appeal_schema" ]]; then
    provision_run_digit create-registry-schema --file "$gov_appeal_schema" --server "$REGISTRY_BASE_URL" --jwt-token "$JWT" || true
  fi

  local gov_fc_schema gov_fc_data
  gov_fc_data="$PROVISION_ROOT/examples/governance-facts-contract-mdms-data.yaml"
  gov_fc_schema="$PROVISION_ROOT/examples/governance-facts-contract-mdms-schema.yaml"
  if [[ -n "${MDMS_BASE_URL:-}" ]]; then
    if [[ -f "$gov_fc_schema" ]]; then
      echo "=== MDMS governance.factsContract schema ==="
      provision_run_digit create-schema --file "$gov_fc_schema" --server "$MDMS_BASE_URL" --jwt-token "$JWT"
    fi
    if [[ -f "$gov_fc_data" ]]; then
      echo "=== MDMS governance.factsContract data (sample contracts) ==="
      local gov_fc_data_rendered
      gov_fc_data_rendered="$(mktemp)"
      sed "s|__TENANT_ID__|${KEYCLOAK_REALM}|g" "$gov_fc_data" >"$gov_fc_data_rendered"
      provision_run_digit create-mdms-data --file "$gov_fc_data_rendered" --server "$MDMS_BASE_URL" --jwt-token "$JWT"
      rm -f "$gov_fc_data_rendered"
    fi
  else
    echo "Skip MDMS facts contracts (MDMS_BASE_URL not set)"
  fi

  echo "=== IdGen template registryId (registry dependency) ==="
  provision_run_digit create-idgen-template --default --template-code registryId --server "$IDGEN_BASE_URL" --jwt-token "$JWT" || true

  echo "=== IdGen templates (studio) ==="
  provision_run_digit create-idgen-template --template-code studio.svc --template "SVC-{SEQ}" --scope global --server "$IDGEN_BASE_URL" --jwt-token "$JWT" || true
  provision_run_digit create-idgen-template --template-code studio.bndl --template "BNDL-{SEQ}" --scope global --server "$IDGEN_BASE_URL" --jwt-token "$JWT" || true
  provision_run_digit create-idgen-template --template-code studio.job --template "JOB-{SEQ}" --scope global --server "$IDGEN_BASE_URL" --jwt-token "$JWT" || true

  echo "=== IdGen templates (governance) ==="
  provision_run_digit create-idgen-template --template-code governance.rul --template "RUL-{SEQ}" --scope global --server "$IDGEN_BASE_URL" --jwt-token "$JWT" || true
  provision_run_digit create-idgen-template --template-code governance.dec --template "DEC-{SEQ}" --scope global --server "$IDGEN_BASE_URL" --jwt-token "$JWT" || true
  provision_run_digit create-idgen-template --template-code governance.rcp --template "RCP-{SEQ}" --scope global --server "$IDGEN_BASE_URL" --jwt-token "$JWT" || true
  provision_run_digit create-idgen-template --template-code governance.apl --template "APL-{SEQ}" --scope global --server "$IDGEN_BASE_URL" --jwt-token "$JWT" || true
  provision_run_digit create-idgen-template --template-code governance.ord --template "ORD-{SEQ}" --scope global --server "$IDGEN_BASE_URL" --jwt-token "$JWT" || true

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
