# Governance service (demo)

Owns rulesets, decisions (compute + receipts), immutable decision trace emission, and appeals/orders for governed public services.

This service is intended to pair with `src/services/coordination/` and uses the same header model:

- `Authorization: Bearer <jwt>`
- `X-Tenant-ID: <tenant>`
- `X-Client-ID: <client>`

## Configuration


| Variable                    | Default                 | Description                                   |
| --------------------------- | ----------------------- | --------------------------------------------- |
| `REGISTRY_BASE_URL`         | `http://localhost:8085` | DIGIT Registry                                |
| `MDMS_BASE_URL`             | `http://localhost:8084` | DIGIT MDMS v2 (optional)                      |
| `IDGEN_BASE_URL`            | `http://localhost:8080` | DIGIT IdGen                                   |
| `FILESTORE_BASE_URL`        | `http://localhost:8083` | DIGIT Filestore (optional for demo)           |
| `NOTIFICATION_BASE_URL`     | `http://localhost:8086` | DIGIT Notification (optional for demo)        |
| `IDGEN_ORG_VARIABLE`        | `REGISTRY`              | `ORG` variable for `/idgen/v1/generate`       |
| `SKIP_JWT_SIGNATURE_VERIFY` | `true`                  | Demo mode: JWT signature not verified         |
| `DEV_AUTH_ENABLED`          | `false`                 | Accept fixed bearer token and grant all roles |
| `DEV_AUTH_TOKEN`            | `dev-local`             | Fixed bearer token for dev auth               |
| `FACTS_CONTRACT_MDMS_SCHEMA`| `governance.factsContract` | MDMS schema code for published contracts   |

## Decisions (`POST /governance/v1/decisions:compute`)

- **Rules YAML**: Either pass `factsSnapshot.rulesYaml` (inline), **or** omit it and load from Registry using `ruleset.rulesetId` + `ruleset.version` (must match the `governance.ruleset` row). Optionally pass `ruleset.registryRecordId` from `RulesetPublishOut` if `_get` uses that id.
- **MDMS**: Optional `mdmsFactChecks` — list of `{ "path", "category", "schemaCode"? }`. When non-empty, each dot-path under `factsSnapshot` (excluding `rulesYaml`) must equal an MDMS `code` whose row `data.category` matches (same pattern as coordination’s `coordination.vocabulary` fetch).
- **Facts contracts**: Optional `factsContractCode` + `factsContractVersion`. Resolves rows in MDMS (`governance.factsContract` by default), follows `extendsContractCode`, merges `factsJsonSchema` with `allOf`, then validates business facts with **jsonschema** (draft 2020-12).

## Rulesets (`POST /governance/v1/rulesets`)

- Optional `factsContractCode` / `factsContractVersion`: enforces rule `outcome.status` values against the contract’s `allowedOutcomeStatuses` (intersection along the extend chain). Persisted on the Registry row when set.

## Registry schema allow-list

- `POST /governance/v1/contracts:validateRegistrySchema` — body: `factsContractCode`, `factsContractVersion`, `registrySchemaCode`. Fails if the contract defines a non-empty `allowedRegistrySchemaCodes` set and the code is not allowed.

## Run (container)

Exposes port `8080`.