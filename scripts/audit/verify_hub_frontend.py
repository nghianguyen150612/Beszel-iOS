#!/usr/bin/env python3
"""Check the frontend assets and SPA routes served by a running candidate Hub."""

from __future__ import annotations

import argparse
import hashlib
import json
import sys
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.parse import quote, urlsplit, urlunsplit
from urllib.request import Request, urlopen


ROUTES = (
    "/",
    "/containers",
    "/smart",
    "/monitors",
    "/system/p015-synthetic-system",
    "/settings",
    "/settings/general",
    "/forgot-password",
    "/request-otp",
)
UNKNOWN_PATHS = (
    "/robots.txt",
    "/.env",
    "/.git/config",
    "/go.mod",
    "/assets/missing-p015.js",
    "/static/missing-p015.svg",
)
MIME_BY_SUFFIX = {
    ".css": "text/css",
    ".js": "text/javascript",
    ".json": "application/json",
    ".png": "image/png",
    ".svg": "image/svg+xml",
    ".woff2": "font/woff2",
}
EMPTY_COLLECTIONS = ("users", "systems", "system_stats", "alerts_history", "user_settings", "alerts")


class HubValidationError(RuntimeError):
    """Raised when the candidate Hub serves incomplete or unexpected content."""


def _request(base_url: str, path: str, headers: dict[str, str] | None = None) -> tuple[int, dict[str, str], bytes]:
    url = base_url.rstrip("/") + path
    try:
        with urlopen(Request(url, headers=headers or {}), timeout=15) as response:
            return response.status, dict(response.headers.items()), response.read()
    except HTTPError as exc:
        return exc.code, dict(exc.headers.items()), exc.read()
    except (OSError, URLError) as exc:
        raise HubValidationError(f"request failed for {url}: {exc}") from exc


def _read_manifest(path: Path) -> list[tuple[str, str]]:
    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except OSError as exc:
        raise HubValidationError(f"cannot read asset manifest {path}: {exc}") from exc
    entries: list[tuple[str, str]] = []
    for number, line in enumerate(lines, 1):
        parts = line.split("  ", 1)
        if len(parts) != 2 or len(parts[0]) != 64:
            raise HubValidationError(f"invalid asset manifest line {number}")
        entries.append((parts[0], parts[1]))
    if not entries or len({name for _, name in entries}) != len(entries):
        raise HubValidationError("asset manifest is empty or contains duplicate paths")
    return entries


