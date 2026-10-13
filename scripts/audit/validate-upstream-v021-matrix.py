#!/usr/bin/env python3
"""Read-only validator for the frozen Beszel v0.21 upstream audit inventory."""

from __future__ import annotations

import argparse
import csv
import re
import subprocess
import sys
from collections import Counter
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable


@dataclass(frozen=True)
class FrozenRefs:
    base: str = "f204dc17e64bba444478c60210d018e43a777193"
    target: str = "bb23f99b3eeaced9608e11e280e7c986dba650be"
    v020: str = "a0dc19eacf6902d2f6934a3efd7ea7277198c40b"
    v021: str = "ec96c134660c998f7eb044e349e3715b65733fe1"
    ios_base: str = "f204dc17e64bba444478c60210d018e43a777193"
    ios_target: str = "a6b73365147fc8c3a820552a145d370418e5b680"


DEFAULT_REFS = FrozenRefs()
FULL_SHA = re.compile(r"^[0-9a-f]{40}$")
REQUIRED_MATRIX_COLUMNS = (
    "sha",
    "subject",
    "commit_type",
    "release_batch",
    "areas",
    "change_type",
    "priority",
    "ios_applicability",
    "ios_patch_ids",
    "integration_prompts",
    "risk_summary",
    "evidence_paths",
    "review_status",
)
REQUIRED_PATCH_COLUMNS = (
    "ios_commit_sha",
    "logical_patch_id",
    "changed_files",
    "implementation_purpose",
    "upstream_files_or_apis",
    "upstream_commits_at_risk",
    "preservation_action",
    "responsible_prompt",
    "risk_category",
    "review_status",
)
ALLOWED_AREAS = {
    "Agent",
    "Agent:CPU/RAM",
    "Agent:Cgroups",
    "Agent:Containers",
    "Agent:Disk I/O",
    "Agent:GPU",
    "Agent:NVMe SMART",
    "Agent:Network",
    "Agent:Package updates",
    "Agent:Platform collectors",
    "Agent:Polling/errors",
    "Agent:Systemd",
    "Agent:Wi-Fi RSSI",
    "Agent:ZFS",
    "Protocol",
    "Protocol:CBOR/JSON",
    "Protocol:Headers/auth",
    "Protocol:SSH",
    "Protocol:TLS",
    "Protocol:WebSocket",
    "Hub",
    "Hub:API",
    "Hub:Alerts",
    "Hub:Authorization",
    "Hub:Monitor records",
    "Hub:Polling",
    "Hub:Sessions/users",
    "Database",
    "Database:History",
    "Database:Migrations",
    "Database:Records",
    "Frontend",
    "Frontend:Alerts UI",
    "Frontend:Charts",
    "Frontend:Dashboard",
    "Frontend:Dependencies",
    "Frontend:Embedded assets",
    "Frontend:Localization",
    "Frontend:Logs",
    "Frontend:Network monitors",
    "Frontend:Package updates UI",
    "Frontend:Settings",
    "Frontend:Wi-Fi charts",
    "Platform",
    "Platform:Docker/release",
    "Platform:Linux",
    "Platform:macOS",
    "Platform:Windows",
    "Platform:iOS",
    "CI",
    "Tests",
    "Documentation",
    "Dependencies",
    "Dependencies:Bun/npm",
    "Dependencies:Go",
    "Installer",
    "Installer:Bootstrap/checksum",
    "Installer:LaunchDaemons",
    "Installer:State/rollback",
}
ALLOWED_TYPES = {
    "Feature",
    "fix",
    "refactor",
    "security",
    "dependency",
    "test",
    "documentation",
    "CI",
    "other",
}
ALLOWED_PRIORITIES = {"P0", "P1", "P2", "none"}
ALLOWED_APPLICABILITY = {"Direct", "shared", "Hub-only", "platform-other", "none"}
ALLOWED_REVIEW = {"Reviewed", "unresolved"}
ALLOWED_BATCHES = {"P004", "P005", "P006"}


def git(repo: Path, *args: str) -> str:
    try:
        return subprocess.check_output(
            ["git", "-C", str(repo), *args], text=True, stderr=subprocess.PIPE
        ).strip()
    except subprocess.CalledProcessError as exc:
        detail = exc.stderr.strip() if exc.stderr else str(exc)
        raise ValueError(f"git {' '.join(args)} failed: {detail}") from exc


def commit_list(repo: Path, rev_range: str) -> list[str]:
    output = git(repo, "rev-list", "--reverse", "--topo-order", rev_range)
    return output.splitlines() if output else []


def csv_rows(path: Path, required: Iterable[str], label: str, errors: list[str]) -> list[dict[str, str]]:
    if not path.is_file():
        errors.append(f"{label} file is missing: {path}")
        return []
    try:
        with path.open(newline="", encoding="utf-8") as stream:
            reader = csv.DictReader(stream)
            headers = reader.fieldnames or []
            missing = [field for field in required if field not in headers]
            if missing:
                errors.append(f"{label} missing required columns: {', '.join(missing)}")
            return [dict(row) for row in reader]
    except (OSError, csv.Error) as exc:
        errors.append(f"{label} cannot be read as CSV: {exc}")
        return []


