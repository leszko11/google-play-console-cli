#!/usr/bin/env python3
"""Generate docs/COMMANDS.md from live `gpc --help` output."""

from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Iterable


ROOT = Path(__file__).resolve().parents[1]
DEFAULT_OUTPUT = ROOT / "docs" / "COMMANDS.md"


def base_command() -> list[str]:
    gpc_bin = os.environ.get("GPC_BIN", "").strip()
    if gpc_bin:
        return [gpc_bin]
    return ["go", "run", "."]


def run_help(path_parts: list[str]) -> str:
    cmd = [*base_command(), *path_parts, "--help"]
    proc = subprocess.run(
        cmd,
        cwd=ROOT,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        joined = " ".join(cmd)
        raise RuntimeError(
            f"help command failed ({proc.returncode}): {joined}\n{proc.stderr.strip()}"
        )
    out = (proc.stdout + proc.stderr).strip()
    return out


def parse_subcommands(help_text: str) -> list[str]:
    lines = help_text.splitlines()
    subcommands: list[str] = []
    in_subcommands = False
    section_header = re.compile(r"^[A-Z][A-Z0-9 _-]*$")
    command_line = re.compile(r"^\s*([a-z0-9][a-z0-9-]*)\b")

    for line in lines:
        stripped = line.strip()
        lowered = stripped.lower()
        if not in_subcommands:
            if lowered.startswith("subcommands"):
                in_subcommands = True
            continue

        if section_header.match(stripped) and lowered != "subcommands":
            break
        if stripped == "":
            continue

        match = command_line.match(line)
        if not match:
            continue

        name = match.group(1)
        if name == "help":
            continue
        if name not in subcommands:
            subcommands.append(name)

    return subcommands


def walk_commands(path_parts: list[str], seen: set[tuple[str, ...]]) -> dict:
    key = tuple(path_parts)
    if key in seen:
        return {"path": path_parts, "help": "", "children": []}
    seen.add(key)

    help_text = run_help(path_parts)
    children = [walk_commands(path_parts + [name], seen) for name in parse_subcommands(help_text)]
    return {"path": path_parts, "help": help_text, "children": children}


def iter_nodes(node: dict) -> Iterable[dict]:
    yield node
    for child in node.get("children", []):
        yield from iter_nodes(child)


def render_markdown(root_node: dict) -> str:
    lines: list[str] = []
    lines.append("# Command Reference Guide")
    lines.append("")
    lines.append("This file is generated from live CLI help output.")
    lines.append("For authoritative command behavior, use:")
    lines.append("")
    lines.append("```bash")
    lines.append("gpc --help")
    lines.append("gpc <command> --help")
    lines.append("gpc <command> <subcommand> --help")
    lines.append("```")
    lines.append("")
    lines.append("To regenerate:")
    lines.append("")
    lines.append("```bash")
    lines.append("make generate-command-docs")
    lines.append("```")
    lines.append("")
    lines.append("## Command Paths")
    lines.append("")

    command_paths: list[str] = []
    for node in iter_nodes(root_node):
        if node["path"]:
            command_paths.append(" ".join(["gpc", *node["path"]]))
    for path in command_paths:
        lines.append(f"- `{path}`")

    for node in iter_nodes(root_node):
        path = " ".join(["gpc", *node["path"]]).strip()
        lines.append("")
        lines.append(f"## `{path} --help`")
        lines.append("")
        lines.append("```text")
        help_text = node.get("help", "").strip("\n")
        lines.append(help_text)
        lines.append("```")

    return "\n".join(lines).rstrip() + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--output",
        default=str(DEFAULT_OUTPUT),
        help="Path to write generated markdown (default: docs/COMMANDS.md)",
    )
    args = parser.parse_args()

    output_path = Path(args.output).resolve()
    tree = walk_commands([], set())
    markdown = render_markdown(tree)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(markdown, encoding="utf-8")
    print(f"wrote {output_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

