#!/usr/bin/env python3
"""Query OSV for every exact npm package/version in the P023 npm lockfile."""

import argparse
import hashlib
import json
from pathlib import Path
import re
import urllib.error
import urllib.request


QUERY_ENDPOINT = "https://api.osv.dev/v1/querybatch"
VULN_ENDPOINT = "https://api.osv.dev/v1/vulns/"
REVIEWED_ID = "GHSA-vfj7-8cjw-p6xm"
REVIEWED_ALIAS = "CVE-2026-93687"
REVIEWED_PACKAGE = "braces"
REVIEWED_VERSION = "3.0.3"
SOURCE_SHA = re.compile(r"^[0-9a-f]{40}$")
BATCH_SIZE = 100
TIMEOUT_SECONDS = 30


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def lock_queries(lock: dict) -> tuple[list[dict], int, int]:
    packages = lock.get("packages")
    if lock.get("lockfileVersion") != 3 or not isinstance(packages, dict) or "" not in packages:
        raise ValueError("npm lock must be a package-lock v3 with a root record")
    nodes = []
    skipped_links = 0
    for install_path, package in sorted(packages.items()):
        if install_path == "":
            continue
        if not isinstance(package, dict):
            raise ValueError(f"npm lock package record is invalid: {install_path}")
        if package.get("link") is True:
            skipped_links += 1
            continue
        version = package.get("version")
        if not isinstance(version, str) or not version:
            raise ValueError(f"npm lock package has no exact version: {install_path}")
        if not install_path.startswith("node_modules/"):
            raise ValueError(f"npm lock package path is not an installed npm package: {install_path}")
        installed_name = install_path.rsplit("node_modules/", 1)[-1]
        name = package.get("name", installed_name)
        if not isinstance(name, str) or not name:
            raise ValueError(f"npm lock package name is invalid: {install_path}")
        nodes.append({
            "path": install_path,
            "name": name,
            "version": version,
            "dev": package.get("dev") is True,
        })
    if not nodes:
        raise ValueError("npm lock contains no versioned package nodes")

    grouped: dict[tuple[str, str], list[dict]] = {}
    for node in nodes:
        key = node["name"], node["version"]
        grouped.setdefault(key, []).append(node)
    queries = [
        {"package": {"name": name, "ecosystem": "npm"}, "version": version}
        for name, version in sorted(grouped)
    ]
    return queries, len(nodes), skipped_links


def request_json(url: str, payload: dict | None = None) -> tuple[bytes, dict]:
    body = None if payload is None else json.dumps(payload, separators=(",", ":")).encode()
    request = urllib.request.Request(
        url,
        data=body,
        headers={
            "Accept": "application/json",
            "Content-Type": "application/json",
            "User-Agent": "Beszel-iOS-P023-security-review",
        },
        method="GET" if body is None else "POST",
    )
    try:
        with urllib.request.urlopen(request, timeout=TIMEOUT_SECONDS) as response:
            raw = response.read()
    except (OSError, urllib.error.URLError) as exc:
        raise ValueError(f"OSV request failed: {exc}") from exc
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise ValueError(f"OSV returned invalid JSON from {url}") from exc
    if not isinstance(parsed, dict):
        raise ValueError(f"OSV returned a non-object response from {url}")
    return raw, parsed