def validate_hub(base_url: str, manifest_path: Path, expect_empty_db: bool = False) -> dict[str, object]:
    parsed = urlsplit(base_url)
    prefix = parsed.path.rstrip("/")
    if (
        parsed.scheme not in {"http", "https"}
        or not parsed.netloc
        or parsed.query
        or parsed.fragment
        or "//" in prefix
        or any(part in {".", ".."} for part in prefix.split("/") if part)
    ):
        raise HubValidationError("--hub-url must be an HTTP(S) origin with an optional safe path prefix")
    base_url = urlunsplit((parsed.scheme, parsed.netloc, prefix, "", ""))
    expected_base_path = f'{prefix}/' if prefix else "/"

    route_results = {}
    for route in ROUTES:
        status, headers, body = _request(base_url, route)
        if status != 200 or b"<html" not in body.lower():
            raise HubValidationError(f"supported SPA route {route} returned HTTP {status}")
        route_results[route] = {"status": status, "content_type": headers.get("Content-Type", ""), "bytes": len(body)}

    unknown_results = {}
    for path in UNKNOWN_PATHS:
        status, _headers, _body = _request(base_url, path)
        if status != 404:
            raise HubValidationError(f"unknown/static path {path} returned HTTP {status}, expected 404")
        unknown_results[path] = status

    entries = _read_manifest(manifest_path)
    served_count = 0
    mime_counts: dict[str, int] = {}
    for expected_digest, name in entries:
        if name == "index.html":
            continue  # Hub injects BASE_PATH/version information into this response.
        status, headers, body = _request(base_url, "/" + quote(name, safe="/"))
        if status != 200:
            raise HubValidationError(f"embedded asset {name} returned HTTP {status}")
        actual_digest = hashlib.sha256(body).hexdigest()
        if actual_digest != expected_digest:
            raise HubValidationError(f"embedded asset hash mismatch for {name}: {actual_digest}")
        suffix = Path(name).suffix.lower()
        expected_mime = MIME_BY_SUFFIX.get(suffix)
        actual_mime = headers.get("Content-Type", "").split(";", 1)[0].strip()
        if expected_mime and actual_mime != expected_mime:
            raise HubValidationError(f"MIME mismatch for {name}: expected {expected_mime}, got {actual_mime!r}")
        cache_control = headers.get("Cache-Control", "").lower()
        if name.startswith(("assets/", "static/")) and "public" not in cache_control:
            raise HubValidationError(f"static asset {name} is missing public cache policy")
        mime_counts[actual_mime] = mime_counts.get(actual_mime, 0) + 1
        served_count += 1

    javascript_asset = next(name for _, name in entries if name.endswith(".js"))
    excessive_ranges = "bytes=" + ",".join(f"{index}-{index}" for index in range(17))
    range_status, _range_headers, _range_body = _request(
        base_url,
        "/" + quote(javascript_asset, safe="/"),
        {"Range": excessive_ranges},
    )
    if range_status != 400:
        raise HubValidationError(f"static request with 17 byte ranges returned HTTP {range_status}, expected 400")

    index_status, _index_headers, index_body = _request(base_url, "/")
    base_path_json = json.dumps(expected_base_path).encode("utf-8")
    if index_status != 200 or b'"BASE_PATH":' + base_path_json not in index_body:
        raise HubValidationError(f"Hub index response is missing dynamic base-path injection for {expected_base_path!r}")

    collection_results = {}
    if expect_empty_db:
        for collection in EMPTY_COLLECTIONS:
            status, _headers, body = _request(base_url, f"/api/collections/{collection}/records")
            if status != 200:
                raise HubValidationError(f"unauthenticated {collection} list returned HTTP {status}")
            try:
                payload = json.loads(body)
            except json.JSONDecodeError as exc:
                raise HubValidationError(f"unauthenticated {collection} list was not JSON") from exc
            items = payload.get("items") if isinstance(payload, dict) else None
            if items != []:
                raise HubValidationError(f"fresh unauthenticated {collection} list was not empty")
            collection_results[collection] = {"status": status, "items": 0}

    return {
        "status": "PASS",
        "hub_url": base_url,
        "spa_routes": route_results,
        "unknown_and_missing_paths": unknown_results,
        "embedded_assets_verified": served_count,
        "asset_manifest_entries": len(entries),
        "mime_counts": mime_counts,
        "index_base_path_injection": "PASS",
        "excessive_byte_range_rejection": {"status": range_status, "maximum_ranges": 16},
        "fresh_unauthenticated_collections": collection_results,
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--hub-url", required=True, help="candidate Hub origin (for example http://127.0.0.1:8097)")
    parser.add_argument("--manifest", type=Path, default=Path("docs/ios-frontend-assets-manifest.sha256"))
    parser.add_argument("--expect-empty-db", action="store_true", help="require locked collections to return no records")
    parser.add_argument("--output", type=Path, help="write the JSON result to this path")
    args = parser.parse_args(argv)
    try:
        result = validate_hub(args.hub_url, args.manifest, args.expect_empty_db)
    except HubValidationError as exc:
        print(f"Hub frontend validation failed: {exc}", file=sys.stderr)
        return 1
    rendered = json.dumps(result, ensure_ascii=False, indent=2) + "\n"
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(rendered, encoding="utf-8")
    print(rendered, end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
