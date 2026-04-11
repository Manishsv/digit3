from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8", extra="ignore")

    registry_base_url: str = "http://localhost:8085"
    idgen_base_url: str = "http://localhost:8080"
    mdms_base_url: str = "http://localhost:8084"
    governance_base_url: str = "http://localhost:8081"

    studio_db_path: str = "./data/studio_index.db"
    idgen_org_variable: str = "REGISTRY"

    # Dev mode: accept fixed bearer token; grants access to Studio APIs.
    dev_auth_enabled: bool = False
    dev_auth_token: str = "dev-local"

