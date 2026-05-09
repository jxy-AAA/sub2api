#!/usr/bin/env python3
from __future__ import annotations

import argparse
import os
import sys
from dataclasses import dataclass
from typing import Iterable


TARGET_TABLES = (
    "affiliate_distribution_usage_jobs",
    "affiliate_distribution_usage_settlements",
    "affiliate_distribution_daily_metrics",
    "affiliate_distribution_rebate_balances",
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Safely preview or clear affiliate demo settlement tables."
    )
    parser.add_argument(
        "--dsn",
        default=os.environ.get("DATABASE_URL", ""),
        help="PostgreSQL DSN. Defaults to DATABASE_URL.",
    )
    parser.add_argument(
        "--schema",
        default=os.environ.get("SUB2API_DB_SCHEMA", "public"),
        help="Target PostgreSQL schema. Defaults to public.",
    )
    parser.add_argument(
        "--execute",
        action="store_true",
        help="Actually run DELETE statements. Without this flag the tool is dry-run only.",
    )
    parser.add_argument(
        "--yes",
        action="store_true",
        help="Skip interactive confirmation. Only valid with --execute.",
    )
    parser.add_argument(
        "--allow-non-demo-dsn",
        action="store_true",
        help="Bypass the demo/local DSN safety check.",
    )
    return parser.parse_args()


def require_psycopg():
    try:
        import psycopg  # type: ignore
    except ImportError as exc:
        raise SystemExit(
            "Missing dependency: psycopg. Run this script from conda env `bot`, "
            "or install `psycopg[binary]` into that environment."
        ) from exc
    return psycopg


def validate_dsn(dsn: str, allow_non_demo: bool) -> None:
    if not dsn.strip():
        raise SystemExit("DATABASE_URL/--dsn is required.")
    lowered = dsn.lower()
    safe_markers = ("localhost", "127.0.0.1", "sub2api.local", "demo", "test")
    if allow_non_demo:
        return
    if not any(marker in lowered for marker in safe_markers):
        raise SystemExit(
            "Refusing to run against a non-demo DSN. Use a local/demo database, "
            "or pass --allow-non-demo-dsn after manual verification."
        )


def quote_ident(name: str) -> str:
    return '"' + name.replace('"', '""') + '"'


def qualified_table(schema: str, table: str) -> str:
    return f"{quote_ident(schema)}.{quote_ident(table)}"


@dataclass
class TableStat:
    table: str
    row_count: int


def fetch_counts(conn, schema: str, tables: Iterable[str]) -> list[TableStat]:
    stats: list[TableStat] = []
    with conn.cursor() as cur:
        for table in tables:
            cur.execute(f"SELECT COUNT(*) FROM {qualified_table(schema, table)}")
            row_count = int(cur.fetchone()[0])
            stats.append(TableStat(table=table, row_count=row_count))
    return stats


def print_plan(stats: list[TableStat], execute: bool) -> None:
    mode = "EXECUTE" if execute else "DRY-RUN"
    print(f"[{mode}] affiliate demo settlement reset plan")
    for stat in stats:
        print(f"  - {stat.table}: {stat.row_count} rows")
    print(f"  - total rows: {sum(stat.row_count for stat in stats)}")


def confirm_or_exit(args: argparse.Namespace, stats: list[TableStat]) -> None:
    if not args.execute:
        print("\nDry-run only. Re-run with --execute to apply deletions.")
        return
    if args.yes:
        return
    print("\nAbout to DELETE rows from the tables above.")
    typed = input("Type RESET DEMO SETTLEMENT to continue: ").strip()
    if typed != "RESET DEMO SETTLEMENT":
        raise SystemExit("Confirmation mismatch. Aborted.")


def execute_reset(conn, schema: str) -> None:
    with conn.cursor() as cur:
        for table in TARGET_TABLES:
            cur.execute(f"DELETE FROM {qualified_table(schema, table)}")


def main() -> int:
    args = parse_args()
    validate_dsn(args.dsn, args.allow_non_demo_dsn)
    psycopg = require_psycopg()

    conn = psycopg.connect(args.dsn)
    try:
        with conn:
            before = fetch_counts(conn, args.schema, TARGET_TABLES)
            print_plan(before, args.execute)
            confirm_or_exit(args, before)
            if not args.execute:
                return 0
            execute_reset(conn, args.schema)
            after = fetch_counts(conn, args.schema, TARGET_TABLES)
            print("\n[POST-CHECK] row counts after delete")
            for stat in after:
                print(f"  - {stat.table}: {stat.row_count} rows")
    finally:
        conn.close()
    return 0


if __name__ == "__main__":
    sys.exit(main())
