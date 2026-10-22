#!/usr/bin/env python3
"""Validate raw JavaScript audits against the bounded P023 braces residual."""

import argparse
import hashlib
import json
from pathlib import Path
import re


GHSA = "GHSA-vfj7-8cjw-p6xm"
CVE = "CVE-2026-93687"
PACKAGE = "braces"
VULNERABLE_VERSION = "3.0.3"
ALLOWED_IDS = {GHSA, CVE}
ID_PATTERN = re.compile(r"(?:GHSA-[0-9a-z]{4}-[0-9a-z]{4}-[0-9a-z]{4}|CVE-\d{4}-\d{4,})", re.IGNORECASE)
SEVERITIES = {"low", "moderate", "medium", "high", "critical"}
ROOT = Path(__file__).resolve().parents[2]
SOURCE_SHA = re.compile(r"^[0-9a-f]{40}$")


def bun_records(path: Path) -> dict[str, list]:
    records = {}
    for line in path.read_text().splitlines():
        match = re.match(r'\s*"([^"\\]+)": (\[.*\]),?\s*$', line)
        if not match:
            continue
        try:
            value = json.loads(match.group(2).rstrip(","))
        except json.JSONDecodeError:
            continue
        if isinstance(value, list) and len(value) >= 3 and isinstance(value[2], dict):
            records[match.group(1)] = value
    return records


def bun_graph(package_json: dict, lock: Path) -> dict:
    records = bun_records(lock)
    root = package_json
    root_dev = root.get("devDependencies", {})
    required = {"@lingui/cli", "@lingui/vite-plugin"}
    if not required.issubset(root_dev):
        raise ValueError("package.json no longer declares both Lingui packages as build-only devDependencies")
    edges = [
        ("@lingui/vite-plugin", "@lingui/cli"),
        ("@lingui/cli", "chokidar"),
        ("@lingui/cli", "micromatch"),
        ("chokidar", PACKAGE),
        ("micromatch", PACKAGE),
    ]
    paths = []
    for parent, child in edges:
        if parent not in records or child not in records:
            raise ValueError(f"Bun lock is missing required transitive package {parent} or {child}")
        dependencies = records[parent][2].get("dependencies", {})
        if child not in dependencies:
            continue
        paths.append({"from": parent, "to": child, "range": dependencies[child]})
    # Include both paths from the CLI to braces; the plugin reaches that CLI.
    for parent in ("chokidar", "micromatch"):
        if PACKAGE not in records[parent][2].get("dependencies", {}):
            raise ValueError(f"Bun lock no longer records {parent} -> {PACKAGE}")
    braces_version = records.get(PACKAGE, [None])[0]
    if braces_version != f"{PACKAGE}@{VULNERABLE_VERSION}":
        raise ValueError(f"Bun lock resolves {PACKAGE} as {braces_version!r}, expected the reviewed vulnerable version")
    for parent in ("@lingui/cli", "@lingui/vite-plugin"):
        if parent not in records:
            raise ValueError(f"Bun lock is missing {parent}")
    if records["@lingui/vite-plugin"][2].get("dependencies", {}).get("@lingui/cli") is None:
        raise ValueError("Bun lock no longer records @lingui/vite-plugin -> @lingui/cli")
    if records["@lingui/cli"][2].get("dependencies", {}).get("chokidar") is None or records["@lingui/cli"][2].get("dependencies", {}).get("micromatch") is None:
        raise ValueError("Bun lock no longer records both Lingui CLI paths to the affected package")
    return {
        "package": PACKAGE,
        "version": VULNERABLE_VERSION,
        "dev_dependency": True,
        "paths": [
            ["@lingui/vite-plugin", "@lingui/cli", "chokidar", PACKAGE],
            ["@lingui/vite-plugin", "@lingui/cli", "micromatch", PACKAGE],
        ],
    }


def npm_resolve(packages: dict, parent: str, dependency: str) -> str | None:
    current = parent
    while True:
        candidate = f"{current}/node_modules/{dependency}" if current else f"node_modules/{dependency}"
        if candidate in packages:
            return candidate
        if not current:
            return None
        current = current.rsplit("/", 1)[0] if "/" in current else ""


