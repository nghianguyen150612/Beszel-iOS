#!/usr/bin/env python3
"""Read-only structural validator for the P003 v0.21 readiness package."""

from __future__ import annotations

import argparse
import csv
import re
import sys
from collections import Counter
from pathlib import Path
from typing import Iterable


SHA_RE = re.compile(r"(?<![0-9a-fA-F])[0-9a-fA-F]{40}(?![0-9a-fA-F])")
PROMPT_RE = re.compile(r"^P(\d{3})$")
SHA_VALUE_RE = re.compile(r"^[0-9a-f]{40}$")
PATCH_ID_RE = re.compile(r"^IOS-[A-Z0-9-]+$")

REQUIRED_SCENARIOS = {
    *(f"COMP-PAIR-{n:03}" for n in range(1, 9)),
    "PROTO-WS-01",
    "PROTO-SSH-01",
    "PROTO-RECONNECT-01",
    "PROTO-KEEPALIVE-01",
    "PROTO-CANCEL-01",
    "PROTO-VERSION-01",
    "PROTO-CBOR-01",
    "PROTO-JSONV2-01",
    "PROTO-HUBHEADERS-01",
    "PROTO-AUTH-01",
    "PROTO-TLS-01",
    "PROTO-MON-TLS-01",
    "PROTO-MON-01",
    "PROTO-MON-RESYNC-01",
    "IOS-BUILD-01",
    "IOS-A7-RUNTIME-01",
    "IOS-METADATA-01",
    "IOS-CPU-RAM-01",
    "IOS-BATTERY-01",
    "IOS-NETWORK-01",
    "IOS-DISK-01",
    "IOS-PKG-01",
    "IOS-WIFI-01",
    "IOS-UNSUPPORTED-01",
    "IOS-REMOTE-01",
}
REQUIRED_ACCEPTANCE_CASES = {
    "A7-BUILD-01",
    "A7-RUNTIME-01",
    "A7-DEVICE-01",
    "DB-BACKUP-01",
    "DB-MIG-01",
    "DB-RESTORE-01",
    "DB-INTERRUPT-01",
}
SEVERITY_VALUES = {
    "Critical": 5,
    "High": 4,
    "Moderate": 3,
    "Low": 2,
    "Negligible": 1,
}
LIKELIHOOD_VALUES = {
    "Rare": 1,
    "Unlikely": 2,
    "Possible": 3,
    "Likely": 4,
    "Almost certain": 5,
}
DETECTABILITY_VALUES = {"High", "Medium", "Low"}
RISK_STATUS_VALUES = {
    "confirmed incompatibility",
    "potential regression",
    "untested risk",
}
COMPAT_RESULT_VALUES = {
    "verified",
    "historically validated",
    "planned",
    "unsupported",
    "blocked",
}
ACCEPTANCE_STATUS_VALUES = {
    "verified",
    "historically validated",
    "planned",
    "unsupported",
    "blocked",
}
GATE_VALUES = {f"Gate {letter}" for letter in "ABCDEF"}
ENTRY_TYPES = {"upstream_commit", "logical_ios_patch"}


def _read_csv(
    path: Path, required_columns: Iterable[str], errors: list[str]
) -> list[dict[str, str]]:
    if not path.is_file():
        errors.append(f"missing required file: {path}")
        return []
    try:
        with path.open(newline="", encoding="utf-8-sig") as stream:
            reader = csv.DictReader(stream)
            headers = set(reader.fieldnames or ())
            missing = set(required_columns) - headers
            if missing:
                errors.append(f"{path}: missing columns: {', '.join(sorted(missing))}")
                return []
            return [dict(row) for row in reader]
    except (OSError, csv.Error, UnicodeError) as exc:
        errors.append(f"cannot read {path}: {exc}")
        return []


def _refs(value: str) -> list[str]:
    return [part.strip() for part in value.split(";") if part.strip() and part.strip().lower() != "none"]


def _check_owner(value: str, context: str, errors: list[str]) -> None:
    match = PROMPT_RE.fullmatch(value.strip())
    if not match or not 3 <= int(match.group(1)) <= 18:
        errors.append(f"{context}: invalid owner prompt {value!r}; expected P003–P018")


