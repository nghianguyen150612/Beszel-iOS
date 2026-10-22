#!/usr/bin/env python3
"""Validate exact Go 1.27.3 scan output against the narrow P023 advisory policy."""

import argparse
import json
from pathlib import Path


EXPECTED_GO = "go1.27.3"
SCANNER_VERSION = "v1.9.0"
GO_6608 = "GO-2026-6608"
GO_6809 = "GO-2026-6809"
GO_5932 = "GO-2026-5932"
OPENPGP_PREFIX = "golang.org/x/crypto/openpgp"
EXPECTED_X509_SOURCE_SHA256 = {
    "src/crypto/x509/root_darwin.go": "69b350fe1afa679935ca831c3beea9833d59d8cd62c6f0b4d02f14526dd46ac1",
    "src/crypto/x509/root_windows.go": "ba910c270b9d096f50ccbf9426c03410ad16bc46a86e0ba4237d27c3381d28d0",
    "src/crypto/x509/verify.go": "a239364c8fd240db693be8ec4f82e84958d9b4e4f265ff3923499e2161a34d86",
}
EXPECTED_PROFILES = {
    "host-default-with-tests",
    "host-testing-no_ui-with-tests",
    "host-development-with-tests",
    "host-development-testing-no_ui-with-tests",
    "ios-arm64-testing-no_ui-with-tests",
    "ios-arm64-development-testing-no_ui-with-tests",
}
POLICY_PATH = Path(__file__).resolve().parents[2] / "docs/ios-p023-advisory-policy.json"


def read_jsonl(report: str) -> list[dict]:
    decoder = json.JSONDecoder()
    messages = []
    offset = 0
    while offset < len(report):
        while offset < len(report) and report[offset].isspace():
            offset += 1
        if offset == len(report):
            break
        try:
            message, offset = decoder.raw_decode(report, offset)
        except json.JSONDecodeError as exc:
            raise ValueError(f"invalid govulncheck JSON message {len(messages) + 1}: {exc}") from exc
        if not isinstance(message, dict):
            raise ValueError("govulncheck emitted a non-object JSON message")
        messages.append(message)
    return messages


def validate_x509(evidence: dict, expected_go: str) -> dict:
    if expected_go != EXPECTED_GO or evidence.get("go_version") != EXPECTED_GO:
        raise ValueError("native x509 evidence is not for Go 1.27.3")
    if evidence.get("status") != "PASS" or evidence.get("advisory") != GO_6809:
        raise ValueError("GO-2026-6809 native x509 source validation is not a PASS")
    if evidence.get("fix_disposition") != "native in official Go 1.27.3; custom P019/P022 backport not applied":
        raise ValueError("GO-2026-6809 x509 transition does not identify the native fix")
    if evidence.get("native_source_sha256") != EXPECTED_X509_SOURCE_SHA256:
        raise ValueError("Go 1.27.3 x509 source fingerprints do not match the reviewed CL 851047 output")
    if evidence.get("matches_reviewed_backport_outputs") is not True:
        raise ValueError("native x509 source does not match the reviewed P019/P022 fix output")
    if evidence.get("standard_library_test", {}).get("status") != "PASS":
        raise ValueError("GO-2026-6809 lacks a passing crypto/x509 test")
    return {
        "advisory": GO_6809,
        "remediation_status": "REMEDIATED_AND_VERIFIED",
        "go_version": EXPECTED_GO,
        "native_source_sha256": evidence["native_source_sha256"],
        "standard_library_test": evidence["standard_library_test"],
        "custom_backport_applied": False,
    }