def npm_graph(package_json: dict, lock: Path) -> dict:
    data = json.loads(lock.read_text())
    packages = data.get("packages")
    if not isinstance(packages, dict):
        raise ValueError("npm lockfile does not contain a package graph")
    root = packages.get("")
    if not isinstance(root, dict):
        raise ValueError("npm lockfile has no root package record")
    root_dev = root.get("devDependencies", {})
    required = {"@lingui/cli", "@lingui/vite-plugin"}
    if not required.issubset(root_dev) or not required.issubset(package_json.get("devDependencies", {})):
        raise ValueError("npm lock no longer places the Lingui CLI and plugin in devDependencies")
    start_cli = "node_modules/@lingui/cli"
    start_plugin = "node_modules/@lingui/vite-plugin"
    if start_cli not in packages or start_plugin not in packages:
        raise ValueError("npm lock is missing a required Lingui package")
    plugin_deps = packages[start_plugin].get("dependencies", {})
    cli_deps = packages[start_cli].get("dependencies", {})
    if plugin_deps.get("@lingui/cli") is None:
        raise ValueError("npm lock no longer records @lingui/vite-plugin -> @lingui/cli")
    if not {"chokidar", "micromatch"}.issubset(cli_deps):
        raise ValueError("npm lock no longer records both Lingui CLI paths to the affected package")

    paths = []
    for middle in ("chokidar", "micromatch"):
        resolved_middle = npm_resolve(packages, start_cli, middle)
        if not resolved_middle:
            raise ValueError(f"npm lock cannot resolve Lingui CLI dependency {middle}")
        resolved_braces = npm_resolve(packages, resolved_middle, PACKAGE)
        if not resolved_braces:
            raise ValueError(f"npm lock cannot resolve {PACKAGE} from {middle}")
        path_names = ["@lingui/vite-plugin", "@lingui/cli", middle, PACKAGE]
        for package_path in (start_plugin, start_cli, resolved_middle, resolved_braces):
            package = packages[package_path]
            if package.get("dev") is not True:
                raise ValueError(f"npm lock marks {package_path} as runtime rather than dev-only")
        paths.append(path_names)
    resolved = packages.get("node_modules/braces", {})
    if resolved.get("version") != VULNERABLE_VERSION or resolved.get("dev") is not True:
        raise ValueError(f"npm lock resolves {PACKAGE} as {resolved.get('version')!r}, expected dev-only {VULNERABLE_VERSION}")
    return {
        "package": PACKAGE,
        "version": resolved["version"],
        "dev_dependency": True,
        "paths": paths,
    }


def audit_report(path: Path, manager: str, exit_code: int) -> dict:
    raw = path.read_text()
    try:
        data = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise ValueError(f"{manager} audit did not produce valid JSON: {exc}") from exc
    if not isinstance(data, dict):
        raise ValueError(f"{manager} audit returned an error payload")
    if "error" in data:
        message = data.get("message", "")
        if manager == "npm" and isinstance(message, str) and message.startswith(
            "network timeout at: https://registry.npmjs.org/-/npm/v1/security/audits/"
        ):
            return {
                "manager": manager,
                "report_sha256": hashlib.sha256(raw.encode()).hexdigest(),
                "report_bytes": len(raw.encode()),
                "advisory_ids": [],
                "command_exit_code": exit_code,
                "status": "UNAVAILABLE_NETWORK",
                "error_message": message,
                "raw_error_retained": True,
            }
        raise ValueError(f"{manager} audit returned an error payload")
    id_map = {item.upper(): item for item in ALLOWED_IDS}
    normalized_ids = {item.upper() for item in ID_PATTERN.findall(raw)}
    if GHSA.upper() not in normalized_ids or normalized_ids - set(id_map):
        raise ValueError(f"{manager} raw scan must retain only the reviewed {GHSA} / {CVE} advisory; found {sorted(normalized_ids)}")
    ids = {id_map[item] for item in normalized_ids}

    # npm uses a package-keyed vulnerabilities object. Bun may expose the same
    # map or a list of advisory records; walk both forms without dropping raw.
    package_nodes = []
    def walk(value, key_path=()):
        if isinstance(value, dict):
            severity = value.get("severity")
            name = value.get("name", value.get("package", value.get("dependency")))
            serialized = json.dumps(value, sort_keys=True)
            if (name == PACKAGE or PACKAGE in key_path) and isinstance(severity, str) and severity.lower() in SEVERITIES and (GHSA in serialized or CVE in serialized):
                package_nodes.append(value)
            for key, child in value.items():
                walk(child, (*key_path, key))
        elif isinstance(value, list):
            for child in value:
                walk(child, key_path)
    walk(data)
    if not package_nodes:
        raise ValueError(f"{manager} audit JSON lacks a severity-bearing {PACKAGE} record linked to {GHSA}")
    if any(node.get("severity", "").lower() != "high" for node in package_nodes):
        raise ValueError(f"{manager} reports {PACKAGE} at an unexpected severity")
    metadata = data.get("metadata", {})
    counts = metadata.get("vulnerabilities", {}) if isinstance(metadata, dict) else {}
    if isinstance(counts, dict) and any(count for level, count in counts.items() if level in SEVERITIES and level != "high"):
        raise ValueError(f"{manager} raw audit includes additional severity classes: {counts}")
    if isinstance(counts, dict) and counts.get("high", 1) > 1:
        raise ValueError(f"{manager} raw audit includes more than the one reviewed high package: {counts}")
    if exit_code not in (0, 1):
        raise ValueError(f"{manager} audit exited {exit_code}; inspect the retained report")
    return {
        "manager": manager,
        "report_sha256": hashlib.sha256(raw.encode()).hexdigest(),
        "report_bytes": len(raw.encode()),
        "advisory_ids": sorted(ids),
        "severity": "high",
        "vulnerable_package_count": 1,
        "command_exit_code": exit_code,
        "exit_status_interpreted_as": "reviewed vulnerability present" if exit_code == 1 else "audit command success; advisory still reported",
        "raw_finding_retained": True,
        "status": "PASS_WITH_REVIEWED_ADVISORY",
    }


