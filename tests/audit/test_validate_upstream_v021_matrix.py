#!/usr/bin/env python3
"""Synthetic Git fixtures for the frozen upstream matrix validator."""

from __future__ import annotations

import csv
import importlib.util
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
VALIDATOR_PATH = ROOT / "scripts/audit/validate-upstream-v021-matrix.py"
SPEC = importlib.util.spec_from_file_location("matrix_validator", VALIDATOR_PATH)
assert SPEC and SPEC.loader
validator = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = validator
SPEC.loader.exec_module(validator)


def run_git(repo: Path, *args: str) -> str:
    return subprocess.check_output(
        ["git", "-C", str(repo), *args], text=True, stderr=subprocess.PIPE
    ).strip()


def write_csv(path: Path, fields: list[str], rows: list[dict[str, str]]) -> None:
    with path.open("w", newline="", encoding="utf-8") as stream:
        writer = csv.DictWriter(stream, fieldnames=fields)
        writer.writeheader()
        writer.writerows(rows)


class MatrixValidatorTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory(prefix="p002-matrix-fixture-")
        self.repo = Path(self.temp.name) / "repo"
        self.repo.mkdir()
        run_git(self.repo, "init", "-q")
        run_git(self.repo, "config", "user.name", "Audit Fixture")
        run_git(self.repo, "config", "user.email", "audit-fixture@example.invalid")
        self.base = self.commit("base.txt", "base\n", "fixture base")
        self.u1 = self.commit("u1.txt", "one\n", "upstream one")
        self.v020 = self.commit("u2.txt", "two\n", "v0.20 boundary")
        self.u3 = self.commit("u3.txt", "three\n", "upstream three")
        self.v021 = self.commit("u4.txt", "four\n", "v0.21 boundary")
        self.target = self.commit("u5.txt", "five\n", "frozen target")
        run_git(self.repo, "switch", "-q", "-c", "ios-fixture", self.base)
        self.i1 = self.commit("ios1.txt", "ios one\n", "historical iOS one")
        self.ios_target = self.commit("ios2.txt", "ios two\n", "P001 iOS snapshot")
        self.refs = validator.FrozenRefs(
            base=self.base,
            target=self.target,
            v020=self.v020,
            v021=self.v021,
            ios_base=self.base,
            ios_target=self.ios_target,
        )
        self.matrix_path = self.repo / "matrix.csv"
        self.patch_path = self.repo / "patches.csv"
        self.report_path = self.repo / "report.md"
        self.report_path.write_text("Unresolved audit rows: 0\n", encoding="utf-8")
        self.write_inputs()

    def tearDown(self) -> None:
        self.temp.cleanup()

    def commit(self, filename: str, content: str, message: str) -> str:
        path = self.repo / filename
        path.write_text(content, encoding="utf-8")
        run_git(self.repo, "add", filename)
        run_git(self.repo, "commit", "-q", "-m", message)
        return run_git(self.repo, "rev-parse", "HEAD")

    def write_inputs(self) -> None:
        shas = [self.u1, self.v020, self.u3, self.v021, self.target]
        labels = ["P004", "P004", "P005", "P005", "P006"]
        self.matrix_rows = []
        for index, (sha, label) in enumerate(zip(shas, labels)):
            self.matrix_rows.append({
                "sha": sha,
                "subject": f"fixture upstream commit {index + 1}",
                "commit_type": "Normal",
                "release_batch": label,
                "areas": "Agent;Agent:Disk I/O",
                "change_type": "fix",
                "priority": "P1",
                "ios_applicability": "shared",
                "ios_patch_ids": "PATCH-X" if index == 0 else "none",
                "integration_prompts": "P007",
                "risk_summary": "Fixture reviewed patch with path evidence.",
                "evidence_paths": f"u{index + 1}.txt",
                "review_status": "Reviewed",
            })
        self.patch_rows = []
        for sha, number in ((self.i1, 1), (self.ios_target, 2)):
            self.patch_rows.append({
                "ios_commit_sha": sha,
                "logical_patch_id": "PATCH-X",
                "changed_files": f"ios{number}.txt",
                "implementation_purpose": "Fixture iOS behavior.",
                "upstream_files_or_apis": "fixture API",
                "upstream_commits_at_risk": "none",
                "preservation_action": "Preserve fixture behavior.",
                "responsible_prompt": "P007",
                "risk_category": "fixture",
                "review_status": "Reviewed",
            })
        self.write_rows()

    def write_rows(self) -> None:
        write_csv(self.matrix_path, list(validator.REQUIRED_MATRIX_COLUMNS), self.matrix_rows)
        write_csv(self.patch_path, list(validator.REQUIRED_PATCH_COLUMNS), self.patch_rows)

    def run_validation(self) -> list[str]:
        errors, _ = validator.validate(
            self.repo,
            self.matrix_path,
            self.patch_path,
            self.report_path,
            self.refs,
        )
        return errors

    def test_valid_synthetic_history_passes(self) -> None:
        self.assertEqual([], self.run_validation())

    def test_missing_upstream_row_is_rejected(self) -> None:
        self.matrix_rows.pop()
        self.write_rows()
        self.assertTrue(any("missing upstream SHA" in error for error in self.run_validation()))

    def test_duplicate_upstream_sha_is_rejected(self) -> None:
        self.matrix_rows.append(dict(self.matrix_rows[0]))
        self.write_rows()
        self.assertTrue(any("duplicate upstream SHA" in error for error in self.run_validation()))

    def test_invalid_category_is_rejected(self) -> None:
        self.matrix_rows[0]["areas"] = "Agent;Bogus"
        self.write_rows()
        self.assertTrue(any("invalid area" in error for error in self.run_validation()))

    def test_unknown_patch_reference_is_rejected(self) -> None:
        self.matrix_rows[0]["ios_patch_ids"] = "PATCH-UNKNOWN"
        self.write_rows()
        self.assertTrue(any("undefined iOS logical patch ID" in error for error in self.run_validation()))

    def test_missing_historical_ios_commit_is_rejected(self) -> None:
        self.patch_rows.pop()
        self.write_rows()
        self.assertTrue(any("missing historical iOS SHA" in error for error in self.run_validation()))

    def test_incorrect_release_batch_is_rejected(self) -> None:
        self.matrix_rows[0]["release_batch"] = "P005"
        self.write_rows()
        self.assertTrue(any("frozen ancestry assigns P004" in error for error in self.run_validation()))


if __name__ == "__main__":
    unittest.main(verbosity=2)
