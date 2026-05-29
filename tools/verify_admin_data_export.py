#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import sys
import urllib.error
import urllib.parse
import urllib.request


EXPECTED_TYPE = "sub2api-data"
EXPECTED_VERSION = 1


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Verify admin JSON export endpoints on a running Sub2API backend."
    )
    parser.add_argument(
        "--base-url",
        required=True,
        help="Backend base URL, for example http://127.0.0.1:8080",
    )
    parser.add_argument(
        "--admin-jwt",
        default="",
        help="Admin JWT used as Authorization: Bearer <token>.",
    )
    parser.add_argument(
        "--admin-api-key",
        default="",
        help="Admin API key used as x-api-key.",
    )
    return parser.parse_args()


def build_headers(args: argparse.Namespace) -> dict[str, str]:
    headers = {"Accept": "application/json"}
    if args.admin_jwt:
        headers["Authorization"] = f"Bearer {args.admin_jwt}"
    if args.admin_api_key:
        headers["x-api-key"] = args.admin_api_key
    if len(headers) == 1:
        raise SystemExit("Provide --admin-jwt or --admin-api-key.")
    return headers


def fetch_json(url: str, headers: dict[str, str]) -> dict:
    request = urllib.request.Request(url, headers=headers, method="GET")
    try:
        with urllib.request.urlopen(request, timeout=20) as response:
            body = response.read().decode("utf-8")
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise SystemExit(f"HTTP {exc.code} for {url}\n{detail}") from exc
    except urllib.error.URLError as exc:
        raise SystemExit(f"Request failed for {url}: {exc}") from exc

    try:
        payload = json.loads(body)
    except json.JSONDecodeError as exc:
        raise SystemExit(f"Invalid JSON from {url}: {exc}") from exc

    if not isinstance(payload, dict):
        raise SystemExit(f"Unexpected response type from {url}: {type(payload).__name__}")
    return payload


def require(condition: bool, message: str) -> None:
    if not condition:
        raise SystemExit(message)


def validate_export_payload(name: str, payload: dict) -> None:
    require(payload.get("code") == 0, f"{name}: expected envelope code=0, got {payload.get('code')!r}")
    data = payload.get("data")
    require(isinstance(data, dict), f"{name}: expected data object")
    require(data.get("type") == EXPECTED_TYPE, f"{name}: expected data.type={EXPECTED_TYPE!r}")
    require(
        data.get("version") == EXPECTED_VERSION,
        f"{name}: expected data.version={EXPECTED_VERSION!r}",
    )
    exported_at = data.get("exported_at")
    require(isinstance(exported_at, str) and exported_at.strip(), f"{name}: exported_at is required")
    require(isinstance(data.get("proxies"), list), f"{name}: proxies must be an array")
    require(isinstance(data.get("accounts"), list), f"{name}: accounts must be an array")


def main() -> int:
    args = parse_args()
    headers = build_headers(args)
    base_url = args.base_url.rstrip("/")

    endpoints = [
        ("accounts export", "/api/v1/admin/accounts/data?include_proxies=true"),
        ("proxies export", "/api/v1/admin/proxies/data"),
    ]

    for name, path in endpoints:
        url = urllib.parse.urljoin(base_url + "/", path.lstrip("/"))
        payload = fetch_json(url, headers)
        validate_export_payload(name, payload)
        data = payload["data"]
        print(
            f"[ok] {name}: exported_at={data['exported_at']} "
            f"accounts={len(data['accounts'])} proxies={len(data['proxies'])}"
        )

    return 0


if __name__ == "__main__":
    sys.exit(main())
