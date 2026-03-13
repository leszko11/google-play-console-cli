#!/usr/bin/env python3
"""Validate docs/COMMANDS.md and README command discovery stay in sync with live help output."""

from __future__ import annotations

import difflib
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DOC_PATH = ROOT / "docs" / "COMMANDS.md"
GEN_SCRIPT = ROOT / "scripts" / "generate-command-docs.py"
README_PATH = ROOT / "README.md"


def run_help(path_parts: list[str]) -> str:
    cmd = ["go", "run", ".", *path_parts, "--help"]
    env = os.environ.copy()
    env.pop("GPC_BIN", None)
    proc = subprocess.run(
        cmd,
        cwd=ROOT,
        env=env,
        text=True,
        capture_output=True,
        check=False,
    )
    if proc.returncode != 0:
        joined = " ".join(cmd)
        raise RuntimeError(
            f"help command failed ({proc.returncode}): {joined}\n{proc.stderr.strip()}"
        )
    return (proc.stdout + proc.stderr).strip()


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
        if name != "help" and name not in subcommands:
            subcommands.append(name)

    return subcommands


def read_readme_command_discovery() -> set[str]:
    if not README_PATH.exists():
        raise RuntimeError(f"{README_PATH} is missing")

    lines = README_PATH.read_text(encoding="utf-8").splitlines()
    section_idx = -1
    for idx, line in enumerate(lines):
        if line.strip() == "## Command Discovery":
            section_idx = idx
            break
    if section_idx == -1:
        raise RuntimeError("README is missing `## Command Discovery` section")

    block_start = -1
    for idx in range(section_idx + 1, len(lines)):
        if lines[idx].strip().startswith("```"):
            block_start = idx + 1
            break
    if block_start == -1:
        raise RuntimeError("README Command Discovery section is missing a fenced code block")

    block_end = -1
    for idx in range(block_start, len(lines)):
        if lines[idx].strip().startswith("```"):
            block_end = idx
            break
    if block_end == -1:
        raise RuntimeError("README Command Discovery fenced code block is not closed")

    pattern = re.compile(r"^gpc(?:\s+([a-z0-9][a-z0-9-]*))?\s+--help$")
    found_root = False
    discovered: set[str] = set()
    for idx in range(block_start, block_end):
        raw = lines[idx].strip()
        if raw == "" or raw.startswith("#"):
            continue
        match = pattern.match(raw)
        if not match:
            raise RuntimeError(
                f"README Command Discovery contains invalid line at README:{idx + 1}: {raw!r}"
            )
        command = match.group(1)
        if command is None:
            found_root = True
        else:
            discovered.add(command)

    if not found_root:
        raise RuntimeError("README Command Discovery must include `gpc --help`")

    return discovered


def validate_readme_discovery() -> int:
    root_help = run_help([])
    expected = set(parse_subcommands(root_help))
    actual = read_readme_command_discovery()

    missing = sorted(expected - actual)
    extra = sorted(actual - expected)
    if not missing and not extra:
        print("README command discovery is in sync")
        return 0

    if missing:
        sys.stderr.write(
            "README command discovery is missing top-level commands: "
            + ", ".join(missing)
            + "\n"
        )
    if extra:
        sys.stderr.write(
            "README command discovery contains unknown commands: "
            + ", ".join(extra)
            + "\n"
        )
    sys.stderr.write(
        "Update the `## Command Discovery` block in README.md to match `gpc --help`.\n"
    )
    return 1


def main() -> int:
    failures = 0
    with tempfile.TemporaryDirectory() as tmpdir:
        generated_path = Path(tmpdir) / "COMMANDS.generated.md"
        cmd = [sys.executable, str(GEN_SCRIPT), "--output", str(generated_path)]
        env = os.environ.copy()
        # Keep the docs check deterministic by always generating from source tree help output.
        env.pop("GPC_BIN", None)
        proc = subprocess.run(
            cmd,
            cwd=ROOT,
            env=env,
            text=True,
            capture_output=True,
            check=False,
        )
        if proc.returncode != 0:
            sys.stderr.write(proc.stdout)
            sys.stderr.write(proc.stderr)
            return proc.returncode

        generated = generated_path.read_text(encoding="utf-8")

    if not DOC_PATH.exists():
        sys.stderr.write(
            f"{DOC_PATH} is missing. Run `make generate-command-docs`.\n"
        )
        failures += 1

    if DOC_PATH.exists():
        existing = DOC_PATH.read_text(encoding="utf-8")
        if existing == generated:
            print("docs/COMMANDS.md is in sync")
        else:
            diff = difflib.unified_diff(
                existing.splitlines(),
                generated.splitlines(),
                fromfile=str(DOC_PATH),
                tofile="generated",
                lineterm="",
            )
            sys.stderr.write("\n".join(diff) + "\n")
            sys.stderr.write("docs/COMMANDS.md is out of sync. Run `make generate-command-docs`.\n")
            failures += 1

    try:
        failures += validate_readme_discovery()
    except RuntimeError as exc:
        sys.stderr.write(f"{exc}\n")
        failures += 1

    return 0 if failures == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
