#!/usr/bin/env python3
"""Generate docs/openapi/COVERAGE.md from discovery paths and current client usage."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DEFAULT_OUTPUT = ROOT / "docs" / "openapi" / "COVERAGE.md"
PATHS_FILE = ROOT / "docs" / "openapi" / "paths.txt"
GPC_DIR = ROOT / "internal" / "gpc"

ENTRY_RE = re.compile(r"^(?P<method>[A-Z]+) (?P<path>[^ ]+) \[(?P<id>[^\]]+)\]$")
CALL_RE = re.compile(r"c\.service\.([A-Za-z0-9_.]+)")


def parse_paths() -> list[dict[str, str]]:
    if not PATHS_FILE.exists():
        raise RuntimeError(f"{PATHS_FILE} is missing")

    entries: list[dict[str, str]] = []
    for line in PATHS_FILE.read_text(encoding="utf-8").splitlines():
        stripped = line.strip()
        if stripped == "" or stripped.startswith("#"):
            continue
        match = ENTRY_RE.match(stripped)
        if not match:
            raise RuntimeError(f"invalid paths entry: {stripped!r}")
        entries.append(match.groupdict())

    if not entries:
        raise RuntimeError("no discovery entries found in paths.txt")
    return entries


def extract_implemented_ids() -> set[str]:
    if not GPC_DIR.exists():
        raise RuntimeError(f"{GPC_DIR} is missing")

    ids: set[str] = set()
    for path in sorted(GPC_DIR.glob("*.go")):
        text = path.read_text(encoding="utf-8")
        for call in CALL_RE.findall(text):
            ids.add("androidpublisher." + ".".join(part.lower() for part in call.split(".")))
    if not ids:
        raise RuntimeError("no service calls found under internal/gpc")
    return ids


def family_for_method(method_id: str) -> str:
    parts = method_id.split(".")
    if len(parts) < 2:
        return method_id
    if parts[1] == "edits" and len(parts) >= 3:
        edit_subresources = {
            "apks",
            "bundles",
            "countryavailability",
            "deobfuscationfiles",
            "details",
            "expansionfiles",
            "images",
            "listings",
            "testers",
            "tracks",
        }
        if parts[2] in edit_subresources:
            return ".".join(parts[1:3])
        return "edits"
    if parts[1] in {"purchases", "monetization", "applications"} and len(parts) >= 3:
        return ".".join(parts[1:3])
    return parts[1]


def render_markdown(entries: list[dict[str, str]], implemented_ids: set[str]) -> str:
    known_ids = {entry["id"].lower() for entry in entries}
    unmatched_ids = sorted(implemented_ids - known_ids)
    matched_ids = implemented_ids & known_ids
    missing_entries = [entry for entry in entries if entry["id"].lower() not in implemented_ids]

    families = sorted({family_for_method(entry["id"]) for entry in entries})
    family_rows: list[tuple[str, int, int, int]] = []
    for family in families:
        family_entries = [entry for entry in entries if family_for_method(entry["id"]) == family]
        implemented = sum(1 for entry in family_entries if entry["id"].lower() in implemented_ids)
        total = len(family_entries)
        family_rows.append((family, implemented, total - implemented, total))

    lines: list[str] = []
    lines.append("# OpenAPI Coverage")
    lines.append("")
    lines.append("This file is generated from:")
    lines.append("")
    lines.append("- `docs/openapi/paths.txt`")
    lines.append("- live Google API service calls detected under `internal/gpc`")
    lines.append("")
    lines.append("To regenerate:")
    lines.append("")
    lines.append("```bash")
    lines.append("make generate-openapi-coverage")
    lines.append("```")
    lines.append("")
    lines.append("## Summary")
    lines.append("")
    lines.append("| Metric | Count |")
    lines.append("| --- | ---: |")
    lines.append(f"| Total discovery endpoints | {len(entries)} |")
    lines.append(f"| Implemented endpoints | {len(matched_ids)} |")
    lines.append(f"| Missing endpoints | {len(missing_entries)} |")
    lines.append(f"| Detected service method IDs | {len(implemented_ids)} |")
    lines.append(f"| Unmatched service method IDs | {len(unmatched_ids)} |")
    lines.append("")
    lines.append("## Family Summary")
    lines.append("")
    lines.append("| Family | Implemented | Missing | Total |")
    lines.append("| --- | ---: | ---: | ---: |")
    for family, implemented, missing, total in family_rows:
        lines.append(f"| `{family}` | {implemented} | {missing} | {total} |")

    lines.append("")
    lines.append("## Missing Endpoints")
    lines.append("")
    if not missing_entries:
        lines.append("No missing discovery endpoints detected.")
    else:
        missing_by_family: dict[str, list[dict[str, str]]] = {}
        for entry in missing_entries:
            missing_by_family.setdefault(family_for_method(entry["id"]), []).append(entry)
        for family in sorted(missing_by_family):
            lines.append(f"### `{family}`")
            lines.append("")
            for entry in missing_by_family[family]:
                lines.append(f"- `{entry['id']}` | `{entry['method']}` `{entry['path']}`")
            lines.append("")
        if lines[-1] == "":
            lines.pop()

    lines.append("")
    lines.append("## Unmatched Service Method IDs")
    lines.append("")
    if not unmatched_ids:
        lines.append("No unmatched service method IDs detected.")
    else:
        for method_id in unmatched_ids:
            lines.append(f"- `{method_id}`")
        raise RuntimeError(
            "detected service method IDs that do not exist in docs/openapi/paths.txt; "
            "refresh the discovery index or fix the coverage mapper"
        )

    return "\n".join(lines).rstrip() + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--output",
        default=str(DEFAULT_OUTPUT),
        help="Path to write generated markdown (default: docs/openapi/COVERAGE.md)",
    )
    args = parser.parse_args()

    output_path = Path(args.output).resolve()
    entries = parse_paths()
    implemented_ids = extract_implemented_ids()

    try:
        markdown = render_markdown(entries, implemented_ids)
    except RuntimeError as exc:
        sys.stderr.write(f"{exc}\n")
        return 1

    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(markdown, encoding="utf-8")
    print(f"wrote {output_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