def scan(lock_bytes: bytes, source_sha: str, batch_responses: list[dict], advisory: dict) -> tuple[dict, dict]:
    if not SOURCE_SHA.fullmatch(source_sha):
        raise ValueError("candidate source commit must be a full 40-character Git SHA")
    try:
        lock = json.loads(lock_bytes)
    except json.JSONDecodeError as exc:
        raise ValueError(f"npm lock is invalid JSON: {exc}") from exc
    queries, package_nodes, skipped_links = lock_queries(lock)
    grouped: dict[tuple[str, str], list[dict]] = {}
    for install_path, package in lock["packages"].items():
        if not install_path or not isinstance(package, dict) or package.get("link") is True:
            continue
        installed_name = install_path.rsplit("node_modules/", 1)[-1]
        name = package.get("name", installed_name)
        grouped.setdefault((name, package["version"]), []).append({
            "path": install_path,
            "dev": package.get("dev") is True,
        })

    if len(batch_responses) != (len(queries) + BATCH_SIZE - 1) // BATCH_SIZE:
        raise ValueError("OSV response batch count does not match the complete npm lock scan")
    matches = []
    offset = 0
    for batch in batch_responses:
        count = min(BATCH_SIZE, len(queries) - offset)
        results = batch.get("results") if isinstance(batch, dict) else None
        if not isinstance(results, list) or len(results) != count:
            raise ValueError("OSV response omitted or added results for npm package queries")
        for query, result in zip(queries[offset:offset + count], results):
            if not isinstance(result, dict) or not isinstance(result.get("vulns", []), list):
                raise ValueError("OSV result has an invalid vulnerability list")
            name = query["package"]["name"]
            version = query["version"]
            for vuln in result.get("vulns", []):
                if not isinstance(vuln, dict) or not isinstance(vuln.get("id"), str):
                    raise ValueError("OSV returned a vulnerability without an ID")
                matches.append({
                    "id": vuln["id"],
                    "modified": vuln.get("modified"),
                    "package": name,
                    "version": version,
                    "install_nodes": grouped[(name, version)],
                })
        offset += count
    if offset != len(queries):
        raise ValueError("OSV response did not cover every npm package/version query")
    ids = sorted({match["id"] for match in matches})
    if ids != [REVIEWED_ID]:
        raise ValueError(f"npm lock OSV scan found missing or unreviewed advisory IDs: {ids}")
    reviewed = [m for m in matches if m["id"] == REVIEWED_ID]
    if len(reviewed) != 1 or reviewed[0]["package"] != REVIEWED_PACKAGE or reviewed[0]["version"] != REVIEWED_VERSION:
        raise ValueError("npm lock OSV scan did not produce exactly the reviewed braces@3.0.3 finding")
    if not reviewed[0]["install_nodes"] or any(node["dev"] is not True for node in reviewed[0]["install_nodes"]):
        raise ValueError("npm lock OSV finding is not confined to dev-only install nodes")

    if advisory.get("id") != REVIEWED_ID or REVIEWED_ALIAS not in advisory.get("aliases", []):
        raise ValueError("OSV advisory detail does not match the reviewed GHSA and CVE IDs")
    affected_records = [
        item for item in advisory.get("affected", [])
        if item.get("package", {}).get("name") == REVIEWED_PACKAGE
        and item.get("package", {}).get("ecosystem") == "npm"
    ]
    if len(affected_records) != 1:
        raise ValueError("OSV advisory no longer identifies exactly the npm braces package")
    events = [
        event
        for interval in affected_records[0].get("ranges", [])
        for event in interval.get("events", [])
    ]
    if not any(event.get("introduced") == "0" for event in events) or not any(event.get("last_affected") == REVIEWED_VERSION for event in events):
        raise ValueError("OSV advisory affected-version range changed from the reviewed braces range")
    if any("fixed" in event for event in events):
        raise ValueError("OSV advisory now lists a fixed version; reassess the dependency before accepting this residual")

    request_payload = {"queries": queries}
    raw_responses = {"endpoint": QUERY_ENDPOINT, "batch_responses": batch_responses}
    severity_vectors = advisory.get("severity", [])
    summary = {
        "schema": 1,
        "status": "PASS_WITH_REVIEWED_ADVISORY",
        "candidate_source_sha": source_sha,
        "scanner": "OSV querybatch API",
        "reviewed_on": "2026-10-22",
        "endpoint": QUERY_ENDPOINT,
        "package_lock_sha256": sha256(lock_bytes),
        "package_lock_nodes_scanned": package_nodes,
        "workspace_links_skipped": skipped_links,
        "unique_package_version_queries": len(queries),
        "canonical_request_json_sha256": sha256(json.dumps(request_payload, separators=(",", ":")).encode()),
        "canonical_response_json_sha256": sha256(json.dumps(raw_responses, separators=(",", ":")).encode()),
        "finding_ids": ids,
        "findings": [{
            "advisory_id": REVIEWED_ID,
            "cve_id": REVIEWED_ALIAS,
            "package": REVIEWED_PACKAGE,
            "version": REVIEWED_VERSION,
            "install_nodes": reviewed[0]["install_nodes"],
            "severity_vectors": severity_vectors,
            "affected_events": events,
            "first_patched_version": None,
            "raw_scanner_status": "retained by full npm lock OSV query",
        }],
        "npm_audit_relationship": "independent full-lock cross-check; fallback is permitted only after the configured npm registry timeout",
        "release_authorization": "NOT_GRANTED",
    }
    return summary, {"queries": queries, "batch_responses": batch_responses, "advisory": advisory}


def write_json(path: Path, value: dict) -> None:
    if path.is_symlink():
        raise ValueError(f"refusing symlink output: {path}")
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2) + "\n", encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--lockfile", type=Path, default=Path("internal/site/package-lock.json"))
    parser.add_argument("--candidate-source-sha", required=True)
    parser.add_argument("--request-output", required=True, type=Path)
    parser.add_argument("--response-output", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    try:
        lock_bytes = args.lockfile.read_bytes()
        lock = json.loads(lock_bytes)
        queries, _, _ = lock_queries(lock)
        batch_responses = []
        for start in range(0, len(queries), BATCH_SIZE):
            _, response = request_json(QUERY_ENDPOINT, {"queries": queries[start:start + BATCH_SIZE]})
            batch_responses.append(response)
        _, advisory = request_json(VULN_ENDPOINT + REVIEWED_ID)
        summary, raw = scan(lock_bytes, args.candidate_source_sha, batch_responses, advisory)
        write_json(args.request_output, {"endpoint": QUERY_ENDPOINT, "queries": raw["queries"]})
        write_json(args.response_output, raw)
        write_json(args.output, summary)
    except (OSError, ValueError, json.JSONDecodeError, urllib.error.URLError) as exc:
        raise SystemExit(f"ERROR: {exc}") from exc
    print(json.dumps(summary, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
