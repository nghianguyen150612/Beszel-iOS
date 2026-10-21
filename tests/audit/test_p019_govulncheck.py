import importlib.util
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts/audit/validate-p019-govulncheck.py"
spec = importlib.util.spec_from_file_location("p019_govulncheck", SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


def message(**kwargs):
    import json

    return json.dumps(kwargs, indent=2)


class P019GovulncheckTests(unittest.TestCase):
    def report(self, findings=""):
        return "\n".join([
            message(config={"protocol_version": "v1.0.0", "scanner_version": "govulncheck v1.9.0", "db": "https://vuln.go.dev", "db_last_modified": "2026-10-08T00:00:00Z", "go_version": "go1.27.2", "scan_mode": "source", "scan_level": "symbol"}),
            message(SBOM={"go_version": "go1.27.2", "roots": ["github.com/henrygd/beszel/internal/cmd/hub"]}),
            message(osv={"id": "GO-2026-6608"}),
            findings,
        ]).replace("\n\n", "\n")

    def test_fixed_target_is_reported_only_as_database_metadata(self):
        result = module.summarize(self.report(message(finding={"osv": "GO-2026-6604", "trace": [{"module": "stdlib", "package": "os", "function": "Example"}]})), "go1.27.2")
        self.assertFalse(result["target_advisory_in_findings"])
        self.assertEqual(result["finding_ids"], ["GO-2026-6604"])
        self.assertEqual(result["database_last_modified"], "2026-10-08T00:00:00Z")

    def test_target_finding_fails(self):
        report = self.report(message(finding={"osv": "GO-2026-6608"}))
        with self.assertRaisesRegex(ValueError, "GO-2026-6608 remains"):
            module.summarize(report, "go1.27.2")

    def test_wrong_toolchain_or_scan_mode_fails(self):
        with self.assertRaisesRegex(ValueError, "expected go1.27.2"):
            module.summarize(self.report().replace("go1.27.2", "go1.27.1"), "go1.27.2")
        with self.assertRaisesRegex(ValueError, "symbol-level source scan"):
            module.summarize(self.report().replace('"scan_mode": "source"', '"scan_mode": "binary"'), "go1.27.2")

    def test_malformed_or_incomplete_json_fails(self):
        with self.assertRaisesRegex(ValueError, "invalid govulncheck JSON"):
            module.summarize("{bad json", "go1.27.2")
        with self.assertRaisesRegex(ValueError, "lacks one config"):
            module.summarize(message(progress={"message": "done"}), "go1.27.2")


if __name__ == "__main__":
    unittest.main()