def validate_osv_lock_scan(report: dict, lock_path: Path, candidate_source_sha: str) -> dict:
    if not SOURCE_SHA.fullmatch(candidate_source_sha) or report.get("candidate_source_sha") != candidate_source_sha:
        raise ValueError("OSV npm-lock scan is not tied to the current candidate source commit")
    if report.get("schema") != 1 or report.get("status") != "PASS_WITH_REVIEWED_ADVISORY":
        raise ValueError("OSV npm-lock scan did not pass its bounded advisory policy")
    if report.get("endpoint") != "https://api.osv.dev/v1/querybatch":
        raise ValueError("OSV npm-lock scan used an unreviewed data source")
    lock_sha = hashlib.sha256(lock_path.read_bytes()).hexdigest()
    if report.get("package_lock_sha256") != lock_sha:
        raise ValueError("OSV npm-lock scan covers a different package-lock.json")
    if not isinstance(report.get("package_lock_nodes_scanned"), int) or report["package_lock_nodes_scanned"] < 1:
        raise ValueError("OSV npm-lock scan did not inspect package nodes")
    if not isinstance(report.get("unique_package_version_queries"), int) or report["unique_package_version_queries"] < 1:
        raise ValueError("OSV npm-lock scan did not issue exact package/version queries")
    if report.get("finding_ids") != [GHSA] or len(report.get("findings", [])) != 1:
        raise ValueError("OSV npm-lock scan must retain only the exact reviewed GHSA finding")
    finding = report["findings"][0]
    expected = {
        "advisory_id": GHSA,
        "cve_id": CVE,
        "package": PACKAGE,
        "version": VULNERABLE_VERSION,
        "first_patched_version": None,
        "raw_scanner_status": "retained by full npm lock OSV query",
    }
    if any(finding.get(key) != value for key, value in expected.items()):
        raise ValueError("OSV npm-lock scan finding differs from the reviewed braces residual")
    nodes = finding.get("install_nodes")
    if not isinstance(nodes, list) or not nodes or any(node.get("dev") is not True for node in nodes):
        raise ValueError("OSV braces finding is not confined to npm dev dependencies")
    events = finding.get("affected_events")
    if not isinstance(events, list) or not any(event.get("last_affected") == VULNERABLE_VERSION for event in events) or any("fixed" in event for event in events):
        raise ValueError("OSV npm-lock scan does not prove the reviewed affected range and missing fix")
    return report


