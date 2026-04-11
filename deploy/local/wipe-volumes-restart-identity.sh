#!/usr/bin/env bash
# Wipe local Docker volumes and restart Postgres + Keycloak + Account.
#
# Keycloak stores realms in the same Postgres volume as other services; `docker compose down -v`
# removes postgres_data (and redis/minio/... named volumes in this compose file), so this is a
# full local DB reset — not "Keycloak only".
#
# Usage (from repo machine):
#   ./wipe-volumes-restart-identity.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE=(docker compose -f "$ROOT/docker-compose.yml")

echo "=== Stopping stack and removing named volumes (Keycloak + all DBs) ==="
"${COMPOSE[@]}" down -v --remove-orphans

echo "=== Starting postgres, keycloak, account (migrations run via depends_on) ==="
"${COMPOSE[@]}" up -d postgres keycloak account

echo "=== Done. Keycloak admin: http://localhost:8080/keycloak/admin/ (admin / admin) ==="
echo "Wait until Keycloak finishes first-time DB init (docker logs -f keycloak) before registering tenants."
