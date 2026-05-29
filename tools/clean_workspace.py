#!/usr/bin/env python3
"""Remove local logs, caches, and generated analysis artifacts."""

from __future__ import annotations

import argparse
import shutil
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]

TARGETS = (
    ".tmp_pdf_pages",
    "_req_pdf_pages",
    ".runtime/logs",
    ".runtime/codex_pdf_pages",
    ".runtime/pdf_pages",
    ".runtime/taoding_pages",
    ".runtime/backend-dev.pid",
    ".runtime/frontend-dev.pid",
    ".runtime/backend.err.log",
    ".runtime/backend.log",
    ".runtime/backend.out.log",
    ".runtime/frontend.err.log",
    ".runtime/frontend.out.log",
    ".runtime/service-unit-test.jsonl",
    ".runtime/data/logs",
    "frontend/coverage",
    "frontend/tsconfig.tsbuildinfo",
    "frontend/tsconfig.node.tsbuildinfo",
    "frontend/vite.config.js",
    "frontend/vite.config.d.ts",
    "tools/__pycache__",
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Clean local Sub2API runtime and analysis artifacts."
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print matching paths without removing them.",
    )
    return parser.parse_args()


def assert_inside_repo(path: Path) -> Path:
    resolved = path.resolve()
    try:
        resolved.relative_to(ROOT)
    except ValueError as exc:
        raise SystemExit(f"Refusing to remove path outside repo: {resolved}") from exc
    return resolved


def path_size(path: Path) -> int:
    if path.is_file() or path.is_symlink():
        return path.stat().st_size
    total = 0
    for child in path.rglob("*"):
        if child.is_file() or child.is_symlink():
            total += child.stat().st_size
    return total


def remove_path(path: Path) -> None:
    if path.is_dir() and not path.is_symlink():
        shutil.rmtree(path)
    else:
        path.unlink()


def main() -> int:
    args = parse_args()
    removed: list[tuple[str, int]] = []

    for target in TARGETS:
        path = ROOT / target
        if not path.exists():
            continue
        resolved = assert_inside_repo(path)
        size = path_size(resolved)
        removed.append((target, size))
        if not args.dry_run:
            remove_path(resolved)

    if not removed:
        print("No local cleanup targets found.")
        return 0

    verb = "Would remove" if args.dry_run else "Removed"
    for target, size in removed:
        print(f"{verb}: {target} ({size / 1024 / 1024:.2f} MB)")

    total = sum(size for _, size in removed)
    print(f"Total: {total / 1024 / 1024:.2f} MB")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
