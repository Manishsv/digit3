#!/usr/bin/env bash
set -euo pipefail
PROVISION_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$PROVISION_ROOT/_common.sh"
provision_load_env

: "${KEYCLOAK_REALM:?Set KEYCLOAK_REALM via env.provision}"

LAST_RS="$PROVISION_ROOT/last-published-ruleset.env"
if [[ -z "${RULESET_ID:-}" || -z "${RULESET_VERSION:-}" ]] && [[ -f "$LAST_RS" ]]; then
  set -a
  # shellcheck source=/dev/null
  source "$LAST_RS"
  set +a
  echo "Loaded RULESET_* from $LAST_RS"
fi

if [[ -z "${RULESET_ID:-}" || -z "${RULESET_VERSION:-}" ]]; then
  echo "Missing RULESET_ID or RULESET_VERSION." >&2
  echo "  export RULESET_ID=RUL-0001 RULESET_VERSION=1.0" >&2
  echo "  optional: export RULESET_REGISTRY_ID=REGISTRY-..." >&2
  echo "  or run deploy/local/reinstall.sh first; it writes $LAST_RS" >&2
  exit 1
fi

GOV="${GOVERNANCE_BASE_URL:-http://localhost:8098}/governance/v1"
CASE_ID="${CASE_ID:-CS-E2E-001}"
FACTS_CC="${FACTS_CONTRACT_CODE:-SBL_DEFAULT}"
FACTS_CV="${FACTS_CONTRACT_VERSION:-1}"
REG_REC="${RULESET_REGISTRY_ID:-}"

AUTH="${GOVERNANCE_AUTH_TOKEN:-dev-local}"
HDR_AUTH="Authorization: Bearer ${AUTH}"
HDR_TEN="X-Tenant-ID: ${KEYCLOAK_REALM}"
HDR_CLI="X-Client-ID: ${GOVERNANCE_X_CLIENT_ID:-governance-e2e-demo}"

ruleset_json="$(jq -n --arg rid "$RULESET_ID" --arg rv "$RULESET_VERSION" --arg rreg "$REG_REC" \
  '{rulesetId:$rid,version:$rv} + (if ($rreg|length) > 0 then {registryRecordId:$rreg} else {} end)')"

echo "=== 1) Compute — facts do not match demo rule (semantic reject → NO_MATCH) ==="
body1="$(jq -n --argjson rs "$ruleset_json" \
  --arg tid "$KEYCLOAK_REALM" --arg cc "$FACTS_CC" --arg cv "$FACTS_CV" --arg caseId "$CASE_ID" \
  '{decisionType:"SBL_LICENSE",correlationId:"e2e-1",requestId:"e2e-req-1",channel:"web",
    caseRef:{system:"coordination",entityType:"Case",entityId:$caseId,tenantId:$tid},
    ruleset:$rs,factsContractCode:$cc,factsContractVersion:$cv,
    factsSnapshot:{application:{status:"DRAFT"}},mdmsFactChecks:[]}')"
resp1="$(curl -sS -X POST "$GOV/decisions:compute" -H "Content-Type: application/json" \
  -H "$HDR_AUTH" -H "$HDR_TEN" -H "$HDR_CLI" -d "$body1")"
echo "$resp1" | jq .
out1="$(echo "$resp1" | jq -r '.outcome.status // empty')"
[[ "$out1" == "NO_MATCH" ]] || { echo "Expected outcome NO_MATCH for DRAFT facts, got: $out1" >&2; exit 1; }
DEC1="$(echo "$resp1" | jq -r '.decisionId')"
RCP1="$(echo "$resp1" | jq -r '.receiptId')"

echo "=== 2) Appeal (against that decision / receipt) ==="
body2="$(jq -n --arg rcp "$RCP1" --arg dec "$DEC1" \
  '{receiptId:$rcp,decisionId:$dec,filedBy:"citizen-1",grounds:"Request review after NO_MATCH",status:"FILED"}')"
resp2="$(curl -sS -X POST "$GOV/appeals" -H "Content-Type: application/json" \
  -H "$HDR_AUTH" -H "$HDR_TEN" -H "$HDR_CLI" -d "$body2")"
echo "$resp2" | jq .
APL="$(echo "$resp2" | jq -r '.appealId')"

echo "=== 3) Order (adjudicator — REMAND) ==="
body3="$(jq -n --arg apl "$APL" --arg dec "$DEC1" --arg rcp "$RCP1" \
  '{appealId:$apl,decisionId:$dec,receiptId:$rcp,issuedBy:"authority-1",outcome:"REMAND",instructions:"Re-run eligibility with corrected facts"}')"
resp3="$(curl -sS -X POST "$GOV/orders" -H "Content-Type: application/json" \
  -H "$HDR_AUTH" -H "$HDR_TEN" -H "$HDR_CLI" -d "$body3")"
echo "$resp3" | jq .
ORD="$(echo "$resp3" | jq -r '.orderId')"

echo "=== 4) Recompute — lineage links parent + appeal + order; facts now match rule (ELIGIBLE) ==="
body4="$(jq -n --argjson rs "$ruleset_json" \
  --arg tid "$KEYCLOAK_REALM" --arg cc "$FACTS_CC" --arg cv "$FACTS_CV" \
  --arg pdec "$DEC1" --arg apl "$APL" --arg ord "$ORD" \
  --arg caseId "$CASE_ID" \
  '{decisionType:"SBL_LICENSE",correlationId:"e2e-2",requestId:"e2e-req-2",channel:"web",
    parentDecisionId:$pdec,appealId:$apl,orderId:$ord,
    caseRef:{system:"coordination",entityType:"Case",entityId:$caseId,tenantId:$tid},
    ruleset:$rs,factsContractCode:$cc,factsContractVersion:$cv,
    factsSnapshot:{application:{status:"SUBMITTED"}},mdmsFactChecks:[]}')"
resp4="$(curl -sS -X POST "$GOV/decisions:recompute" -H "Content-Type: application/json" \
  -H "$HDR_AUTH" -H "$HDR_TEN" -H "$HDR_CLI" -d "$body4")"
echo "$resp4" | jq .
out4="$(echo "$resp4" | jq -r '.outcome.status // empty')"
[[ "$out4" == "ELIGIBLE" ]] || { echo "Expected ELIGIBLE after SUBMITTED facts, got: $out4" >&2; exit 1; }
RCP2="$(echo "$resp4" | jq -r '.receiptId')"

echo "=== 5) Audit — fetch receipts from local index (immutable payloads in Registry) ==="
curl -sS "$GOV/decisions/$RCP1" -H "$HDR_AUTH" -H "$HDR_TEN" -H "$HDR_CLI" | jq .
curl -sS "$GOV/decisions/$RCP2" -H "$HDR_AUTH" -H "$HDR_TEN" -H "$HDR_CLI" | jq .

echo "=== Done. Trace/receipt registry ids from steps 1 and 4 are in the JSON above (traceRegistryRecordId / receiptRegistryRecordId). ==="
