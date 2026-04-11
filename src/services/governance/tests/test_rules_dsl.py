import pytest

from app.rules_dsl import RulesDslError, compile_ruleset, evaluate_compiled_ruleset


def test_compile_requires_meta():
    with pytest.raises(RulesDslError):
        compile_ruleset("rules: []")


def test_compile_and_evaluate_eq_present():
    y = """
ruleset:
  code: SBL_LICENSE
  version: "1.0.0"
inputs: {}
rules:
  - id: missing
    predicate: eq
    args: { path: "facts.fireNocId", value: "" }
    outcome: { status: "REJECTED", code: "MISSING_FIRE_NOC" }
  - id: present
    predicate: present
    args: { path: "facts.fireNocId" }
    outcome: { status: "APPROVED" }
"""
    compiled = compile_ruleset(y)
    out, expl = evaluate_compiled_ruleset(compiled, {"facts": {"fireNocId": ""}})
    assert out["status"] == "REJECTED"
    assert expl[0]["matched"] is True

