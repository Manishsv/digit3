#!/usr/bin/env bash
# One-shot local reinstall + provision + smoke test for DIGIT demo stack.
#
# What it does:
# - docker compose down -v (local wipe)
# - docker compose up -d (stack)
# - provision tenant + platform + service setup (writes provision/env.provision)
# - runs provision/service-test.sh plus studio/governance end-to-end checks
#
# Requirements:
# - docker + docker compose
# - jq
# - either `digit` CLI in PATH OR go toolchain + DIGIT_CLI_ROOT set (see provision/_common.sh)
#
# Optional:
# - yq (only needed if FACTS_CONTRACT_CODE is set for registry schema gating)
#
# Env overrides:
# - FACTS_CONTRACT_CODE (default: SBL_DEFAULT) used by provision/service-setup.sh registry gate
# - FACTS_CONTRACT_VERSION (default: 1)

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

command -v docker >/dev/null || { echo "docker required" >&2; exit 1; }
command -v jq >/dev/null || { echo "jq required" >&2; exit 1; }

COMPOSE_FILE="$ROOT/docker-compose.yml"
PROVISION_ROOT="$ROOT/provision"

FACTS_CONTRACT_CODE="${FACTS_CONTRACT_CODE:-SBL_DEFAULT}"
FACTS_CONTRACT_VERSION="${FACTS_CONTRACT_VERSION:-1}"

echo "=== Teardown (wipe volumes) ==="
docker compose -f "$COMPOSE_FILE" down -v --remove-orphans

echo "=== Build + up ==="
# Bring up only the minimal set required for provisioning + demo chain.
# (Some optional services in the full compose can fail migrations on certain machines.)
docker compose -f "$COMPOSE_FILE" up -d --build \
  postgres redis \
  keycloak account \
  registry idgen mdms-v2 \
  boundary-service filestore workflow-service \
  governance coordination studio

echo "=== Wait for account + keycloak ==="
# Cold start often resets TCP until the JVM listens; curl -sS prints scary errors to stderr.
# Suppress probe stderr and use short timeouts so we poll without hanging.
_curl_code() {
  curl -sS --connect-timeout 3 --max-time 15 -o /dev/null -w "%{http_code}" "$1" 2>/dev/null || echo "000"
}

account_ok=0
keycloak_ok=0
max_attempts=120
sleep 3
for ((i = 1; i <= max_attempts; i++)); do
  code_acc="$(_curl_code "http://localhost:8094/account/health")"
  code_kc="$(_curl_code "http://localhost:8080/keycloak/realms/master")"
  [[ "$code_acc" =~ ^(200|404)$ ]] && account_ok=1
  [[ "$code_kc" =~ ^(200|302|401|403|404)$ ]] && keycloak_ok=1
  if [[ "$account_ok" -eq 1 && "$keycloak_ok" -eq 1 ]]; then
    echo "Account + Keycloak reachable (account HTTP $code_acc, keycloak realm HTTP $code_kc) after ${i} attempt(s)."
    break
  fi
  if (( i % 15 == 0 )); then
    echo "Still waiting for Account/Keycloak (attempt $i/$max_attempts, account=$code_acc keycloak_realm=$code_kc)..."
  fi
  sleep 2
done
if [[ "$account_ok" -ne 1 || "$keycloak_ok" -ne 1 ]]; then
  echo "Timed out waiting for Account (8094) and/or Keycloak (8080)." >&2
  echo "  account/health last HTTP: ${code_acc:-?}" >&2
  echo "  keycloak realm last HTTP: ${code_kc:-?}" >&2
  echo "Try: docker logs keycloak --tail 80 && docker logs account --tail 80" >&2
  exit 1
fi

echo "=== Provision (tenant → platform → service → onboarding → service-test) ==="
export FACTS_CONTRACT_CODE FACTS_CONTRACT_VERSION
"$PROVISION_ROOT/run-full-provision.sh"

echo "=== Extra smoke: studio → apply → governance compute ==="
set -a
# shellcheck source=/dev/null
source "$PROVISION_ROOT/env.provision"
set +a

TENANT="$KEYCLOAK_REALM"
AUTH="Authorization: Bearer dev-local"
TEN_HDR="X-Tenant-ID: ${TENANT}"
CLI_HDR="X-Client-ID: demo-reinstall"

echo "=== Wait for studio + governance (provision can take minutes; containers may still be binding) ==="
studio_ok=0
gov_ok=0
smoke_max=90
for ((s = 1; s <= smoke_max; s++)); do
  code_st="$(_curl_code "http://localhost:8107/health")"
  code_gv="$(_curl_code "http://localhost:8098/health")"
  [[ "$code_st" == "200" ]] && studio_ok=1
  [[ "$code_gv" == "200" ]] && gov_ok=1
  if [[ "$studio_ok" -eq 1 && "$gov_ok" -eq 1 ]]; then
    echo "Studio + governance healthy (studio HTTP $code_st, governance HTTP $code_gv) after ${s} attempt(s)."
    break
  fi
  if (( s % 15 == 0 )); then
    echo "Still waiting for Studio (8107/health) and/or Governance (8098/health) (attempt $s/$smoke_max, studio=$code_st governance=$code_gv)..."
  fi
  sleep 2
done
if [[ "$studio_ok" -ne 1 || "$gov_ok" -ne 1 ]]; then
  echo "Timed out waiting for Studio (localhost:8107) and/or Governance (localhost:8098)." >&2
  echo "  studio /health last HTTP: ${code_st:-?}" >&2
  echo "  governance /health last HTTP: ${code_gv:-?}" >&2
  echo "Try: docker ps -a --filter name=studio --filter name=governance && docker logs studio --tail 80 && docker logs governance --tail 80" >&2
  exit 1
