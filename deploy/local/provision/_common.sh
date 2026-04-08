# shellcheck shell=bash
# Shared helpers. Each script must set PROVISION_ROOT before sourcing this file.

provision_load_env() {
  : "${PROVISION_ROOT:?PROVISION_ROOT must be set by caller}"
  ENV_FILE="${ENV_FILE:-$PROVISION_ROOT/env.provision}"
  if [[ -f "$ENV_FILE" ]]; then
    set -a
    # shellcheck source=/dev/null
    source "$ENV_FILE"
    set +a
  fi
}

provision_get_token() {
  : "${KEYCLOAK_ORIGIN:?KEYCLOAK_ORIGIN required}"
  : "${KEYCLOAK_REALM:?KEYCLOAK_REALM required}"
  : "${OAUTH_CLIENT_ID:?OAUTH_CLIENT_ID required}"
  : "${OAUTH_CLIENT_SECRET:?OAUTH_CLIENT_SECRET required}"
  : "${OAUTH_USERNAME:?OAUTH_USERNAME required}"
  : "${OAUTH_PASSWORD:?OAUTH_PASSWORD required}"
  curl -sS -X POST "${KEYCLOAK_ORIGIN%/}/keycloak/realms/${KEYCLOAK_REALM}/protocol/openid-connect/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "grant_type=password&client_id=${OAUTH_CLIENT_ID}&client_secret=${OAUTH_CLIENT_SECRET}&username=${OAUTH_USERNAME}&password=${OAUTH_PASSWORD}"
}

provision_extract_jwt() {
  local tok_json=$1
  if ! command -v jq >/dev/null 2>&1; then
    echo "jq is required to parse token JSON" >&2
    return 1
  fi
  JWT=$(echo "$tok_json" | jq -r '.access_token // empty')
  local err
  err=$(echo "$tok_json" | jq -r '.error_description // .error // empty')
  if [[ -z "$JWT" ]]; then
    echo "Failed to obtain JWT: $tok_json" >&2
    return 1
  fi
  if [[ -n "$err" ]] && [[ "$err" != "null" ]]; then
    echo "Keycloak error: $err" >&2
    return 1
  fi
  export JWT
}

provision_run_digit() {
  if command -v digit >/dev/null 2>&1; then
    digit "$@"
    return $?
  fi
  local cli_root="${DIGIT_CLI_ROOT:-}"
  if [[ -z "$cli_root" && -n "${PROVISION_ROOT:-}" ]]; then
    cli_root="$(cd "$PROVISION_ROOT/../../../../digit-client-tools/digit-cli" 2>/dev/null && pwd)"
  fi
  if [[ -n "$cli_root" && -f "$cli_root/go.mod" ]] && command -v go >/dev/null 2>&1; then
    (cd "$cli_root" && go run . "$@")
    return $?
  fi
  echo "[skip] digit not in PATH and go run unavailable — install digit or set DIGIT_CLI_ROOT" >&2
  return 127
}

provision_examples_dir() {
  : "${PROVISION_ROOT:?}"
  echo "${DIGIT_EXAMPLES_DIR:-$PROVISION_ROOT/../../../../digit-client-tools/digit-cli}"
}
