#!/usr/bin/env bash
# Service setup: module/case configuration — MDMS, case-type registry, filestore categories, workflow.
# Run after account-setup (including --platform-only). Requires JWT (same OAUTH_* as platform).
#
# Usage:
#   set -a && source env.provision && set +a
#   ./service-setup.sh
#
# Override YAML paths via env (see env.example).

set -euo pipefail

PROVISION_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$PROVISION_ROOT/_common.sh"
provision_load_env

: "${KEYCLOAK_ORIGIN:?Set KEYCLOAK_ORIGIN}"
: "${KEYCLOAK_REALM:?Set KEYCLOAK_REALM}"
: "${MDMS_BASE_URL:?Set MDMS_BASE_URL}"
: "${REGISTRY_BASE_URL:?Set REGISTRY_BASE_URL}"
: "${FILESTORE_BASE_URL:?Set FILESTORE_BASE_URL}"
: "${WORKFLOW_BASE_URL:?Set WORKFLOW_BASE_URL}"

tok=$(provision_get_token)
provision_extract_jwt "$tok" || exit 1
echo "Obtained JWT (length ${#JWT})."

EXAMPLES="$(provision_examples_dir)"
MDMS_SCHEMA_FILE="${MDMS_SCHEMA_FILE:-$EXAMPLES/example-schema.yaml}"
MDMS_DATA_FILE="${MDMS_DATA_FILE:-$EXAMPLES/example-mdms-data.yaml}"
CASE_REGISTRY_SCHEMA_FILE="${CASE_REGISTRY_SCHEMA_FILE:-$PROVISION_ROOT/examples/case-registry-schema.yaml}"
WORKFLOW_FILE="${WORKFLOW_FILE:-$EXAMPLES/example-workflow.yaml}"
FACTS_CONTRACT_CODE="${FACTS_CONTRACT_CODE:-}"
FACTS_CONTRACT_VERSION="${FACTS_CONTRACT_VERSION:-1}"

if [[ -f "$MDMS_SCHEMA_FILE" ]]; then
  echo "=== MDMS schema (module masters) ==="
  provision_run_digit create-schema --file "$MDMS_SCHEMA_FILE" --server "$MDMS_BASE_URL" --jwt-token "$JWT"
else
  echo "Skip MDMS schema (missing): $MDMS_SCHEMA_FILE"
fi

if [[ -f "$MDMS_DATA_FILE" ]]; then
  echo "=== MDMS data ==="
  provision_run_digit create-mdms-data --file "$MDMS_DATA_FILE" --server "$MDMS_BASE_URL" --jwt-token "$JWT"
else
  echo "Skip MDMS data (missing): $MDMS_DATA_FILE"
fi

if [[ -f "$CASE_REGISTRY_SCHEMA_FILE" ]]; then
  echo "=== Case-type registry schema ==="
  if [[ -n "$FACTS_CONTRACT_CODE" ]]; then
    "$PROVISION_ROOT/create-registry-schema-with-contract.sh" \
      --file "$CASE_REGISTRY_SCHEMA_FILE" \
      --contract "$FACTS_CONTRACT_CODE" \
      --version "$FACTS_CONTRACT_VERSION"
  else
    provision_run_digit create-registry-schema --file "$CASE_REGISTRY_SCHEMA_FILE" --server "$REGISTRY_BASE_URL" --jwt-token "$JWT"
  fi
else
  echo "Skip case registry (missing): $CASE_REGISTRY_SCHEMA_FILE"
fi

echo "=== Filestore document category (case attachments) ==="
provision_run_digit create-document-category \
  --type ComplaintEvidence \
  --code COMPLAINT_ATTACHMENT \
  --allowed-formats "pdf,jpg,jpeg,png" \
  --server "$FILESTORE_BASE_URL" \
  --jwt-token "$JWT"

if [[ -f "$WORKFLOW_FILE" ]]; then
  echo "=== Workflow ==="
  provision_run_digit create-workflow --file "$WORKFLOW_FILE" --server "$WORKFLOW_BASE_URL" --jwt-token "$JWT"
else
  echo "Skip workflow (missing): $WORKFLOW_FILE"
fi

echo "=== Service setup complete ==="
