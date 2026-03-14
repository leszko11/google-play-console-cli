#!/usr/bin/env python3
"""Generate llms.txt from live CLI help plus stable repo guidance."""

from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DEFAULT_OUTPUT = ROOT / "llms.txt"


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
    return (proc.stdout + proc.stderr).strip()


def collect_section_lines(help_text: str, header: str) -> list[str]:
    lines = help_text.splitlines()
    section_lines: list[str] = []
    in_section = False
    section_header = re.compile(r"^[A-Z][A-Z0-9 _-]*$")

    for line in lines:
        stripped = line.strip()
        lowered = stripped.lower()
        if not in_section:
            if lowered.startswith(header.lower()):
                in_section = True
            continue

        if section_header.match(stripped) and lowered != header.lower():
            break
        if stripped == "":
            continue

        section_lines.append(line)

    return section_lines


def parse_command_items(help_text: str, header: str) -> list[tuple[str, str]]:
    items: list[tuple[str, str]] = []
    item_line = re.compile(r"^\s*([a-z0-9][a-z0-9-]*)\s{2,}(.*)$")

    for line in collect_section_lines(help_text, header):
        match = item_line.match(line)
        if not match:
            continue
        name = match.group(1)
        if name == "help":
            continue
        items.append((name, match.group(2).strip()))

    return items


def parse_flag_items(help_text: str) -> list[tuple[str, str]]:
    items: list[tuple[str, str]] = []
    item_line = re.compile(r"^\s*-([a-z0-9][a-z0-9-]*)(?:[ =][^ ]+)?\s{2,}(.*)$")

    for line in collect_section_lines(help_text, "FLAGS"):
        match = item_line.match(line)
        if not match:
            continue
        items.append((match.group(1), match.group(2).strip()))

    return items


def render_llms(root_help: str) -> str:
    commands = parse_command_items(root_help, "SUBCOMMANDS")
    flags = parse_flag_items(root_help)

    lines: list[str] = []
    lines.append("# gpc")
    lines.append("")
    lines.append("> Scriptable Google Play Console CLI for publishing, reporting, release, and monetization workflows.")
    lines.append("")
    lines.append("## Start Here")
    lines.append("")
    lines.append("- Discover commands with `gpc --help` and `gpc <command> --help`.")
    lines.append("- Run the full local verification gate with `make dev`.")
    lines.append("- Build the CLI locally with `make build`.")
    lines.append("")
    lines.append("## Automation-Safe Usage")
    lines.append("")
    lines.append("- Non-interactive sessions default to `json`; interactive terminals default to `table`.")
    lines.append("- Keep stdout for data and stderr for help, warnings, and errors.")
    lines.append("- Prefer explicit `--output json` or `GPC_DEFAULT_OUTPUT=json` in CI.")
    lines.append("- Prefer explicit `--package-name`, `--profile`, `--service-account`, and `--strict-auth` in scripts.")
    lines.append("")
    lines.append("## Global Flags")
    lines.append("")
    for name, description in flags:
        lines.append(f"- `--{name}`: {description}")
    lines.append("")
    lines.append("## Top-Level Commands")
    lines.append("")
    for name, description in commands:
        lines.append(f"- `{name}`: {description}")
    lines.append("")
    lines.append("## Common Flows")
    lines.append("")
    lines.append("- Authenticate: `gpc auth init --service-account /path/to/service-account.json`")
    lines.append("- Inspect configured profiles: `gpc auth status` and `gpc auth profiles list --output table`")
    lines.append("- Bootstrap a local store workspace: `gpc bootstrap --package-name com.example.app --dir ./store --write-project-config`")
    lines.append("- Validate a release before upload: `gpc release verify --package-name com.example.app --aab /path/to/app.aab`")
    lines.append("- Wait for bundle processing in CI: `gpc bundles wait --package-name com.example.app --version-code 123 --output json`")
    lines.append("")
    lines.append("## Repo Docs")
    lines.append("")
    lines.append("- [README.md](README.md): install, quickstart, command discovery, and examples")
    lines.append("- [docs/COMMANDS.md](docs/COMMANDS.md): generated command reference from live help output")
    lines.append("- [docs/AUTH.md](docs/AUTH.md): auth flows, config precedence, and credential storage")
    lines.append("- [docs/API_NOTES.md](docs/API_NOTES.md): Google Play API caveats and behavior notes")
    lines.append("- [docs/TESTING.md](docs/TESTING.md): smoke tests and local verification")
    lines.append("- [docs/RELEASING.md](docs/RELEASING.md): release process for maintainers")
    lines.append("- [docs/openapi/COVERAGE.md](docs/openapi/COVERAGE.md): endpoint coverage status")
    lines.append("- [AGENTS.md](AGENTS.md): repository-specific AI agent guidance")
    return "\n".join(lines).rstrip() + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--output",
        default=str(DEFAULT_OUTPUT),
        help="Path to write generated text (default: llms.txt)",
    )
    args = parser.parse_args()

    output_path = Path(args.output).resolve()
    llms_text = render_llms(run_help([]))
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(llms_text, encoding="utf-8")
    print(f"wrote {output_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
