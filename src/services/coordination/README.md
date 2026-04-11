# Coordination thin service

HTTP facade for a DIGIT tenant. Clients call these APIs with the same `Authorization` (Bearer JWT), `X-Tenant-ID`, and `X-Client-ID` headers used for other DIGIT services. The service forwards those headers to **Registry**, **IdGen**, and optionally **MDMS**.

## Local query index

The bundled DIGIT client only exposes registry reads by `registryId`. This service keeps a small **SQLite** file (`COORDINATION_DB_PATH`) to resolve mappings and timelines quickly. **Registry rows remain the durable coordination record**; the index is a query accelerator.

## Configuration


| Variable               | Default                        | Description                                                         |
| ---------------------- | ------------------------------ | ------------------------------------------------------------------- |
| `REGISTRY_BASE_URL`    | `http://localhost:8085`        | Registry service                                                    |
| `MDMS_BASE_URL`        | `http://localhost:8084`        | MDMS v2                                                             |
| `IDGEN_BASE_URL`       | `http://localhost:8080`        | IdGen                                                               |
| `COORDINATION_DB_PATH` | `./data/coordination_index.db` | SQLite index path                                                   |
| `IDGEN_ORG_VARIABLE`   | `REGISTRY`                     | `ORG` variable for `/idgen/v1/generate`                             |
| `VALIDATE_VOCAB`       | `true`                         | Validate codes against embedded vocabulary (aligned with MDMS seed) |
| `GOVERNANCE_BASE_URL`  | `http://localhost:8098`        | Governance service (for `.../governance:decide`)                  |
| `DEV_AUTH_ENABLED`     | `false`                        | Accept fixed bearer token and grant all roles (dev only)            |
| `DEV_AUTH_TOKEN`       | `dev-local`                    | Fixed bearer token for dev auth                                     |

`POST .../v1/cases/{case_id}/governance:decide` forwards to Governance `decisions:compute`. Pass `registryRecordId` and/or `mdmsFactChecks` on the body when you need Registry-backed rulesets or MDMS validation (see governance README).

## Run (container)

Exposes port `8080`.