def _check_unique(rows: list[dict[str, str]], field: str, context: str, errors: list[str]) -> set[str]:
    values = [row.get(field, "").strip() for row in rows]
    counts = Counter(value for value in values if value)
    duplicates = sorted(value for value, count in counts.items() if count > 1)
    if duplicates:
        errors.append(f"{context}: duplicate {field}: {', '.join(duplicates)}")
    missing = sum(not value for value in values)
    if missing:
        errors.append(f"{context}: {missing} row(s) have an empty {field}")
    return set(counts)


def _check_evidence(
    rows: list[dict[str, str]], field: str, context: str, errors: list[str]
) -> None:
    for index, row in enumerate(rows, start=2):
        if not row.get(field, "").strip():
            errors.append(f"{context} row {index}: required evidence field {field} is empty")


def _acceptance_table(path: Path, errors: list[str]) -> list[list[str]]:
    if not path.is_file():
        errors.append(f"missing required file: {path}")
        return []
    try:
        content = path.read_text(encoding="utf-8")
    except (OSError, UnicodeError) as exc:
        errors.append(f"cannot read {path}: {exc}")
        return []
    records: list[list[str]] = []
    for line in content.splitlines():
        stripped = line.strip()
        if not stripped.startswith("|"):
            continue
        cells = [cell.strip() for cell in stripped.strip("|").split("|")]
        if cells and re.fullmatch(r"[A-Z][A-Z0-9-]+", cells[0]):
            records.append(cells)
    values = [row[0] for row in records]
    duplicates = sorted(value for value, count in Counter(values).items() if count > 1)
    if duplicates:
        errors.append(f"{path.name}: duplicate acceptance IDs: {', '.join(duplicates)}")
    return records