fi

curl -sS --connect-timeout 5 --max-time 30 "http://localhost:8107/studio/v1/whoami" -H "$AUTH" -H "$TEN_HDR" -H "$CLI_HDR" >/dev/null
curl -sS --connect-timeout 5 --max-time 30 "http://localhost:8098/governance/v1/whoami" -H "$AUTH" -H "$TEN_HDR" -H "$CLI_HDR" >/dev/null

RULES_YAML=$(cat <<'YAML'
ruleset:
  code: DEMO_SBL
  version: "1.0"
inputs: {}
rules:
  - id: r1
    predicate: eq
    args:
      path: application.status
      value: SUBMITTED
    outcome:
      status: ELIGIBLE
    reason: submitted
YAML
)

echo "Creating Studio service + bundle..."
svc_resp="$(curl -sS -X POST "http://localhost:8107/studio/v1/services" \
  -H "Content-Type: application/json" -H "$AUTH" -H "$TEN_HDR" -H "$CLI_HDR" \
  -d "$(jq -n '{serviceCode:"PGR67",name:"Demo Service",moduleType:"SERVICE",status:"ENABLED",metadata:{}}')")"

bundle_resp="$(curl -sS -X POST "http://localhost:8107/studio/v1/bundles" \
  -H "Content-Type: application/json" -H "$AUTH" -H "$TEN_HDR" -H "$CLI_HDR" \
  -d "$(jq -n --arg cc "$FACTS_CONTRACT_CODE" --arg cv "$FACTS_CONTRACT_VERSION" --arg ry "$RULES_YAML" \
        '{serviceCode:"PGR67",version:"1",status:"DRAFT",factsContractCode:$cc,factsContractVersion:$cv,rulesets:[{yamlText:$ry,issuerAuthorityId:"REG-DEMO"}],metadata:{}}')")"

bundle_id="$(echo "$bundle_resp" | jq -r '.bundleId // empty')"
[[ -n "$bundle_id" ]] || { echo "Bundle create failed: $bundle_resp" >&2; exit 1; }

echo "Applying bundle via Studio job..."
job_resp="$(curl -sS -X POST "http://localhost:8107/studio/v1/jobs" \
  -H "Content-Type: application/json" -H "$AUTH" -H "$TEN_HDR" -H "$CLI_HDR" \
  -d "$(jq -n --arg bid "$bundle_id" '{serviceCode:"PGR67",bundleId:$bid,action:"APPLY_RULESETS"}')")"

job_status="$(echo "$job_resp" | jq -r '.status // empty')"
[[ "$job_status" == "SUCCEEDED" ]] || { echo "Studio job failed: $job_resp" >&2; exit 1; }

ruleset_id="$(echo "$job_resp" | jq -r '.results.publishedRulesets[0].rulesetId // empty')"
ruleset_ver="$(echo "$job_resp" | jq -r '.results.publishedRulesets[0].version // empty')"
ruleset_reg="$(echo "$job_resp" | jq -r '.results.publishedRulesets[0].registryRecordId // empty')"
ruleset_code="$(echo "$job_resp" | jq -r '.results.publishedRulesets[0].code // empty')"
[[ -n "$ruleset_id" && -n "$ruleset_ver" ]] || { echo "Missing ruleset info from job: $job_resp" >&2; exit 1; }

echo "=== Published ruleset (use for coordination governance:decide / decisions:compute) ==="
echo "rulesetId=$ruleset_id"
echo "version=$ruleset_ver"
echo "code=${ruleset_code:-}"
echo "registryRecordId=${ruleset_reg:-}"
echo "publishedRulesets (full):"
echo "$job_resp" | jq '.results.publishedRulesets // []'

{
  echo "RULESET_ID=$ruleset_id"
  echo "RULESET_VERSION=$ruleset_ver"
  echo "RULESET_REGISTRY_ID=$ruleset_reg"
  echo "RULESET_CODE=$ruleset_code"
} >"$PROVISION_ROOT/last-published-ruleset.env"
echo "Wrote $PROVISION_ROOT/last-published-ruleset.env (source before governance-e2e-flow.sh)"

echo "Computing decision (no inline rulesYaml; load from Registry; contract auto-applies)..."
ruleset_json="$(jq -n --arg rid "$ruleset_id" --arg rv "$ruleset_ver" --arg rreg "$ruleset_reg" \
  '{rulesetId:$rid,version:$rv} + (if ($rreg|length) > 0 then {registryRecordId:$rreg} else {} end)')"
dec_resp="$(curl -sS -X POST "http://localhost:8098/governance/v1/decisions:compute" \
  -H "Content-Type: application/json" -H "$AUTH" -H "$TEN_HDR" -H "$CLI_HDR" \
  -d "$(jq -n --argjson rs "$ruleset_json" \
        '{decisionType:"SBL_LICENSE",correlationId:"corr-1",requestId:"req-1",caseRef:{system:"coordination",entityType:"Case",entityId:"CASE-001"},ruleset:$rs,factsSnapshot:{application:{status:"SUBMITTED"}}}')")"

outcome_status="$(echo "$dec_resp" | jq -r '.outcome.status // empty')"
[[ "$outcome_status" == "ELIGIBLE" ]] || { echo "Decision compute failed/unexpected: $dec_resp" >&2; exit 1; }

echo "=== Reinstall complete: all checks passed ==="

