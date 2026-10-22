"""P023 JavaScript audit and lock-graph policy tests."""

import importlib.util
import hashlib
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
    def npm_report(self):
        advisory = {
            "source": 1240992,
            "name": "braces",
            "dependency": "braces",
            "title": "stack exhaustion",
            "url": "https://github.com/advisories/GHSA-vfj7-8cjw-p6xm",
            "severity": "high",
            "range": "<=3.0.3",
        }
        vulnerabilities = {
            "@lingui/cli": {"name": "@lingui/cli", "severity": "high", "via": ["chokidar", "micromatch"], "nodes": ["node_modules/@lingui/cli"]},
            "@lingui/vite-plugin": {"name": "@lingui/vite-plugin", "severity": "high", "via": ["@lingui/cli"], "nodes": ["node_modules/@lingui/vite-plugin"]},
            "braces": {"name": "braces", "severity": "high", "via": [advisory], "nodes": ["node_modules/braces"]},
            "chokidar": {"name": "chokidar", "severity": "high", "via": ["braces"], "nodes": ["node_modules/chokidar"]},
            "micromatch": {"name": "micromatch", "severity": "high", "via": ["braces"], "nodes": ["node_modules/micromatch"]},
        }
        return {
            "auditReportVersion": 2,
            "vulnerabilities": vulnerabilities,
            "metadata": {"vulnerabilities": {"info": 0, "low": 0, "moderate": 0, "high": 5, "critical": 0, "total": 5}},
        }

    def test_real_lockfiles_show_build_only_lingui_paths(self):
        package = json.loads((ROOT / "internal/site/package.json").read_text())
        bun = module.bun_graph(package, ROOT / "internal/site/bun.lock")
        npm = module.npm_graph(package, ROOT / "internal/site/package-lock.json")
        self.assertEqual(bun["version"], "3.0.3")
        self.assertEqual(bun["paths"], npm["paths"])
        self.assertTrue(npm["dev_dependency"])

    def test_npm_high_advisory_is_retained_even_with_nonzero_audit_exit(self):
        report = self.npm_report()
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "npm-audit.json"
            path.write_text(json.dumps(report))
            result = module.audit_report(path, "npm", 1)
        self.assertIn("GHSA-vfj7-8cjw-p6xm", result["advisory_ids"])
        self.assertEqual(result["severity"], "high")
        self.assertEqual(result["command_exit_code"], 1)
        self.assertEqual(result["vulnerable_package_count"], 5)
        self.assertEqual(result["root_advisory_count"], 1)

    def test_npm_audit_rejects_a_new_package_in_the_vulnerability_tree(self):
        report = self.npm_report()
        report["vulnerabilities"]["unexpected"] = {
            "name": "unexpected", "severity": "high", "via": ["braces"], "nodes": ["node_modules/unexpected"]
        }
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "npm-audit.json"
            path.write_text(json.dumps(report))
            with self.assertRaisesRegex(ValueError, "five-package Lingui/braces chain"):
                module.audit_report(path, "npm", 1)

    def test_npm_registry_timeout_is_retained_as_unavailable(self):
        report = {
            "message": "network timeout at: https://registry.npmjs.org/-/npm/v1/security/audits/quick",
            "error": {"summary": "", "detail": ""},
        }
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "npm-audit.json"
            path.write_text(json.dumps(report))
            result = module.audit_report(path, "npm", 1)
        self.assertEqual(result["status"], "UNAVAILABLE_NETWORK")
        self.assertTrue(result["raw_error_retained"])
        self.assertIn("registry.npmjs.org", result["error_message"])

    def test_unrelated_npm_errors_do_not_activate_the_fallback(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "npm-audit.json"
            path.write_text(json.dumps({"message": "invalid lockfile", "error": {}}))
            with self.assertRaisesRegex(ValueError, "error payload"):
                module.audit_report(path, "npm", 1)

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
        self.assertEqual(policy["npm_audit_fallback"]["new_or_missing_findings_action"], "FAIL")

    def test_full_lock_osv_fallback_is_source_and_lock_bound(self):
        lock = ROOT / "internal/site/package-lock.json"
        report = {
            "schema": 1,
            "status": "PASS_WITH_REVIEWED_ADVISORY",
            "candidate_source_sha": "a" * 40,
            "endpoint": "https://api.osv.dev/v1/querybatch",
            "package_lock_sha256": hashlib.sha256(lock.read_bytes()).hexdigest(),
            "package_lock_nodes_scanned": 510,
            "unique_package_version_queries": 494,
            "finding_ids": ["GHSA-vfj7-8cjw-p6xm"],
            "findings": [{
                "advisory_id": "GHSA-vfj7-8cjw-p6xm",
                "cve_id": "CVE-2026-93687",
                "package": "braces",
                "version": "3.0.3",
                "install_nodes": [{"path": "node_modules/braces", "dev": True}],
                "first_patched_version": None,
                "raw_scanner_status": "retained by full npm lock OSV query",
                "affected_events": [{"introduced": "0"}, {"last_affected": "3.0.3"}],
            }],
        }
        result = module.validate_osv_lock_scan(report, lock, "a" * 40)
        self.assertEqual(result["status"], "PASS_WITH_REVIEWED_ADVISORY")
        report["candidate_source_sha"] = "b" * 40
        with self.assertRaisesRegex(ValueError, "current candidate source commit"):
            module.validate_osv_lock_scan(report, lock, "a" * 40)


if __name__ == "__main__":
    unittest.main()
