from __future__ import annotations

import base64
import json
from typing import Any

from fastapi import Header, HTTPException, Request

# Roles aligned with provision/keycloak/realm-roles-coordination.json
ROLE_ADMIN = "COORDINATION_ADMIN"
ROLE_WRITER = "COORDINATION_WRITER"
ROLE_READER = "COORDINATION_READER"
ROLE_PARTICIPANT_ADMIN = "PARTICIPANT_ADMIN"
ROLE_AUTHORITY_ADMIN = "AUTHORITY_ADMIN"
ROLE_EMITTER = "SYSTEM_EMITTER"

DEV_ALL_ROLES = frozenset(
    {
        ROLE_ADMIN,
        ROLE_WRITER,
        ROLE_READER,
        ROLE_PARTICIPANT_ADMIN,
        ROLE_AUTHORITY_ADMIN,
        ROLE_EMITTER,
    }
)


def _b64url_decode(segment: str) -> bytes:
    pad = "=" * ((4 - len(segment) % 4) % 4)
    return base64.urlsafe_b64decode(segment + pad)


def decode_jwt_payload(token: str) -> dict[str, Any]:
    parts = token.split(".")
    if len(parts) != 3:
        raise HTTPException(status_code=401, detail="Invalid bearer token")
    try:
        return json.loads(_b64url_decode(parts[1]).decode("utf-8"))
    except Exception as exc:  # noqa: BLE001
        raise HTTPException(status_code=401, detail="Cannot decode JWT") from exc


def realm_from_issuer(iss: str | None) -> str | None:
    if not iss:
        return None
    parts = iss.rstrip("/").split("/")
    return parts[-1] if parts else None


def extract_roles(payload: dict[str, Any]) -> set[str]:
    ra = payload.get("realm_access") or {}
    roles = ra.get("roles") or []
    return set(roles) if isinstance(roles, list) else set()


def require_any_role(roles: set[str], *allowed: str) -> None:
    if ROLE_ADMIN in roles:
        return
    if not allowed:
        return
    if not roles.intersection(allowed):
        raise HTTPException(status_code=403, detail="Missing required realm role")


def require_reader(roles: set[str]) -> None:
    require_any_role(
        roles,
        ROLE_READER,
        ROLE_WRITER,
        ROLE_ADMIN,
        ROLE_PARTICIPANT_ADMIN,
        ROLE_AUTHORITY_ADMIN,
    )


def require_writer(roles: set[str]) -> None:
    require_any_role(roles, ROLE_WRITER, ROLE_ADMIN)


def require_emitter(roles: set[str]) -> None:
    require_any_role(roles, ROLE_EMITTER, ROLE_WRITER, ROLE_ADMIN)


async def digit_headers(
    request: Request,
    authorization: str | None = Header(None),
    x_tenant_id: str | None = Header(None, alias="X-Tenant-ID"),
    x_client_id: str | None = Header(None, alias="X-Client-ID"),
) -> dict[str, str]:
    if not authorization or not authorization.lower().startswith("bearer "):
        raise HTTPException(status_code=401, detail="Authorization: Bearer required")
    token = authorization.split(" ", 1)[1].strip()
    settings = getattr(request.app.state, "settings", None)

    if (
        settings
        and getattr(settings, "dev_auth_enabled", False)
        and token == getattr(settings, "dev_auth_token", "dev-local")
    ):
        tenant = x_tenant_id or "COORDINATION"
        client = x_client_id or "dev-client"
        return {
            "Authorization": authorization,
            "X-Tenant-ID": tenant,
            "X-Client-ID": client[:64],
        }

    payload = decode_jwt_payload(token)
    tenant = x_tenant_id or realm_from_issuer(payload.get("iss"))
    if not tenant:
        raise HTTPException(status_code=400, detail="X-Tenant-ID header or JWT iss realm required")
    client = x_client_id or payload.get("preferred_username") or payload.get("sub") or "unknown"
    return {
        "Authorization": authorization,
        "X-Tenant-ID": tenant,
        "X-Client-ID": client[:64],
    }

