from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Callable

import yaml


class RulesDslError(ValueError):
    pass


@dataclass(frozen=True)
class CompiledRuleset:
    code: str
    version: str
    inputs: dict[str, Any]
    compiled: dict[str, Any]


PredicateFn = Callable[[dict[str, Any]], bool]


class RuleFunctionRegistry:
    def __init__(self) -> None:
        self._predicates: dict[str, Callable[[dict[str, Any], dict[str, Any]], bool]] = {}

    def register_predicate(self, name: str, fn: Callable[[dict[str, Any], dict[str, Any]], bool]) -> None:
        if not name or not isinstance(name, str):
            raise RulesDslError("Predicate name required")
        self._predicates[name] = fn

    def get_predicate(self, name: str) -> Callable[[dict[str, Any], dict[str, Any]], bool]:
        if name not in self._predicates:
            raise RulesDslError(f"Unknown predicate: {name}")
        return self._predicates[name]


def default_function_registry() -> RuleFunctionRegistry:
    reg = RuleFunctionRegistry()

    def eq(facts: dict[str, Any], args: dict[str, Any]) -> bool:
        path = args.get("path")
        value = args.get("value")
        if not isinstance(path, str) or not path:
            raise RulesDslError("eq predicate requires args.path")
        cur: Any = facts
        for part in path.split("."):
            if not isinstance(cur, dict) or part not in cur:
                return False
            cur = cur[part]
        return cur == value

    def present(facts: dict[str, Any], args: dict[str, Any]) -> bool:
        path = args.get("path")
        if not isinstance(path, str) or not path:
            raise RulesDslError("present predicate requires args.path")
        cur: Any = facts
        for part in path.split("."):
            if not isinstance(cur, dict) or part not in cur:
                return False
            cur = cur[part]
        return cur is not None and cur != ""

    reg.register_predicate("eq", eq)
    reg.register_predicate("present", present)
    return reg


def load_ruleset_yaml(yaml_text: str) -> dict[str, Any]:
    try:
        obj = yaml.safe_load(yaml_text)
    except Exception as exc:  # noqa: BLE001
        raise RulesDslError(f"Invalid YAML: {exc}") from exc
    if not isinstance(obj, dict):
        raise RulesDslError("Ruleset YAML must be a mapping")
    return obj


def compile_ruleset(yaml_text: str, functions: RuleFunctionRegistry | None = None) -> CompiledRuleset:
    functions = functions or default_function_registry()
    obj = load_ruleset_yaml(yaml_text)

    meta = obj.get("ruleset") or {}
    if not isinstance(meta, dict):
        raise RulesDslError("ruleset metadata must be a mapping")

    code = meta.get("code")
    version = meta.get("version")
    if not isinstance(code, str) or not code:
        raise RulesDslError("ruleset.code required")
    if not isinstance(version, str) or not version:
        raise RulesDslError("ruleset.version required")

    inputs = obj.get("inputs") or {}
    if not isinstance(inputs, dict):
        raise RulesDslError("inputs must be a mapping")

    rules = obj.get("rules") or []
    if not isinstance(rules, list) or not rules:
        raise RulesDslError("rules must be a non-empty list")

    compiled_rules: list[dict[str, Any]] = []
    for r in rules:
        if not isinstance(r, dict):
            raise RulesDslError("each rule must be a mapping")
        rid = r.get("id")
        pred = r.get("predicate")
        args = r.get("args") or {}
        outcome = r.get("outcome") or {}
        reason = r.get("reason")
        if not isinstance(rid, str) or not rid:
            raise RulesDslError("rule.id required")
        if not isinstance(pred, str) or not pred:
            raise RulesDslError(f"rule {rid}: predicate required")
        if not isinstance(args, dict):
            raise RulesDslError(f"rule {rid}: args must be a mapping")
        if not isinstance(outcome, dict) or "status" not in outcome:
            raise RulesDslError(f"rule {rid}: outcome.status required")

        _ = functions.get_predicate(pred)

        compiled_rules.append(
            {
                "id": rid,
                "predicate": pred,
                "args": args,
                "outcome": outcome,
                "reason": reason,
            }
        )

    return CompiledRuleset(
        code=code,
        version=version,
        inputs=inputs,
        compiled={"rules": compiled_rules},
    )


def evaluate_compiled_ruleset(
    compiled: CompiledRuleset,
    facts: dict[str, Any],
    functions: RuleFunctionRegistry | None = None,
) -> tuple[dict[str, Any], list[dict[str, Any]]]:
    functions = functions or default_function_registry()
    explanations: list[dict[str, Any]] = []

    final_outcome: dict[str, Any] | None = None
    for r in compiled.compiled.get("rules", []):
        pred_name = r["predicate"]
        args = r.get("args") or {}
        fn = functions.get_predicate(pred_name)
        matched = bool(fn(facts, args))
        e = {"ruleId": r["id"], "matched": matched, "outputs": {}}
        if r.get("reason"):
            e["reason"] = r.get("reason")
        explanations.append(e)
        if matched and final_outcome is None:
            final_outcome = dict(r["outcome"])

    if final_outcome is None:
        final_outcome = {"status": "NO_MATCH"}
    return final_outcome, explanations

