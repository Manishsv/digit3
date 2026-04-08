#!/usr/bin/env bash
# Onboarding: Keycloak realm roles + users (digit create-role / create-user / assign-role).
# Requires a JWT for a user that can administer the tenant realm (realm-management or equivalent).
#
# Prerequisites: KEYCLOAK_REALM, KEYCLOAK_ORIGIN, OAUTH_* (bootstrap admin), digit CLI.
#
# Usage:
#   set -a && source env.provision && set +a
#   ./onboarding.sh
#
# Env:
#   ONBOARDING_ROLES     — space-separated realm roles to create (default: workflow roles)
#   ONBOARDING_USERS_CSV — CSV: username,password,email,roles (roles pipe-separated)

set -euo pipefail

PROVISION_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$PROVISION_ROOT/_common.sh"
provision_load_env

: "${KEYCLOAK_ORIGIN:?Set KEYCLOAK_ORIGIN}"
: "${KEYCLOAK_REALM:?Set KEYCLOAK_REALM}"

tok=$(provision_get_token)
provision_extract_jwt "$tok" || exit 1
echo "Using admin JWT for Keycloak onboarding (length ${#JWT})."

# Realm roles aligned with default workflow example (customize via ONBOARDING_ROLES)
ONBOARDING_ROLES="${ONBOARDING_ROLES:-CITIZEN CSR GRO LME}"

for role in $ONBOARDING_ROLES; do
  [[ -z "$role" ]] && continue
  echo "=== create-role: $role ==="
  provision_run_digit create-role \
    --role-name "$role" \
    --account "$KEYCLOAK_REALM" \
    --server "$KEYCLOAK_ORIGIN" \
    --jwt-token "$JWT" || true
done

USERS_CSV="${ONBOARDING_USERS_CSV:-$PROVISION_ROOT/onboarding-users.csv}"
if [[ ! -f "$USERS_CSV" ]]; then
  echo "No user CSV at $USERS_CSV — copy onboarding-users.example.csv, add rows, set ONBOARDING_USERS_CSV or create onboarding-users.csv"
  echo "=== Onboarding (roles only) complete ==="
  exit 0
fi

echo "=== Users from $USERS_CSV ==="
while IFS= read -r line || [[ -n "$line" ]]; do
  [[ -z "${line// }" ]] && continue
  [[ "$line" =~ ^[[:space:]]*# ]] && continue
  [[ "$line" =~ ^username,password,email,roles ]] && continue
  IFS=',' read -r username password email rolecol <<<"$line"
  [[ -z "${username:-}" ]] && continue
  echo "--- user: $username ---"
  provision_run_digit create-user \
    --username "$username" \
    --password "$password" \
    --email "$email" \
    --account "$KEYCLOAK_REALM" \
    --server "$KEYCLOAK_ORIGIN" \
    --jwt-token "$JWT" || true
  if [[ -n "${rolecol:-}" ]]; then
    IFS='|' read -ra RLIST <<<"$rolecol"
    for r in "${RLIST[@]}"; do
      r="${r// /}"
      [[ -z "$r" ]] && continue
      provision_run_digit assign-role \
        --username "$username" \
        --role-name "$r" \
        --account "$KEYCLOAK_REALM" \
        --server "$KEYCLOAK_ORIGIN" \
        --jwt-token "$JWT" || true
    done
  fi
done < "$USERS_CSV"

echo "=== Onboarding complete ==="