def expected_batches(repo: Path, refs: FrozenRefs) -> tuple[dict[str, set[str]], list[str]]:
    errors: list[str] = []
    for ancestor, descendant, label in (
        (refs.base, refs.v020, "fork point -> v0.20.0"),
        (refs.v020, refs.v021, "v0.20.0 -> v0.21.0"),
        (refs.v021, refs.target, "v0.21.0 -> frozen target"),
        (refs.base, refs.target, "fork point -> frozen target"),
        (refs.ios_base, refs.ios_target, "fork point -> P001 iOS snapshot"),
    ):
        try:
            git(repo, "merge-base", "--is-ancestor", ancestor, descendant)
        except ValueError:
            errors.append(f"frozen ancestry check failed for {label}: {ancestor} is not an ancestor of {descendant}")
    batches = {
        "P004": set(commit_list(repo, f"{refs.base}..{refs.v020}")),
        "P005": set(commit_list(repo, f"{refs.v020}..{refs.v021}")),
        "P006": set(commit_list(repo, f"{refs.v021}..{refs.target}")),
    }
    total = set(commit_list(repo, f"{refs.base}..{refs.target}"))
    union = set().union(*batches.values())
    if union != total:
        errors.append(
            "release batches do not exactly cover frozen range "
            f"(missing {len(total - union)}, unexpected {len(union - total)})"
        )
    duplicated = sum(len(batch) for batch in batches.values()) - len(union)
    if duplicated:
        errors.append(f"release batches contain {duplicated} duplicate SHA memberships")
    return batches, errors