def validate_policy(policy: dict) -> dict:
    if policy.get("schema") != 1 or policy.get("milestone") != "P023" or policy.get("reviewed_on") != "2026-10-22":
        raise ValueError("unexpected P023 JavaScript advisory policy metadata")
    entries = policy.get("frontend_residuals")
    if not isinstance(entries, list) or len(entries) != 1:
        raise ValueError("P023 JavaScript policy must contain exactly one bounded frontend residual")
    entry = entries[0]
    expected = {
        "advisory_id": GHSA,
        "cve_id": CVE,
        "package": PACKAGE,
        "resolved_version": VULNERABLE_VERSION,
        "severity": "high",
        "first_patched_version": None,
        "execution_classification": "BUILD_ONLY_TRANSITIVE",
        "remediation_status": "MITIGATED_WITH_RESIDUAL_RISK",
        "release_owner_decision_required": True,
    }
    for key, value in expected.items():
        if entry.get(key) != value:
            raise ValueError(f"P023 JavaScript policy has unexpected {key}")
    if not entry.get("actual_mitigation") or not entry.get("exposure_limit"):
        raise ValueError("P023 JavaScript policy lacks its mitigation or exposure limit")
    fallback = policy.get("npm_audit_fallback")
    expected_fallback = {
        "allowed_only_for_error_prefix": "network timeout at: https://registry.npmjs.org/-/npm/v1/security/audits/",
        "raw_npm_error_must_be_retained": True,
        "fallback_scanner": "https://api.osv.dev/v1/querybatch",
        "scan_scope": "every exact package/version node in package-lock.json",
        "allowed_finding_ids": [GHSA],
        "new_or_missing_findings_action": "FAIL",
        "release_authorization": "NOT_GRANTED",
    }
    if fallback != expected_fallback:
        raise ValueError("P023 npm audit fallback policy is missing or wider than the reviewed OSV lock scan")
    return entry


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--bun-report", required=True, type=Path)
    parser.add_argument("--bun-exit-code", required=True, type=int)
    parser.add_argument("--npm-report", required=True, type=Path)
    parser.add_argument("--npm-exit-code", required=True, type=int)
    parser.add_argument("--npm-osv-report", required=True, type=Path)
    parser.add_argument("--candidate-source-sha", required=True)
    parser.add_argument("--package-json", type=Path, default=ROOT / "internal/site/package.json")
    parser.add_argument("--bun-lock", type=Path, default=ROOT / "internal/site/bun.lock")
    parser.add_argument("--npm-lock", type=Path, default=ROOT / "internal/site/package-lock.json")
    parser.add_argument("--policy", type=Path, default=ROOT / "docs/ios-p023-advisory-policy.json")
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    try:
        if not SOURCE_SHA.fullmatch(args.candidate_source_sha):
            raise ValueError("candidate source commit must be a full 40-character Git SHA")
        package_json = json.loads(args.package_json.read_text())
        policy = validate_policy(json.loads(args.policy.read_text()))
        graph = {"bun": bun_graph(package_json, args.bun_lock), "npm": npm_graph(package_json, args.npm_lock)}
        audits = {
            "bun": audit_report(args.bun_report, "bun", args.bun_exit_code),
            "npm": audit_report(args.npm_report, "npm", args.npm_exit_code),
        }
        osv = validate_osv_lock_scan(
            json.loads(args.npm_osv_report.read_text()), args.npm_lock, args.candidate_source_sha
        )
        if audits["bun"].get("status") != "PASS_WITH_REVIEWED_ADVISORY" or audits["bun"].get("advisory_ids") != [GHSA]:
            raise ValueError("Bun audit must retain the reviewed braces advisory")
        if audits["npm"].get("status") == "UNAVAILABLE_NETWORK":
            if audits["npm"].get("command_exit_code") == 0:
                raise ValueError("npm audit reported a network error with a successful exit status")
        elif audits["npm"].get("status") != "PASS_WITH_REVIEWED_ADVISORY" or audits["npm"].get("advisory_ids") != [GHSA]:
            raise ValueError("npm audit must retain the reviewed braces advisory")
        if graph["bun"]["version"] != graph["npm"]["version"]:
            raise ValueError("Bun and npm locks resolve different braces versions")
        result = {
            "schema": 1,
            "candidate_source_sha": args.candidate_source_sha,
            "reviewed_on": "2026-10-22",
            "package_manager_lock_consistency": "PASS",
            "dependency_audit_status": "PASS_WITH_REVIEWED_ADVISORY",
            "npm_lock_fallback": "full-lock OSV query used because npm registry audit endpoint timed out" if audits["npm"].get("status") == "UNAVAILABLE_NETWORK" else "not needed; npm audit completed",
            "dependency_graph": graph,
            "raw_audits": audits,
            "npm_osv_lock_scan": osv,
            "finding": {
                **policy,
                "raw_scanner_status": "OPEN in Bun audit and full npm-lock OSV query; npm registry audit endpoint unavailable" if audits["npm"].get("status") == "UNAVAILABLE_NETWORK" else "OPEN in Bun and npm audit reports",
                "source_dependency_version": graph["bun"]["version"],
                "production_runtime_dependency": False,
                "generated_hub_assets_include_node_runtime": False,
                "raw_scanner_reports_hashed": True,
                "not_fixed_or_removed": True,
            },
            "release_authorization": "NOT_GRANTED",
        }
        if args.output.is_symlink():
            raise ValueError("summary output path is symlinked")
        args.output.parent.mkdir(parents=True, exist_ok=True)
        rendered = json.dumps(result, indent=2) + "\n"
        args.output.write_text(rendered)
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
        raise SystemExit(f"ERROR: {exc}") from exc
    print(rendered, end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
