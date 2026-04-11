from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8", extra="ignore")

    registry_base_url: str = "http://localhost:8085"
    mdms_base_url: str = "http://localhost:8084"
    idgen_base_url: str = "http://localhost:8080"
    filestore_base_url: str = "http://localhost:8083"
    notification_base_url: str = "http://localhost:8086"

    governance_db_path: str = "./data/governance_index.db"

    idgen_org_variable: str = "REGISTRY"

    facts_contract_mdms_schema: str = "governance.factsContract"

    skip_jwt_signature_verify: bool = True

    # Local demo: accept fixed bearer token and grant all governance roles (no Keycloak).
    dev_auth_enabled: bool = False  # set DEV_AUTH_ENABLED=true
    dev_auth_token: str = "dev-local"