def validate(
    repo: Path,
    matrix_path: Path,
    patch_path: Path,
    report_en: Path,
    refs: FrozenRefs = DEFAULT_REFS,
) -> tuple[list[str], dict[str, object]]:
    errors: list[str] = []
    batches, ancestry_errors = expected_batches(repo, refs)
    errors.extend(ancestry_errors)
    expected = set(commit_list(repo, f"{refs.base}..{refs.target}"))
    ios_expected = set(commit_list(repo, f"{refs.ios_base}..{refs.ios_target}"))
    rows = csv_rows(matrix_path, REQUIRED_MATRIX_COLUMNS, "matrix", errors)
    patch_rows = csv_rows(patch_path, REQUIRED_PATCH_COLUMNS, "iOS patch map", errors)

    sha_counts = Counter(row.get("sha", "") for row in rows)
    for sha, count in sha_counts.items():
        if not FULL_SHA.fullmatch(sha):
            errors.append(f"matrix has malformed full SHA: {sha!r}")
        if count > 1:
            errors.append(f"matrix has duplicate upstream SHA {sha} ({count} rows)")
    actual = set(sha_counts) - {""}
    for sha in sorted(expected - actual):
        errors.append(f"matrix is missing upstream SHA {sha}")
    for sha in sorted(actual - expected):
        errors.append(f"matrix contains unexpected upstream SHA {sha}")

    area_counts: Counter[str] = Counter()
    priority_counts: Counter[str] = Counter()
    batch_counts: Counter[str] = Counter()
    unresolved: list[str] = []
    batch_for_sha = {sha: label for label, shas in batches.items() for sha in shas}
    for line, row in enumerate(rows, start=2):
        sha = row.get("sha", "")
        for field in REQUIRED_MATRIX_COLUMNS:
            if not row.get(field, "").strip() and field != "ios_patch_ids":
                errors.append(f"matrix row {line} ({sha or 'blank SHA'}) has empty required field {field}")
        commit_type = row.get("commit_type", "")
        if commit_type not in {"Normal", "merge"}:
            errors.append(f"matrix row {line} has invalid commit_type {commit_type!r}")
        batch = row.get("release_batch", "")
        if batch not in ALLOWED_BATCHES:
            errors.append(f"matrix row {line} has invalid release_batch {batch!r}")
        elif sha in expected:
            batch_counts[batch] += 1
            if batch != batch_for_sha.get(sha):
                errors.append(
                    f"matrix row {line} assigns {sha} to {batch}; frozen ancestry assigns {batch_for_sha.get(sha)}"
                )
        areas = [value.strip() for value in row.get("areas", "").split(";") if value.strip()]
        if not areas:
            errors.append(f"matrix row {line} has no subsystem area")
        for area in areas:
            if area not in ALLOWED_AREAS:
                errors.append(f"matrix row {line} has invalid area {area!r}")
            area_counts[area] += 1
        change_type = row.get("change_type", "")
        if change_type not in ALLOWED_TYPES:
            errors.append(f"matrix row {line} has invalid change_type {change_type!r}")
        priority = row.get("priority", "")
        if priority not in ALLOWED_PRIORITIES:
            errors.append(f"matrix row {line} has invalid priority {priority!r}")
        priority_counts[priority] += 1
        applicability = row.get("ios_applicability", "")
        if applicability not in ALLOWED_APPLICABILITY:
            errors.append(f"matrix row {line} has invalid ios_applicability {applicability!r}")
        review = row.get("review_status", "")
        if review not in ALLOWED_REVIEW:
            errors.append(f"matrix row {line} has invalid review_status {review!r}")
        if review == "unresolved":
            unresolved.append(sha)

    for batch in sorted(ALLOWED_BATCHES):
        if batch_counts[batch] != len(batches.get(batch, set())):
            errors.append(
                f"matrix release batch {batch} has {batch_counts[batch]} rows; expected {len(batches.get(batch, set()))}"
            )

    patch_counts = Counter(row.get("ios_commit_sha", "") for row in patch_rows)
    for sha, count in patch_counts.items():
        if not FULL_SHA.fullmatch(sha):
            errors.append(f"iOS patch map has malformed full SHA: {sha!r}")
        if count > 1:
            errors.append(f"iOS patch map has duplicate historical iOS SHA {sha} ({count} rows)")
    patch_shas = set(patch_counts) - {""}
    for sha in sorted(ios_expected - patch_shas):
        errors.append(f"iOS patch map is missing historical iOS SHA {sha}")
    for sha in sorted(patch_shas - ios_expected):
        errors.append(f"iOS patch map contains unexpected historical iOS SHA {sha}")
    defined_ids = {row.get("logical_patch_id", "").strip() for row in patch_rows if row.get("logical_patch_id", "").strip()}
    for line, row in enumerate(patch_rows, start=2):
        for field in REQUIRED_PATCH_COLUMNS:
            if not row.get(field, "").strip():
                errors.append(f"iOS patch map row {line} has empty required field {field}")
    for line, row in enumerate(rows, start=2):
        refs_in_row = [value.strip() for value in row.get("ios_patch_ids", "").split(";") if value.strip() and value.strip() != "none"]
        for patch_id in refs_in_row:
            if patch_id not in defined_ids:
                errors.append(f"matrix row {line} references undefined iOS logical patch ID {patch_id!r}")

    if not report_en.is_file():
        errors.append(f"English audit report is missing: {report_en}")
    else:
        report = report_en.read_text(encoding="utf-8")
        unresolved_match = re.search(r"Unresolved audit rows:\s*(\d+)", report)
        if unresolved_match is None:
            errors.append("English audit report does not report the unresolved audit row count")
        elif int(unresolved_match.group(1)) != len(unresolved):
            errors.append(
                "English audit report unresolved count is "
                f"{unresolved_match.group(1)}; matrix has {len(unresolved)}"
            )
        for sha in unresolved:
            if sha not in report:
                errors.append(f"English audit report does not identify unresolved SHA {sha}")

    summary: dict[str, object] = {
        "expected_upstream": len(expected),
        "matrix_rows": len(rows),
        "covered_upstream": len(expected & actual),
        "expected_ios": len(ios_expected),
        "patch_rows": len(patch_rows),
        "covered_ios": len(ios_expected & patch_shas),
        "batches": {batch: batch_counts[batch] for batch in ("P004", "P005", "P006")},
        "priorities": {key: priority_counts[key] for key in ("P0", "P1", "P2", "none")},
        "areas": dict(sorted(area_counts.items())),
        "unresolved": unresolved,
    }
    return errors, summary


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, default=Path(__file__).resolve().parents[2])
    parser.add_argument("--matrix", type=Path)
    parser.add_argument("--patch-map", type=Path)
    parser.add_argument("--report-en", type=Path)
    args = parser.parse_args(argv)
    repo = args.repo.resolve()
    matrix = args.matrix or repo / "docs/upstream-v021-change-matrix.csv"
    patch_map = args.patch_map or repo / "docs/upstream-v021-ios-patch-map.csv"
    report_en = args.report_en or repo / "docs/upstream-v021-change-matrix.md"
    try:
        errors, summary = validate(repo, matrix, patch_map, report_en)
    except (OSError, ValueError) as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        return 1
    print("Frozen upstream v0.21 matrix validation")
    print(f"Upstream commits: {summary['covered_upstream']}/{summary['expected_upstream']}")
    print(f"Historical iOS commits: {summary['covered_ios']}/{summary['expected_ios']}")
    print("Release batches: " + ", ".join(f"{k}={v}" for k, v in summary["batches"].items()))
    print("Priorities: " + ", ".join(f"{k}={v}" for k, v in summary["priorities"].items()))
    print("Subsystem rows: " + ", ".join(f"{k}={v}" for k, v in summary["areas"].items()))
    print(f"Unresolved audit rows: {len(summary['unresolved'])}")
    if errors:
        print(f"Validation failures: {len(errors)}", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1
    print("Validation: PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
