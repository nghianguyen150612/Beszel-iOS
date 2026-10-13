import csv
import hashlib
import importlib.util
import shutil
import tempfile
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPT = REPO_ROOT / "scripts/audit/validate-upstream-v021-readiness.py"
SPEC = importlib.util.spec_from_file_location("readiness_validator", SCRIPT)
readiness_validator = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(readiness_validator)


INPUTS = (
    "upstream-v021-change-matrix.csv",
    "upstream-v021-ios-patch-map.csv",
    "upstream-v021-implementation-crosswalk.csv",
    "upstream-v021-risk-register.csv",
    "upstream-v021-compatibility-matrix.csv",
    "upstream-v021-acceptance.md",
    "upstream-v021-acceptance.vi.md",
)


class ReadinessValidatorTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name) / "repo"
        docs = self.root / "docs"
        docs.mkdir(parents=True)
        for name in INPUTS:
            shutil.copy2(REPO_ROOT / "docs" / name, docs / name)

    def tearDown(self):
        self.temp.cleanup()

    def read_rows(self, filename):
        path = self.root / "docs" / filename
        with path.open(newline="", encoding="utf-8-sig") as stream:
            reader = csv.DictReader(stream)
            fields = list(reader.fieldnames or [])
            return list(reader), fields

    def write_rows(self, filename, rows, fields):
        path = self.root / "docs" / filename
        with path.open("w", newline="", encoding="utf-8") as stream:
            writer = csv.DictWriter(stream, fieldnames=fields, lineterminator="\n")
            writer.writeheader()
            writer.writerows(rows)

    def test_current_package_is_structurally_valid(self):
        errors, counts = readiness_validator.validate(self.root)
        self.assertEqual(errors, [])
        self.assertEqual(counts["upstream_commits"], 178)
        self.assertEqual(counts["p0_risks"], 21)
        self.assertEqual(counts["logical_ios_patches"], 13)

    def test_missing_p0_risk_coverage_fails(self):
        rows, fields = self.read_rows("upstream-v021-risk-register.csv")
        rows = [row for row in rows if row["risk_id"] != "P0-01"]
        self.write_rows("upstream-v021-risk-register.csv", rows, fields)
        errors, _ = readiness_validator.validate(self.root)
        self.assertTrue(any("P0 coverage is 20/21" in error for error in errors))

    def test_missing_logical_patch_mapping_fails(self):
        rows, fields = self.read_rows("upstream-v021-implementation-crosswalk.csv")
        rows = [row for row in rows if row["mapping_id"] != "PATCH-IOS-A7-RUNTIME"]
        self.write_rows("upstream-v021-implementation-crosswalk.csv", rows, fields)
        errors, _ = readiness_validator.validate(self.root)
        self.assertTrue(any("missing logical iOS patches" in error for error in errors))

    def test_invalid_owner_fails(self):
        rows, fields = self.read_rows("upstream-v021-compatibility-matrix.csv")
        rows[0]["owning_prompt"] = "P019"
        self.write_rows("upstream-v021-compatibility-matrix.csv", rows, fields)
        errors, _ = readiness_validator.validate(self.root)
        self.assertTrue(any("invalid owner prompt 'P019'" in error for error in errors))

    def test_duplicate_stable_id_fails(self):
        rows, fields = self.read_rows("upstream-v021-compatibility-matrix.csv")
        rows.append(dict(rows[0]))
        self.write_rows("upstream-v021-compatibility-matrix.csv", rows, fields)
        errors, _ = readiness_validator.validate(self.root)
        self.assertTrue(any("duplicate scenario_id" in error for error in errors))

    def test_malformed_reference_fails(self):
        rows, fields = self.read_rows("upstream-v021-risk-register.csv")
        rows[0]["relevant_upstream_shas"] = "not-a-sha"
        self.write_rows("upstream-v021-risk-register.csv", rows, fields)
        errors, _ = readiness_validator.validate(self.root)
        self.assertTrue(any("malformed upstream SHA reference" in error for error in errors))

    def test_validation_does_not_modify_inputs(self):
        def snapshot():
            result = {}
            for path in sorted((self.root / "docs").iterdir()):
                result[path.name] = hashlib.sha256(path.read_bytes()).hexdigest()
            return result

        before = snapshot()
        readiness_validator.validate(self.root)
        self.assertEqual(snapshot(), before)


if __name__ == "__main__":
    unittest.main()
