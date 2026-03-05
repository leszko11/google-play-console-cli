#!/usr/bin/env python3
"""Generate docs/openapi/paths.txt from an Android Publisher discovery document."""

from __future__ import annotations

import argparse
import json
import urllib.request
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DEFAULT_OUTPUT = ROOT / "docs" / "openapi" / "paths.txt"
DEFAULT_URL = "https://androidpublisher.googleapis.com/$discovery/rest?version=v3"


def load_discovery(source: str | None, fetch: bool) -> dict:
    if source:
        path = Path(source).expanduser().resolve()
        if not path.exists():
            raise FileNotFoundError(f"discovery file not found: {path}")
        return json.loads(path.read_text(encoding="utf-8"))

    if not fetch:
        raise FileNotFoundError(
            "no discovery source provided. Pass --source <file> for offline refresh "
            "or pass --fetch to download the current discovery document."
        )

    with urllib.request.urlopen(DEFAULT_URL) as response:
        payload = response.read().decode("utf-8")
    return json.loads(payload)


def collect_methods(node: dict, out: set[str]) -> None:
    methods = node.get("methods", {})
    for method in methods.values():
        http_method = str(method.get("httpMethod", "")).strip().upper()
        path = str(method.get("flatPath") or method.get("path") or "").strip()
        method_id = str(method.get("id", "")).strip()
        if http_method == "" or path == "":
            continue
        if method_id:
            out.add(f"{http_method} {path} [{method_id}]")
        else:
            out.add(f"{http_method} {path}")

    for resource in node.get("resources", {}).values():
        collect_methods(resource, out)


def render_lines(discovery: dict) -> list[str]:
    methods: set[str] = set()
    collect_methods(discovery, methods)
    if not methods:
        raise RuntimeError("no methods found in discovery document")

    lines = [
        "# Android Publisher API v3 endpoint index",
        "#",
        "# Generated from discovery document.",
        "# Format: <HTTP_METHOD> <PATH> [<DISCOVERY_METHOD_ID>]",
        "",
    ]
    lines.extend(sorted(methods))
    lines.append("")
    return lines


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--source",
        default="",
        help="Path to discovery JSON file (offline mode).",
    )
    parser.add_argument(
        "--fetch",
        action="store_true",
        help="Fetch discovery JSON from Google and generate paths.txt.",
    )
    parser.add_argument(
        "--output",
        default=str(DEFAULT_OUTPUT),
        help="Output path (default: docs/openapi/paths.txt)",
    )
    args = parser.parse_args()

    discovery = load_discovery(args.source or None, args.fetch)
    lines = render_lines(discovery)

    output_path = Path(args.output).expanduser().resolve()
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text("\n".join(lines), encoding="utf-8")
    print(f"wrote {output_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
