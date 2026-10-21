import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts/audit/summarize-p019-js-audits.py"
spec = importlib.util.spec_from_file_location("p019_js_audit", SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class P019JavaScriptAuditTests(unittest.TestCase):
    def write_report(self, root, name, data):
        path = root / name
        path.write_text(json.dumps(data))
        return path

    def test_high_build_advisory_is_recorded_without_hiding_it(self):
        with tempfile.TemporaryDirectory() as temp:
            report = self.write_report(Path(temp), "audit.json", {
                "metadata": {"vulnerabilities": {"info": 0, "low": 0, "moderate": 0, "high": 1, "critical": 0}},
                "vulnerabilities": {"braces": {"severity": "high", "via": [{"url": "https://github.com/advisories/GHSA-vfj7-8cjw-p6xm"}]}},
            })
            result = module.audit(report, "bun", 0)
            self.assertIn("GHSA-vfj7-8cjw-p6xm", result["advisory_ids"])
            self.assertTrue(result["known_build_advisory_open_in_report"])
            self.assertEqual(result["critical_finding_count"], 0)

    def test_command_errors_and_critical_findings_fail(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            report = self.write_report(root, "audit.json", {"vulnerabilities": []})
            with self.assertRaisesRegex(ValueError, "exited 1"):
                module.audit(report, "npm", 1)
            critical = self.write_report(root, "critical.json", {"advisories": {"1": {"severity": "critical"}}})
            with self.assertRaisesRegex(ValueError, "critical findings"):
                module.audit(critical, "npm", 0)

    def test_invalid_json_fails(self):
        with tempfile.TemporaryDirectory() as temp:
            report = Path(temp) / "invalid.json"
            report.write_text("not json")
            with self.assertRaisesRegex(ValueError, "valid JSON"):
                module.audit(report, "bun", 0)


if __name__ == "__main__":
    unittest.main()
