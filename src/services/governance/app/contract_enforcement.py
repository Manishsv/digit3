from __future__ import annotations

from typing import Any

from jsonschema import Draft202012Validator
from jsonschema.exceptions import ValidationError

from app.facts_contract import ResolvedFactsContract
from app.rules_dsl import CompiledRuleset


def validate_facts_against_contract(
    facts: dict[str, Any],
    contract: ResolvedFactsContract,
) -> None:
    """Raise ValueError with message if facts violate contract JSON Schema."""
    schema = contract.facts_json_schema
    if not schema:
        return
    validator = Draft202012Validator(schema)
    try:
        validator.validate(facts)
    except ValidationError as e:
        raise ValueError(f"Facts contract validation failed: {e.message}") from e


def validate_compiled_rules_against_contract(
    compiled: CompiledRuleset,
    contract: ResolvedFactsContract,
) -> None:
    allowed = contract.allowed_outcome_statuses
    if not allowed:
        return
    for r in compiled.compiled.get("rules", []):
        out = r.get("outcome") or {}
        st = out.get("status")
        if st is not None and str(st) not in allowed:
            raise ValueError(
                f"Rule {r.get('id')!r}: outcome.status {st!r} not allowed by contract "
                f"(allowed: {sorted(allowed)})"
            )


def validate_registry_schema_code_allowed(
    registry_schema_code: str,
    contract: ResolvedFactsContract,
) -> None:
    allowed = contract.allowed_registry_schema_codes
    if not allowed:
        return
    if registry_schema_code not in allowed:
        raise ValueError(
            f"Registry schema code {registry_schema_code!r} not allowed by contract "
            f"(allowed: {sorted(allowed)})"
        )
