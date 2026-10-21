#!/usr/bin/env python3
"""Require an exact pinned Go source scan and verify GO-2026-6608 is absent."""

import argparse
import json
from pathlib import Path

TARGET = "GO-2026-6608"
SCANNER_VERSION = "v1.9.0"


def summarize(report: str, expected_go: str) -> dict:
    messages = []
    for number, line in enumerate(report.splitlines(), 1):
        if not line.strip():
            continue
        try:
            message = json.loads(line)
        except json.JSONDecodeError as exc:
            raise ValueError(f"invalid govulncheck JSON on line {number}: {exc}") from exc
        if not isinstance(message, dict):
            raise ValueError(f"govulncheck JSON line {number} is not an object")
        messages.append(message)
    configs = [m["config"] for m in messages if isinstance(m.get("config"), dict)]
    sboms = [m["SBOM"] for m in messages if isinstance(m.get("SBOM"), dict)]
    findings = [m["finding"] for m in messages if isinstance(m.get("finding"), dict)]
    if len(configs) != 1 or not sboms:
        raise ValueError("govulncheck report lacks one config and source SBOM")
    config = configs[0]
    if config.get("go_version") != expected_go:
        raise ValueError(f"govulncheck used {config.get('go_version')!r}; expected {expected_go}")
    if config.get("scan_mode") != "source" or config.get("scan_level") != "symbol":
        raise ValueError("govulncheck must complete a symbol-level source scan")
    if SCANNER_VERSION not in config.get("scanner_version", ""):
        raise ValueError(f"govulncheck scanner version is not pinned to {SCANNER_VERSION}")
    if any(sbom.get("go_version") not in (None, expected_go) for sbom in sboms):
        raise ValueError("govulncheck SBOM reports a different Go version")
    counts: dict[str, int] = {}
    details = []
    for finding in findings:
        osv = finding.get("osv")
        if not isinstance(osv, str) or not osv:
            raise ValueError("govulncheck emitted a finding without an OSV ID")
        counts[osv] = counts.get(osv, 0) + 1
        traces = []
        for trace in finding.get("trace", []):
            if isinstance(trace, dict):
                traces.append({key: trace[key] for key in ("module", "version", "package", "function") if key in trace})
        details.append({"id": osv, "fixed_version": finding.get("fixed_version", ""), "trace": traces})
    if TARGET in counts:
        raise ValueError(f"{TARGET} remains in the source-level vulnerability findings")
    return {
        "schema": 1,
        "scanner": config.get("scanner_version"),
        "database": config.get("db"),
        "database_last_modified": config.get("db_last_modified"),
        "go_version": expected_go,
        "scan_mode": config["scan_mode"],
        "scan_level": config["scan_level"],
        "target_advisory": TARGET,
        "target_advisory_in_findings": False,
        "finding_count": len(findings),
        "finding_ids": sorted(counts),
        "finding_occurrences": counts,
        "findings": details,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--report", required=True, type=Path)
    parser.add_argument("--go-version", required=True)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    try:
        result = summarize(args.report.read_text(), args.go_version)
    except (OSError, ValueError) as exc:
        raise SystemExit(f"ERROR: {exc}") from exc
    text = json.dumps(result, indent=2) + "\n"
    if args.output:
        args.output.write_text(text)
    print(text, end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
