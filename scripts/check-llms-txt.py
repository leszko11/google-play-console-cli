#!/usr/bin/env python3
"""Validate llms.txt stays in sync with live CLI help output."""

from __future__ import annotations

import difflib
import subprocess
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
LLMS_PATH = ROOT / "llms.txt"
GEN_SCRIPT = ROOT / "scripts" / "generate-llms-txt.py"


def main() -> int:
    with tempfile.TemporaryDirectory() as tmpdir:
        generated_path = Path(tmpdir) / "llms.generated.txt"
        proc = subprocess.run(
            [sys.executable, str(GEN_SCRIPT), "--output", str(generated_path)],
            cwd=ROOT,
            text=True,
            capture_output=True,
            check=False,
        )
        if proc.returncode != 0:
            sys.stderr.write(proc.stdout)
            sys.stderr.write(proc.stderr)
            return proc.returncode

        generated = generated_path.read_text(encoding="utf-8")

    if not LLMS_PATH.exists():
        sys.stderr.write(f"{LLMS_PATH} is missing. Run `make generate-llms-txt`.\n")
        return 1

    existing = LLMS_PATH.read_text(encoding="utf-8")
    if existing == generated:
        print("llms.txt is in sync")
        return 0

    diff = difflib.unified_diff(
        existing.splitlines(),
        generated.splitlines(),
        fromfile=str(LLMS_PATH),
        tofile="generated",
        lineterm="",
    )
    sys.stderr.write("\n".join(diff) + "\n")
    sys.stderr.write("llms.txt is out of sync. Run `make generate-llms-txt`.\n")
    return 1


if __name__ == "__main__":
    sys.exit(main())
