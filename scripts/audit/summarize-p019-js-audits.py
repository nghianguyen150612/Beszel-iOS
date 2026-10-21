#!/usr/bin/env python3
"""Summarize Bun/npm JSON audits and fail on command errors or critical advisories."""

import argparse
import hashlib
import json
from pathlib import Path
import re

IDS = re.compile(r"(?:GHSA-[0-9a-z]{4}-[0-9a-z]{4}-[0-9a-z]{4}|CVE-\d{4}-\d{4,})", re.IGNORECASE)
KNOWN_BUILD_ADVISORY = "GHSA-vfj7-8cjw-p6xm"


def collect_critical(value) -> int:
    if isinstance(value, dict):
        return int(value.get("severity") == "critical") + sum(
            collect_critical(v) for k, v in value.items() if k != "critical"
        )
    if isinstance(value, list):
        return sum(collect_critical(v) for v in value)
    return 0


def audit(path: Path, manager: str, exit_code: int) -> dict:
    raw = path.read_text()
    try:
        data = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise ValueError(f"{manager} audit did not produce valid JSON: {exc}") from exc
    ids = set(IDS.findall(raw))
    if isinstance(data, dict):
        for key in ("advisories", "vulnerabilities"):
            values = data.get(key)
            if isinstance(values, dict):
                ids.update(str(item) for item in values if str(item).isdigit())
    metadata = data.get("metadata", {}) if isinstance(data, dict) else {}
    vulnerability_counts = metadata.get("vulnerabilities", {}) if isinstance(metadata, dict) else {}
    critical = vulnerability_counts.get("critical") if isinstance(vulnerability_counts, dict) else None
    critical = max(critical if isinstance(critical, int) else 0, collect_critical(data))
    if exit_code != 0:
        raise ValueError(f"{manager} audit exited {exit_code}; inspect its retained report")
    if critical:
        raise ValueError(f"{manager} audit reported {critical} critical findings")
    return {
        "manager": manager,
        "report_sha256": hashlib.sha256(raw.encode()).hexdigest(),
        "report_bytes": len(raw.encode()),
        "advisory_ids": sorted(ids, key=str.casefold),
        "critical_finding_count": critical,
        "known_build_advisory_open_in_report": KNOWN_BUILD_ADVISORY.casefold() in {item.casefold() for item in ids},
        "exit_code": exit_code,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--bun-report", type=Path, required=True)
    parser.add_argument("--bun-exit-code", type=int, required=True)
    parser.add_argument("--npm-report", type=Path, required=True)
    parser.add_argument("--npm-exit-code", type=int, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    try:
        result = {
            "schema": 1,
            "bun": audit(args.bun_report, "bun", args.bun_exit_code),
            "npm": audit(args.npm_report, "npm", args.npm_exit_code),
            "critical_findings": 0,
            "unresolved_high_findings_are_retained_for_review": True,
        }
    except (OSError, ValueError) as exc:
        raise SystemExit(f"ERROR: {exc}") from exc
    output = json.dumps(result, indent=2) + "\n"
    args.output.write_text(output)
    print(output, end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