def validate_openpgp_guard(evidence: dict) -> dict:
    if evidence.get("status") != "PASS" or not isinstance(evidence.get("source_files_scanned"), int) or evidence["source_files_scanned"] < 1:
        raise ValueError("OpenPGP import guard did not pass its repository source scan")
    if evidence.get("source_imports") != []:
        raise ValueError("an OpenPGP import is present in repository Go source")
    profiles = evidence.get("dependency_profiles")
    if not isinstance(profiles, list) or {p.get("name") for p in profiles if isinstance(p, dict)} != EXPECTED_PROFILES:
        raise ValueError("OpenPGP dependency evidence does not cover the required host and iOS profiles")
    for profile in profiles:
        if profile.get("openpgp_packages") != []:
            raise ValueError(f"OpenPGP package is reachable in {profile.get('name')}")
    return {
        "status": "PASS",
        "source_files_scanned": evidence["source_files_scanned"],
        "source_openpgp_imports": [],
        "dependency_profiles": profiles,
    }


def validate_policy(policy: dict) -> dict:
    if policy.get("schema") != 1 or policy.get("milestone") != "P023":
        raise ValueError("unsupported P023 advisory policy schema")
    if policy.get("reviewed_on") != "2026-10-22" or policy.get("default_action") != "FAIL_ON_UNREVIEWED_FINDING":
        raise ValueError("P023 policy review date or fail-closed default is invalid")
    if policy.get("raw_scan_must_be_retained") is not True or policy.get("release_authorization") != "NOT_GRANTED":
        raise ValueError("P023 policy must retain raw scans and deny release authorization")
    entries = policy.get("allowlist")
    if not isinstance(entries, list) or len(entries) != 1:
        raise ValueError("P023 policy must contain exactly one narrowly scoped advisory entry")
    entry = entries[0]
    expected = {
        "advisory_id": GO_5932,
        "affected_module": "golang.org/x/crypto",
        "resolved_module_version": "v0.58.0",
        "affected_package_prefix": OPENPGP_PREFIX,
        "allowed_finding_shape": "module-only trace with no package or function symbols",
        "minimum_occurrences": 1,
        "maximum_occurrences": 1,
        "remediation_status": "NOT_REACHABLE_WITH_EVIDENCE",
        "release_owner_decision_required": True,
    }
    for key, value in expected.items():
        if entry.get(key) != value:
            raise ValueError(f"P023 GO-2026-5932 policy has unexpected {key}")
    for required in ("required_evidence", "justification", "fix_availability", "residual_risk"):
        if not entry.get(required):
            raise ValueError(f"P023 GO-2026-5932 policy lacks {required}")
    return entry


