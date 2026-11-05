import argparse
import hashlib
import json
import re
from pathlib import Path

import jsonschema


def expressions(value):
    if isinstance(value, dict):
        for item in value.values():
            yield from expressions(item)
    elif isinstance(value, list):
        for item in value:
            yield from expressions(item)
    elif isinstance(value, str) and value.startswith("@"):
        yield value


def check_expression(value):
    stack = []
    quoted = False
    index = 0
    while index < len(value):
        char = value[index]
        if char == "'":
            if quoted and index + 1 < len(value) and value[index + 1] == "'":
                index += 2
                continue
            quoted = not quoted
        elif not quoted:
            if char in "([":
                stack.append(char)
            elif char in ")]":
                expected = "(" if char == ")" else "["
                if not stack or stack.pop() != expected:
                    raise ValueError("Expression has mismatched delimiters")
        index += 1
    if stack or quoted:
        raise ValueError("Expression has unfinished delimiters or a string")


def collect_actions(group, actions, variables):
    for name, action in group.items():
        if name in actions:
            raise ValueError(f"Duplicate workflow action: {name}")
        actions[name] = action
        for dependency in action.get("runAfter", {}):
            if dependency not in group or dependency == name:
                raise ValueError(f"Invalid sibling dependency: {name}")
        if action["type"] == "InitializeVariable":
            for variable in action["inputs"]["variables"]:
                if variable["name"] in variables:
                    raise ValueError("Variable has more than one initializer")
                variables.add(variable["name"])
        if action["type"] == "Http":
            secured = action.get("runtimeConfiguration", {}).get("secureData", {})
            if set(secured.get("properties", [])) != {"inputs", "outputs"}:
                raise ValueError("HTTP history must hide inputs and outputs")
            if action.get("operationOptions") != "DisableAsyncPattern":
                raise ValueError("HTTP actions must not poll Location headers")
        collect_actions(action.get("actions", {}), actions, variables)
        collect_actions(action.get("else", {}).get("actions", {}), actions, variables)
    visited = set()
    active = set()

    def visit(name):
        if name in active:
            raise ValueError("Action dependency cycle")
        if name in visited:
            return
        active.add(name)
        for dependency in group[name].get("runAfter", {}):
            visit(dependency)
        active.remove(name)
        visited.add(name)

    for name in group:
        visit(name)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--workflow", required=True, type=Path)
    parser.add_argument("--schema", required=True, type=Path)
    parser.add_argument("--schema-sha256", required=True)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    schema_bytes = args.schema.read_bytes()
    if hashlib.sha256(schema_bytes).hexdigest() != args.schema_sha256:
        raise ValueError("Schema checksum differs from the selected source")
    schema = json.loads(schema_bytes)
    workflow = json.loads(args.workflow.read_bytes())
    jsonschema.Draft4Validator.check_schema(schema)
    jsonschema.Draft4Validator(schema).validate(workflow)
    actions = {}
    variables = set()
    collect_actions(workflow["actions"], actions, variables)
    count = 0
    for expression in expressions(workflow):
        check_expression(expression)
        count += 1
        for kind, name in re.findall(
            r"\b(body|outputs|actions|items|variables)\('([^']+)'\)", expression
        ):
            if name not in (variables if kind == "variables" else actions):
                raise ValueError(f"Unknown {kind} reference: {name}")
            if kind == "items" and actions[name]["type"] != "Foreach":
                raise ValueError("Item reference must select a foreach action")
    result = {
        "schema": str(args.schema.resolve()),
        "schemaSHA256": args.schema_sha256,
        "workflowSHA256": hashlib.sha256(args.workflow.read_bytes()).hexdigest(),
        "actions": len(actions),
        "variables": len(variables),
        "expressions": count,
        "schemaValid": True,
        "staticReferencesValid": True,
        "executesWorkflow": False,
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(result, indent=2) + "\n")
    print(
        f"Workflow schema and static references pass: {len(actions)} actions, "
        f"{count} expressions"
    )


if __name__ == "__main__":
    main()
