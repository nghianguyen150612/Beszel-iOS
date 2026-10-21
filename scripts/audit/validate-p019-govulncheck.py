#!/usr/bin/env python3
"""Verify pinned Go vulnerability evidence and the reviewed x509 backport."""

import argparse
import json
from pathlib import Path

TARGET = "GO-2026-6608"
BACKPORTED = "GO-2026-6809"
SCANNER_VERSION = "v1.9.0"
BACKPORT_PATCH_SHA256 = "111bf066e27cb60df43881cbfa9b1d22b7682b720628b4b691a960a853b48150"
BACKPORT_COMMIT = "7f86f957633ee31a1c17a1eb8452ca7a3d9bc141"
BACKPORT_BASELINE = {
    "src/crypto/x509/root_darwin.go": "e84652a32aa3806e13c4845107b57f5da5a069ec7b91855da4a645d5fa9ea5b5",
    "src/crypto/x509/root_windows.go": "cae38f158aebec723791d96194f30bf1639a4e9e765878bcb5ee4eec5e0a1568",
    "src/crypto/x509/verify.go": "8b3d059f20a75ac8a103d97b473fbe90e01d40654c4548c50d9b0937f54f3c8f",
}
BACKPORT_PATCHED = {
    "src/crypto/x509/root_darwin.go": "69b350fe1afa679935ca831c3beea9833d59d8cd62c6f0b4d02f14526dd46ac1",
    "src/crypto/x509/root_windows.go": "ba910c270b9d096f50ccbf9426c03410ad16bc46a86e0ba4237d27c3381d28d0",
    "src/crypto/x509/verify.go": "a239364c8fd240db693be8ec4f82e84958d9b4e4f265ff3923499e2161a34d86",
}


def validate_backport(evidence: dict, expected_go: str) -> dict:
    if evidence.get("status") != "PASS" or evidence.get("advisory") != BACKPORTED:
        raise ValueError("GO-2026-6809 backport evidence is not a PASS")
    if evidence.get("go_version") != expected_go or expected_go != "go1.27.2":
        raise ValueError("GO-2026-6809 backport evidence uses the wrong Go version")
    if evidence.get("upstream_commit") != BACKPORT_COMMIT:
        raise ValueError("GO-2026-6809 evidence does not identify the reviewed Go fix")
    if evidence.get("patch_sha256") != BACKPORT_PATCH_SHA256:
        raise ValueError("GO-2026-6809 evidence has an unexpected patch digest")
    if evidence.get("baseline_source_sha256") != BACKPORT_BASELINE:
        raise ValueError("GO-2026-6809 evidence is not tied to the pinned Go 1.27.2 source")
    if evidence.get("standard_library_test", {}).get("status") != "PASS":
        raise ValueError("GO-2026-6809 backport lacks a passing crypto/x509 test")
    patched = evidence.get("patched_source_sha256")
    if patched != BACKPORT_PATCHED:
        raise ValueError("GO-2026-6809 evidence has unexpected patched-source fingerprints")
    return {
        "advisory": BACKPORTED,
        "disposition": "backported official Go fix",
        "upstream_commit": BACKPORT_COMMIT,
        "patch_sha256": BACKPORT_PATCH_SHA256,
        "standard_library_test": evidence["standard_library_test"],
        "patched_source_sha256": patched,
    }


def summarize(report: str, expected_go: str, backport_evidence: dict | None = None) -> dict:
    messages = []
    decoder = json.JSONDecoder()
    offset = 0
    number = 0
    while offset < len(report):
        while offset < len(report) and report[offset].isspace():
            offset += 1
        if offset == len(report):
            break
        number += 1
        try:
            message, offset = decoder.raw_decode(report, offset)
        except json.JSONDecodeError as exc:
            raise ValueError(f"invalid govulncheck JSON message {number}: {exc}") from exc
        if not isinstance(message, dict):
            raise ValueError(f"govulncheck JSON message {number} is not an object")
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
    backport = None
    if BACKPORTED in counts:
        if backport_evidence is None:
            raise ValueError(f"{BACKPORTED} is reachable and requires verified backport evidence")
        backport = validate_backport(backport_evidence, expected_go)
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
        "security_dispositions": [backport] if backport else [],
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--report", required=True, type=Path)
    parser.add_argument("--go-version", required=True)
    parser.add_argument("--backport-evidence", type=Path)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    try:
        backport = json.loads(args.backport_evidence.read_text()) if args.backport_evidence else None
        result = summarize(args.report.read_text(), args.go_version, backport)
    except (OSError, ValueError) as exc:
        raise SystemExit(f"ERROR: {exc}") from exc
    text = json.dumps(result, indent=2) + "\n"
    if args.output:
        args.output.write_text(text)
    print(text, end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