def summarize(report: str, expected_go: str, x509_evidence: dict, openpgp_evidence: dict, policy: dict) -> dict:
    x509 = validate_x509(x509_evidence, expected_go)
    openpgp = validate_openpgp_guard(openpgp_evidence)
    policy_entry = validate_policy(policy)
    messages = read_jsonl(report)
    configs = [m["config"] for m in messages if isinstance(m.get("config"), dict)]
    sboms = [m["SBOM"] for m in messages if isinstance(m.get("SBOM"), dict)]
    findings = [m["finding"] for m in messages if isinstance(m.get("finding"), dict)]
    if len(configs) != 1 or not sboms:
        raise ValueError("govulncheck report lacks one config and source SBOM")
    config = configs[0]
    if config.get("go_version") != expected_go or expected_go != EXPECTED_GO:
        raise ValueError(f"govulncheck used {config.get('go_version')!r}; expected {EXPECTED_GO}")
    if config.get("scan_mode") != "source" or config.get("scan_level") != "symbol":
        raise ValueError("govulncheck must complete a symbol-level source scan")
    if SCANNER_VERSION not in config.get("scanner_version", ""):
        raise ValueError(f"govulncheck scanner version is not pinned to {SCANNER_VERSION}")
    roots = {root for sbom in sboms for root in sbom.get("roots", []) if isinstance(root, str)}
    if not {"github.com/henrygd/beszel/internal/cmd/agent", "github.com/henrygd/beszel/internal/cmd/hub"}.issubset(roots):
        raise ValueError("govulncheck SBOM does not include both Agent and Hub roots")

    counts: dict[str, int] = {}
    details = []
    for finding in findings:
        osv = finding.get("osv")
        if not isinstance(osv, str) or not osv:
            raise ValueError("govulncheck emitted a finding without an OSV ID")
        counts[osv] = counts.get(osv, 0) + 1
        trace = []
        for step in finding.get("trace", []):
            if isinstance(step, dict):
                trace.append({key: step[key] for key in ("module", "version", "package", "function") if step.get(key) is not None})
        details.append({"id": osv, "fixed_version": finding.get("fixed_version"), "trace": trace})

    if GO_6608 in counts:
        raise ValueError(f"{GO_6608} remains in Go 1.27.3 source findings")
    if GO_6809 in counts:
        raise ValueError(f"{GO_6809} remains in Go 1.27.3 source findings despite the verified native x509 fix")
    unreviewed = sorted(set(counts) - {GO_5932})
    if unreviewed:
        raise ValueError(f"unreviewed Go advisory findings fail the P023 gate: {unreviewed}")
    if not policy_entry["minimum_occurrences"] <= counts.get(GO_5932, 0) <= policy_entry["maximum_occurrences"]:
        raise ValueError("GO-2026-5932 raw scanner finding is missing or exceeds its reviewed occurrence bound")

    for finding in details:
        if finding["id"] != GO_5932:
            continue
        if any(step.get("package", "").startswith(OPENPGP_PREFIX) or "openpgp" in step.get("function", "").lower() for step in finding["trace"]):
            raise ValueError("GO-2026-5932 contains an OpenPGP package or function trace")
        for step in finding["trace"]:
            if step.get("module") != policy_entry["affected_module"] or step.get("version") != policy_entry["resolved_module_version"]:
                raise ValueError("GO-2026-5932 finding is outside the reviewed x/crypto module/version scope")
            if step.get("package") is not None or step.get("function") is not None:
                raise ValueError("GO-2026-5932 finding is not module-only")

    return {
        "schema": 1,
        "scanner": config.get("scanner_version"),
        "database": config.get("db"),
        "database_last_modified": config.get("db_last_modified"),
        "go_version": EXPECTED_GO,
        "scan_mode": config["scan_mode"],
        "scan_level": config["scan_level"],
        "finding_count": len(findings),
        "finding_ids": sorted(counts),
        "finding_occurrences": counts,
        "findings": details,
        "security_dispositions": [
            x509,
            {
                "advisory": GO_6608,
                "raw_scanner_status": "absent from the exact Go 1.27.3 source scan",
                "affected_versions": "Go 1.27.0 before 1.27.2",
                "fixed_version": "go1.27.2",
                "current_go_version": EXPECTED_GO,
                "remediation_status": "REMEDIATED_AND_VERIFIED",
                "actual_mitigation": "Run the official Go 1.27.3 toolchain, which includes the Go 1.27.2 standard-library fix; exact source scan contains no GO-2026-6608 finding.",
                "release_owner_decision_required": False,
            },
            {
                "advisory": GO_5932,
                "raw_scanner_status": "module-level finding retained" if GO_5932 in counts else "absent from raw source findings",
                "remediation_status": "NOT_REACHABLE_WITH_EVIDENCE",
                "affected_package_imported_or_reachable": False,
                "module_remains_present": True,
                "actual_mitigation": "Static Go AST import guard plus host default, testing/no_ui, development, development/testing/no_ui, iOS testing/no_ui, and iOS development/testing/no_ui go list profiles; no OpenPGP paths found.",
                "not_a_vulnerability_fix": True,
                "release_owner_decision_required": True,
            },
        ],
        "openpgp_guard": openpgp,
        "advisory_policy": {"reviewed_on": policy["reviewed_on"], "default_action": policy["default_action"]},
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--report", required=True, type=Path)
    parser.add_argument("--go-version", required=True)
    parser.add_argument("--x509-evidence", required=True, type=Path)
    parser.add_argument("--openpgp-evidence", required=True, type=Path)
    parser.add_argument("--policy", type=Path, default=POLICY_PATH)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    try:
        result = summarize(
            args.report.read_text(), args.go_version,
            json.loads(args.x509_evidence.read_text()),
            json.loads(args.openpgp_evidence.read_text()),
            json.loads(args.policy.read_text()),
        )
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