def validate(root: Path) -> tuple[list[str], dict[str, int]]:
    """Return structural errors and coverage counts without modifying files."""
    errors: list[str] = []
    counts: dict[str, int] = {}
    docs = root / "docs"

    matrix = _read_csv(
        docs / "upstream-v021-change-matrix.csv",
        {"sha", "priority", "release_batch", "integration_prompts", "ios_patch_ids", "evidence_paths"},
        errors,
    )
    p2_shas: set[str] = set()
    if matrix:
        _check_unique(matrix, "sha", "P002 change matrix", errors)
        for index, row in enumerate(matrix, start=2):
            sha = row.get("sha", "").strip()
            if not SHA_VALUE_RE.fullmatch(sha):
                errors.append(f"P002 change matrix row {index}: malformed upstream SHA {sha!r}")
            else:
                p2_shas.add(sha)
        counts["upstream_commits"] = len(p2_shas)
        if len(matrix) != 178 or len(p2_shas) != 178:
            errors.append(f"P002 change matrix: expected 178 unique upstream rows, found {len(p2_shas)}")
        priorities = Counter(row.get("priority", "").strip() for row in matrix)
        expected_priorities = {"P0": 21, "P1": 74, "P2": 83}
        if dict(priorities) != expected_priorities:
            errors.append(f"P002 priorities drifted: expected {expected_priorities}, found {dict(priorities)}")
        batches = Counter(row.get("release_batch", "").strip() for row in matrix)
        expected_batches = {"P004": 25, "P005": 116, "P006": 37}
        if dict(batches) != expected_batches:
            errors.append(f"P002 release batches drifted: expected {expected_batches}, found {dict(batches)}")

    patch_map = _read_csv(
        docs / "upstream-v021-ios-patch-map.csv",
        {"ios_commit_sha", "logical_patch_id", "upstream_commits_at_risk", "responsible_prompt"},
        errors,
    )
    patch_ids: set[str] = set()
    if patch_map:
        patch_ids = {row.get("logical_patch_id", "").strip() for row in patch_map}
        patch_ids.discard("")
        counts["logical_ios_patches"] = len(patch_ids)
        if len(patch_ids) != 13:
            errors.append(f"P002 patch map: expected 13 logical patch IDs, found {len(patch_ids)}")
        for index, row in enumerate(patch_map, start=2):
            patch_id = row.get("logical_patch_id", "").strip()
            if not PATCH_ID_RE.fullmatch(patch_id):
                errors.append(f"P002 patch map row {index}: malformed patch ID {patch_id!r}")

    matrix_patch_ids: set[str] = set()
    for row in matrix:
        matrix_patch_ids.update(_refs(row.get("ios_patch_ids", "")))
    unknown_matrix_patches = matrix_patch_ids - patch_ids
    if unknown_matrix_patches:
        errors.append("P002 matrix references unknown logical patch IDs: " + ", ".join(sorted(unknown_matrix_patches)))

    crosswalk = _read_csv(
        docs / "upstream-v021-implementation-crosswalk.csv",
        {"mapping_id", "entry_type", "upstream_sha", "logical_patch_id", "original_owner", "canonical_owner_prompt", "integration_owner_prompt", "affected_subsystem", "evidence_references", "reason_for_reassignment"},
        errors,
    )
    _check_unique(crosswalk, "mapping_id", "P003 crosswalk", errors)
    _check_evidence(crosswalk, "evidence_references", "P003 crosswalk", errors)
    commit_rows: dict[str, dict[str, str]] = {}
    patch_rows: dict[str, dict[str, str]] = {}
    if crosswalk:
        for index, row in enumerate(crosswalk, start=2):
            entry_type = row.get("entry_type", "").strip()
            if entry_type not in ENTRY_TYPES:
                errors.append(f"P003 crosswalk row {index}: invalid entry_type {entry_type!r}")
            _check_owner(row.get("canonical_owner_prompt", ""), f"P003 crosswalk row {index}", errors)
            if not row.get("original_owner", "").strip() or not row.get("reason_for_reassignment", "").strip():
                errors.append(f"P003 crosswalk row {index}: original owner and reason are required")
            if not row.get("affected_subsystem", "").strip():
                errors.append(f"P003 crosswalk row {index}: affected_subsystem is required")
            if entry_type == "upstream_commit":
                sha = row.get("upstream_sha", "").strip()
                if sha in commit_rows:
                    errors.append(f"P003 crosswalk: duplicate upstream SHA {sha}")
                commit_rows[sha] = row
                if not SHA_VALUE_RE.fullmatch(sha):
                    errors.append(f"P003 crosswalk row {index}: malformed upstream SHA {sha!r}")
                if row.get("integration_owner_prompt", "").strip() not in {"P004", "P005", "P006"}:
                    errors.append(f"P003 crosswalk row {index}: invalid integration owner")
            elif entry_type == "logical_ios_patch":
                patch_id = row.get("logical_patch_id", "").strip()
                if patch_id in patch_rows:
                    errors.append(f"P003 crosswalk: duplicate logical patch mapping {patch_id}")
                patch_rows[patch_id] = row
                if not PATCH_ID_RE.fullmatch(patch_id):
                    errors.append(f"P003 crosswalk row {index}: malformed logical patch ID {patch_id!r}")
        if set(commit_rows) != p2_shas:
            missing = sorted(p2_shas - set(commit_rows))
            extra = sorted(set(commit_rows) - p2_shas)
            if missing:
                errors.append(f"P003 crosswalk missing canonical ownership for upstream SHAs: {', '.join(missing)}")
            if extra:
                errors.append(f"P003 crosswalk references unknown upstream SHAs: {', '.join(extra)}")
        if set(patch_rows) != patch_ids:
            missing = sorted(patch_ids - set(patch_rows))
            extra = sorted(set(patch_rows) - patch_ids)
            if missing:
                errors.append(f"P003 crosswalk missing logical iOS patches: {', '.join(missing)}")
            if extra:
                errors.append(f"P003 crosswalk references unknown logical iOS patches: {', '.join(extra)}")
        for row in matrix:
            sha = row.get("sha", "").strip()
            mapped = commit_rows.get(sha)
            if mapped and (
                mapped.get("original_owner", "").strip() != row.get("integration_prompts", "").strip()
                or mapped.get("integration_owner_prompt", "").strip() != row.get("release_batch", "").strip()
            ):
                errors.append(f"P003 crosswalk does not preserve P002 owner/batch for {sha}")
            if mapped:
                mapped_patches = set(_refs(mapped.get("logical_patch_id", "")))
                expected_patches = set(_refs(row.get("ios_patch_ids", "")))
                if mapped_patches != expected_patches:
                    errors.append(f"P003 crosswalk patch references drifted for upstream SHA {sha}")

    risk_rows = _read_csv(
        docs / "upstream-v021-risk-register.csv",
        {"risk_id", "relevant_upstream_shas", "related_ios_logical_patch_ids", "severity", "likelihood", "risk_score", "detectability", "required_test", "owner_prompt", "blocking_gate", "source_evidence", "status"},
        errors,
    )
    _check_unique(risk_rows, "risk_id", "P003 risk register", errors)
    _check_evidence(risk_rows, "source_evidence", "P003 risk register", errors)
    p0_shas = {row.get("sha", "").strip() for row in matrix if row.get("priority", "").strip() == "P0"}
    covered_p0: set[str] = set()
    risk_patch_ids: set[str] = set()
    for index, row in enumerate(risk_rows, start=2):
        severity = row.get("severity", "").strip()
        likelihood = row.get("likelihood", "").strip()
        status = row.get("status", "").strip()
        gate = row.get("blocking_gate", "").strip()
        detectability = row.get("detectability", "").strip()
        _check_owner(row.get("owner_prompt", ""), f"P003 risk row {index}", errors)
        if severity not in SEVERITY_VALUES:
            errors.append(f"P003 risk row {index}: invalid severity {severity!r}")
        if likelihood not in LIKELIHOOD_VALUES:
            errors.append(f"P003 risk row {index}: invalid likelihood {likelihood!r}")
        if detectability not in DETECTABILITY_VALUES:
            errors.append(f"P003 risk row {index}: invalid detectability {detectability!r}")
        if status not in RISK_STATUS_VALUES:
            errors.append(f"P003 risk row {index}: invalid status {status!r}")
        if gate not in GATE_VALUES:
            errors.append(f"P003 risk row {index}: invalid blocking_gate {gate!r}")
        if severity in SEVERITY_VALUES and likelihood in LIKELIHOOD_VALUES:
            expected_score = SEVERITY_VALUES[severity] * LIKELIHOOD_VALUES[likelihood]
            try:
                actual_score = int(row.get("risk_score", ""))
            except ValueError:
                actual_score = -1
            if actual_score != expected_score:
                errors.append(f"P003 risk row {index}: risk_score {row.get('risk_score')!r} != {expected_score}")
        refs = _refs(row.get("relevant_upstream_shas", ""))
        for sha in refs:
            if not SHA_VALUE_RE.fullmatch(sha):
                errors.append(f"P003 risk row {index}: malformed upstream SHA reference {sha!r}")
            elif sha not in p2_shas:
                errors.append(f"P003 risk row {index}: unknown upstream SHA reference {sha}")
            if sha in p0_shas:
                covered_p0.add(sha)
        risk_patch_ids.update(_refs(row.get("related_ios_logical_patch_ids", "")))
        for field in ("title", "area", "failure_mode", "trigger_or_prerequisite", "affected_data_security_functionality", "mitigation", "rollback_containment_action", "required_test"):
            if not row.get(field, "").strip():
                errors.append(f"P003 risk row {index}: required field {field} is empty")
    counts["p0_risks"] = len(covered_p0)
    if covered_p0 != p0_shas:
        errors.append(f"P003 risk register P0 coverage is {len(covered_p0)}/{len(p0_shas)}; missing: {', '.join(sorted(p0_shas-covered_p0))}")
    if risk_patch_ids != patch_ids:
        missing = sorted(patch_ids - risk_patch_ids)
        extra = sorted(risk_patch_ids - patch_ids)
        if missing:
            errors.append(f"P003 risk register missing logical patch references: {', '.join(missing)}")
        if extra:
            errors.append(f"P003 risk register references unknown logical patch IDs: {', '.join(extra)}")

    compat_rows = _read_csv(
        docs / "upstream-v021-compatibility-matrix.csv",
        {"scenario_id", "category", "agent_version_platform", "hub_version_platform", "required_capability", "expected_behavior", "protocol_api_dependency", "minimum_supported_version_if_established", "required_test", "test_fixture_environment", "evidence_currently_available", "result", "owning_prompt", "blocking_condition"},
        errors,
    )
    compat_ids = _check_unique(compat_rows, "scenario_id", "P003 compatibility matrix", errors)
    _check_evidence(compat_rows, "evidence_currently_available", "P003 compatibility matrix", errors)
    if compat_rows:
        missing_scenarios = REQUIRED_SCENARIOS - compat_ids
        if missing_scenarios:
            errors.append("P003 compatibility matrix missing required scenarios: " + ", ".join(sorted(missing_scenarios)))
        for index, row in enumerate(compat_rows, start=2):
            _check_owner(row.get("owning_prompt", ""), f"P003 compatibility row {index}", errors)
            result = row.get("result", "").strip()
            if result not in COMPAT_RESULT_VALUES:
                errors.append(f"P003 compatibility row {index}: invalid result {result!r}")
            for field in ("category", "agent_version_platform", "hub_version_platform", "required_capability", "expected_behavior", "protocol_api_dependency", "required_test", "test_fixture_environment", "blocking_condition"):
                if not row.get(field, "").strip():
                    errors.append(f"P003 compatibility row {index}: required field {field} is empty")
    counts["compatibility_scenarios"] = len(compat_ids)

    acceptance_rows = _acceptance_table(docs / "upstream-v021-acceptance.md", errors)
    acceptance_ids = {row[0] for row in acceptance_rows}
    acceptance_vi_rows = _acceptance_table(docs / "upstream-v021-acceptance.vi.md", errors)
    for index, row in enumerate(acceptance_rows, start=1):
        if len(row) != 9:
            errors.append(f"P003 acceptance row {index}: expected 9 table fields, found {len(row)}")
            continue
        _check_owner(row[6], f"P003 acceptance row {index}", errors)
        if row[7] not in GATE_VALUES:
            errors.append(f"P003 acceptance row {index}: invalid gate {row[7]!r}")
        if row[8] not in ACCEPTANCE_STATUS_VALUES:
            errors.append(f"P003 acceptance row {index}: invalid status {row[8]!r}")
        if any(not value for value in row[1:]):
            errors.append(f"P003 acceptance row {index}: objective, procedure, expected result, and evidence fields are required")
    acceptance_vi_ids = {row[0] for row in acceptance_vi_rows}
    if acceptance_vi_ids != acceptance_ids:
        errors.append("Vietnamese acceptance case IDs do not match the English plan")
    missing_cases = REQUIRED_ACCEPTANCE_CASES - acceptance_ids
    if missing_cases:
        errors.append("P003 acceptance plan missing required cases: " + ", ".join(sorted(missing_cases)))
    counts["acceptance_cases"] = len(acceptance_ids)

    # Every 40-hex SHA in the P003 CSV reference fields must belong to the P002
    # canonical upstream set. The patch map's ios_commit_sha column is excluded:
    # those are historical iOS commits, not upstream targets.
    reference_files = [
        docs / "upstream-v021-implementation-crosswalk.csv",
        docs / "upstream-v021-risk-register.csv",
        docs / "upstream-v021-compatibility-matrix.csv",
    ]
    for path in reference_files:
        try:
            with path.open(newline="", encoding="utf-8-sig") as stream:
                for index, row in enumerate(csv.DictReader(stream), start=2):
                    for field, value in row.items():
                        if not value:
                            continue
                        for sha in SHA_RE.findall(value):
                            sha = sha.lower()
                            if sha not in p2_shas:
                                errors.append(f"{path.name} row {index}: unknown upstream SHA in {field}: {sha}")
        except (OSError, csv.Error, UnicodeError) as exc:
            errors.append(f"cannot scan references in {path}: {exc}")

    # Validate upstream hashes called out by the original P002 patch map too.
    for index, row in enumerate(patch_map, start=2):
        for sha in _refs(row.get("upstream_commits_at_risk", "")):
            if not SHA_VALUE_RE.fullmatch(sha) or sha not in p2_shas:
                errors.append(f"P002 patch map row {index}: malformed or unknown upstream SHA {sha!r}")

    # Stable IDs are unique both within each inventory and across inventories.
    all_ids: list[tuple[str, str]] = []
    all_ids += [("risk_id", row.get("risk_id", "").strip()) for row in risk_rows]
    all_ids += [("scenario_id", row.get("scenario_id", "").strip()) for row in compat_rows]
    all_ids += [("mapping_id", row.get("mapping_id", "").strip()) for row in crosswalk]
    all_ids += [("acceptance_id", value) for value in acceptance_ids]
    global_id_counts = Counter(value for _, value in all_ids if value)
    duplicate_global_ids = sorted(value for value, count in global_id_counts.items() if count > 1)
    if duplicate_global_ids:
        errors.append("duplicate stable IDs across P003 inventories: " + ", ".join(duplicate_global_ids))

    return errors, counts


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--root",
        type=Path,
        default=Path(__file__).resolve().parents[2],
        help="repository root to audit (read-only; defaults to this checkout)",
    )
    args = parser.parse_args(argv)
    errors, counts = validate(args.root.resolve())
    if counts:
        print("Readiness coverage: " + ", ".join(f"{key}={value}" for key, value in sorted(counts.items())))
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        print(f"Readiness validation failed with {len(errors)} error(s).", file=sys.stderr)
        return 1
    print("Readiness validation passed; no files were modified.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
