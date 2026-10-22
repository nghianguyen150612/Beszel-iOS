"""P023 JavaScript audit and lock-graph policy tests."""

import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts/audit/summarize-p023-js-audits.py"
spec = importlib.util.spec_from_file_location("p023_js_audits", SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class P023JavaScriptAuditTests(unittest.TestCase):
    def test_real_lockfiles_show_build_only_lingui_paths(self):
        package = json.loads((ROOT / "internal/site/package.json").read_text())
        bun = module.bun_graph(package, ROOT / "internal/site/bun.lock")
        npm = module.npm_graph(package, ROOT / "internal/site/package-lock.json")
        self.assertEqual(bun["version"], "3.0.3")
        self.assertEqual(bun["paths"], npm["paths"])
        self.assertTrue(npm["dev_dependency"])

    def test_npm_high_advisory_is_retained_even_with_nonzero_audit_exit(self):
        report = {
            "auditReportVersion": 2,
            "vulnerabilities": {
                "braces": {
                    "name": "braces",
                    "severity": "high",
                    "via": [{
                        "source": 93687,
                        "name": "braces",
                        "dependency": "braces",
                        "title": "stack exhaustion",
                        "url": "https://github.com/advisories/GHSA-vfj7-8cjw-p6xm",
                        "range": "<=3.0.3",
                    }],
                    "effects": [],
                    "range": "<=3.0.3",
                    "nodes": ["node_modules/braces"],
                    "fixAvailable": False,
                }
            },
            "metadata": {"vulnerabilities": {"info": 0, "low": 0, "moderate": 0, "high": 1, "critical": 0, "total": 1}},
        }
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "npm-audit.json"
            path.write_text(json.dumps(report))
            result = module.audit_report(path, "npm", 1)
        self.assertIn("GHSA-vfj7-8cjw-p6xm", result["advisory_ids"])
        self.assertEqual(result["severity"], "high")
        self.assertEqual(result["command_exit_code"], 1)

    def test_bun_advisory_record_is_retained(self):
        report = {
            "vulnerabilities": [{
                "name": "braces",
                "severity": "high",
                "url": "https://github.com/advisories/GHSA-vfj7-8cjw-p6xm",
                "range": "<=3.0.3",
            }],
            "metadata": {"vulnerabilities": {"high": 1}},
        }
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "bun-audit.json"
            path.write_text(json.dumps(report))
            result = module.audit_report(path, "bun", 1)
        self.assertEqual(result["advisory_ids"], ["GHSA-vfj7-8cjw-p6xm"])

    def test_rejects_an_unreviewed_advisory(self):
        report = {"vulnerabilities": [{
            "name": "braces",
            "severity": "high",
            "url": "https://github.com/advisories/GHSA-aaaa-bbbb-cccc",
        }]}
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "audit.json"
            path.write_text(json.dumps(report))
            with self.assertRaisesRegex(ValueError, "only the reviewed"):
                module.audit_report(path, "bun", 1)

    def test_rejects_critical_or_extra_severity_records(self):
        report = {"vulnerabilities": [{
            "name": "braces",
            "severity": "critical",
            "url": "https://github.com/advisories/GHSA-vfj7-8cjw-p6xm",
        }]}
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "audit.json"
            path.write_text(json.dumps(report))
            with self.assertRaisesRegex(ValueError, "unexpected severity"):
                module.audit_report(path, "bun", 1)

    def test_policy_keeps_frontend_finding_open_and_release_authorization_denied(self):
        policy = json.loads((ROOT / "docs/ios-p023-advisory-policy.json").read_text())
        result = module.validate_policy(policy)
        self.assertEqual(result["remediation_status"], "MITIGATED_WITH_RESIDUAL_RISK")
        self.assertIsNone(result["first_patched_version"])
        self.assertEqual(policy["release_authorization"], "NOT_GRANTED")


if __name__ == "__main__":
    unittest.main()
