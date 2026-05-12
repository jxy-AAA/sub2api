#!/usr/bin/env python3
"""High-confidence secret scanner for repository quality gates.

The scanner intentionally defaults to git-tracked files only. That keeps local
runtime configuration and ignored build outputs from creating noisy failures,
while still catching accidental secret commits.
"""

from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable


ALLOWLIST_MARKERS = (
    "pragma: allowlist secret",
    "nosec",
    "gitleaks:allow",
)

PLACEHOLDER_HINTS = (
    "example",
    "placeholder",
    "dummy",
    "fake",
    "test",
    "sample",
    "changeme",
    "change-this",
    "configured-",
    "your_",
    "your-",
    "<",
    "${",
    "...",
    "\\nabc\\n",
    "\\ndata\\n",
)

SKIP_DIRS = {
    ".git",
    ".github",
    ".runtime",
    ".serena",
    "node_modules",
    "dist",
    "build",
    "release",
    "coverage",
    "vendor",
}

SKIP_SUFFIXES = {
    ".jpg",
    ".jpeg",
    ".png",
    ".gif",
    ".webp",
    ".ico",
    ".pdf",
    ".zip",
    ".gz",
    ".tgz",
    ".xz",
    ".7z",
    ".exe",
    ".dll",
    ".so",
    ".dylib",
}


@dataclass(frozen=True)
class Rule:
    name: str
    pattern: re.Pattern[str]


RULES = (
    Rule(
        "private-key",
        re.compile(r"-----BEGIN (?:RSA |DSA |EC |OPENSSH |PGP )?PRIVATE KEY-----"),
    ),
    Rule("aws-access-key", re.compile(r"\b(?:AKIA|ASIA)[0-9A-Z]{16}\b")),
    Rule("github-token", re.compile(r"\bgh[pousr]_[A-Za-z0-9_]{36,}\b")),
    Rule("slack-token", re.compile(r"\bxox[baprs]-[A-Za-z0-9-]{20,}\b")),
    Rule("stripe-live-secret", re.compile(r"\bsk_live_[A-Za-z0-9]{20,}\b")),
    Rule("openai-key", re.compile(r"\bsk-(?:proj-)?[A-Za-z0-9_-]{32,}\b")),
    Rule(
        "generic-assigned-secret",
        re.compile(
            r"(?i)\b(?:secret|token|api[_-]?key|password|passwd|pwd)\b"
            r"\s*[:=]\s*['\"]([A-Za-z0-9_./+=-]{32,})['\"]"
        ),
    ),
)


@dataclass(frozen=True)
class Finding:
    path: Path
    line_number: int
    rule: str
    line: str


def repo_root(start: Path) -> Path:
    try:
        result = subprocess.run(
            ["git", "rev-parse", "--show-toplevel"],
            cwd=start,
            check=True,
            capture_output=True,
            text=True,
        )
    except (OSError, subprocess.CalledProcessError):
        return start.resolve()
    return Path(result.stdout.strip()).resolve()


def git_tracked_files(root: Path) -> list[Path] | None:
    try:
        result = subprocess.run(
            ["git", "ls-files", "-z"],
            cwd=root,
            check=True,
            capture_output=True,
        )
    except (OSError, subprocess.CalledProcessError):
        return None

    paths = []
    for raw in result.stdout.split(b"\0"):
        if raw:
            paths.append(root / raw.decode("utf-8", errors="surrogateescape"))
    return paths


def walked_files(root: Path) -> Iterable[Path]:
    for current_root, dir_names, file_names in os.walk(root):
        dir_names[:] = [name for name in dir_names if name not in SKIP_DIRS]
        for file_name in file_names:
            yield Path(current_root) / file_name


def should_scan(path: Path, root: Path) -> bool:
    try:
        relative = path.resolve().relative_to(root)
    except ValueError:
        return False
    if any(part in SKIP_DIRS for part in relative.parts[:-1]):
        return False
    if path.suffix.lower() in SKIP_SUFFIXES:
        return False
    return path.is_file()


def is_binary(path: Path) -> bool:
    try:
        with path.open("rb") as handle:
            sample = handle.read(2048)
    except OSError:
        return True
    return b"\0" in sample


def is_placeholder(line: str) -> bool:
    lower = line.lower()
    return any(hint in lower for hint in PLACEHOLDER_HINTS)


def is_allowlisted(line: str) -> bool:
    lower = line.lower()
    return any(marker in lower for marker in ALLOWLIST_MARKERS)


def scan_file(path: Path) -> Iterable[Finding]:
    if is_binary(path):
        return
    try:
        lines = path.read_text(encoding="utf-8", errors="replace").splitlines()
    except OSError:
        return

    for line_number, line in enumerate(lines, start=1):
        if is_allowlisted(line) or is_placeholder(line):
            continue
        for rule in RULES:
            if rule.pattern.search(line):
                yield Finding(path=path, line_number=line_number, rule=rule.name, line=line)


def redact(line: str) -> str:
    stripped = line.strip()
    if len(stripped) <= 120:
        return stripped
    return f"{stripped[:80]}...{stripped[-20:]}"


def collect_files(root: Path, all_files: bool) -> list[Path]:
    if all_files:
        candidates = list(walked_files(root))
    else:
        candidates = git_tracked_files(root) or list(walked_files(root))
    return [path for path in candidates if should_scan(path, root)]


def main() -> int:
    parser = argparse.ArgumentParser(description="Scan repository files for committed secrets.")
    parser.add_argument("--path", default=".", help="Repository path to scan.")
    parser.add_argument(
        "--all-files",
        action="store_true",
        help="Scan all non-ignored-looking files instead of git-tracked files only.",
    )
    args = parser.parse_args()

    root = repo_root(Path(args.path).resolve())
    findings: list[Finding] = []
    for path in collect_files(root, args.all_files):
        findings.extend(scan_file(path))

    if not findings:
        print("Secret scan passed: no high-confidence secrets found.")
        return 0

    sys.stderr.write("Potential committed secrets found:\n")
    for finding in findings:
        relative = finding.path.resolve().relative_to(root)
        sys.stderr.write(
            f"- {relative}:{finding.line_number} [{finding.rule}] {redact(finding.line)}\n"
        )
    sys.stderr.write(
        "Add a documented allowlist marker only after confirming the value is not secret.\n"
    )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
