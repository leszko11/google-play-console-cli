#!/usr/bin/env python3
"""Validate docs/openapi/COVERAGE.md stays in sync with generated coverage output."""

from __future__ import annotations

import difflib
import subprocess
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DOC_PATH = ROOT / "docs" / "openapi" / "COVERAGE.md"
GEN_SCRIPT = ROOT / "scripts" / "generate-openapi-coverage.py"


def main() -> int:
    with tempfile.NamedTemporaryFile(suffix=".md") as tmp:
        proc = subprocess.run(
            [sys.executable, str(GEN_SCRIPT), "--output", tmp.name],
            cwd=ROOT,
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
        sys.stderr.write(f"{DOC_PATH} is missing. Run `make generate-openapi-coverage`.\n")
        return 1

    existing = DOC_PATH.read_text(encoding="utf-8")
    if existing == generated:
        print("docs/openapi/COVERAGE.md is in sync")
        return 0

    diff = difflib.unified_diff(
        existing.splitlines(),
        generated.splitlines(),
        fromfile=str(DOC_PATH),
        tofile="generated",
        lineterm="",
    )
    sys.stderr.write("\n".join(diff) + "\n")
    sys.stderr.write("docs/openapi/COVERAGE.md is out of sync. Run `make generate-openapi-coverage`.\n")
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
