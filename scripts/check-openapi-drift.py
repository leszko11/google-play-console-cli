#!/usr/bin/env python3
"""Check live Android Publisher discovery methods against committed paths and deferred IDs."""

from __future__ import annotations

import argparse
import json
import re
import sys
import urllib.request
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DEFAULT_URL = "https://androidpublisher.googleapis.com/$discovery/rest?version=v3"
PATHS_FILE = ROOT / "docs" / "openapi" / "paths.txt"
DEFERRED_FILE = ROOT / "docs" / "openapi" / "deferred_method_ids.txt"
ENTRY_RE = re.compile(r"^(?P<method>[A-Z]+) (?P<path>[^ ]+) \[(?P<id>[^\]]+)\]$")


def load_discovery(source: str | None, fetch: bool) -> dict:
    if source:
        path = Path(source).expanduser().resolve()
        if not path.exists():
            raise FileNotFoundError(f"discovery file not found: {path}")
        return json.loads(path.read_text(encoding="utf-8"))

    if not fetch:
        raise FileNotFoundError(
            "no discovery source provided. Pass --source <file> for offline checks "
            "or pass --fetch to download the current discovery document."
        )

    with urllib.request.urlopen(DEFAULT_URL) as response:
        payload = response.read().decode("utf-8")
    return json.loads(payload)


def collect_method_ids(node: dict, out: set[str]) -> None:
    methods = node.get("methods", {})
    for method in methods.values():
        method_id = str(method.get("id", "")).strip().lower()
        if method_id:
            out.add(method_id)

    for resource in node.get("resources", {}).values():
        collect_method_ids(resource, out)


def parse_committed_ids(path: Path) -> set[str]:
    if not path.exists():
        raise FileNotFoundError(f"{path} is missing")

    ids: set[str] = set()
    for line in path.read_text(encoding="utf-8").splitlines():
        stripped = line.strip()
        if stripped == "" or stripped.startswith("#"):
            continue
        match = ENTRY_RE.match(stripped)
        if not match:
            raise RuntimeError(f"invalid paths entry: {stripped!r}")
        ids.add(match.group("id").strip().lower())
    if not ids:
        raise RuntimeError("no committed discovery IDs found in paths.txt")
    return ids


def parse_deferred_ids(path: Path) -> set[str]:
    if not path.exists():
        raise FileNotFoundError(f"{path} is missing")

    ids: set[str] = set()
    for line in path.read_text(encoding="utf-8").splitlines():
        stripped = line.strip()
        if stripped == "" or stripped.startswith("#"):
            continue
        method_id = stripped.split("#", 1)[0].strip().lower()
        if method_id == "":
            continue
        ids.add(method_id)
    return ids


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", default="", help="Path to discovery JSON file (offline mode).")
    parser.add_argument("--fetch", action="store_true", help="Fetch discovery JSON from Google.")
    parser.add_argument(
        "--paths",
        default=str(PATHS_FILE),
        help="Committed paths index to compare against (default: docs/openapi/paths.txt)",
    )
    parser.add_argument(
        "--deferred",
        default=str(DEFERRED_FILE),
        help="Deferred method ID registry (default: docs/openapi/deferred_method_ids.txt)",
    )
    args = parser.parse_args()

    discovery = load_discovery(args.source or None, args.fetch or not args.source)
    live_ids: set[str] = set()
    collect_method_ids(discovery, live_ids)
    if not live_ids:
        sys.stderr.write("no discovery method IDs found in live document\n")
        return 1

    committed_ids = parse_committed_ids(Path(args.paths).expanduser().resolve())
    deferred_ids = parse_deferred_ids(Path(args.deferred).expanduser().resolve())

    new_ids = sorted(live_ids - committed_ids - deferred_ids)
    removed_ids = sorted(committed_ids - live_ids)
    stale_deferred = sorted(deferred_ids - live_ids)

    if not new_ids and not removed_ids and not stale_deferred:
        print("live discovery drift check passed")
        return 0

    if new_ids:
        sys.stderr.write(
            "live discovery contains new method IDs not tracked in docs/openapi/paths.txt "
            "or docs/openapi/deferred_method_ids.txt:\n"
        )
        for method_id in new_ids:
            sys.stderr.write(f"- {method_id}\n")
    if removed_ids:
        sys.stderr.write("committed paths.txt contains method IDs no longer present in live discovery:\n")
        for method_id in removed_ids:
            sys.stderr.write(f"- {method_id}\n")
    if stale_deferred:
        sys.stderr.write("deferred method IDs no longer present in live discovery:\n")
        for method_id in stale_deferred:
            sys.stderr.write(f"- {method_id}\n")

    return 1


if __name__ == "__main__":
    raise SystemExit(main())
