#!/usr/bin/env python3
"""Validate docs/COMMANDS.md stays in sync with live help output."""

from __future__ import annotations

import difflib
import os
import subprocess
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DOC_PATH = ROOT / "docs" / "COMMANDS.md"
GEN_SCRIPT = ROOT / "scripts" / "generate-command-docs.py"


def main() -> int:
    with tempfile.NamedTemporaryFile(suffix=".md") as tmp:
        cmd = [sys.executable, str(GEN_SCRIPT), "--output", tmp.name]
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

        generated = Path(tmp.name).read_text(encoding="utf-8")

    if not DOC_PATH.exists():
        sys.stderr.write(
            f"{DOC_PATH} is missing. Run `make generate-command-docs`.\n"
        )
        return 1

    existing = DOC_PATH.read_text(encoding="utf-8")
    if existing == generated:
        print("docs/COMMANDS.md is in sync")
        return 0

    diff = difflib.unified_diff(
        existing.splitlines(),
        generated.splitlines(),
        fromfile=str(DOC_PATH),
        tofile="generated",
        lineterm="",
    )
    sys.stderr.write("\n".join(diff) + "\n")
    sys.stderr.write("docs/COMMANDS.md is out of sync. Run `make generate-command-docs`.\n")
    return 1


if __name__ == "__main__":
    sys.exit(main())
