"""Fail-closed P023 Go advisory policy tests."""

import importlib.util
import json
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts/audit/validate-p023-govulncheck.py"
spec = importlib.util.spec_from_file_location("p023_govulncheck", SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


def message(**values):
    return json.dumps(values)


class P023GoVulncheckTests(unittest.TestCase):
    def setUp(self):
        self.config = {
            "config": {
                "scanner_version": "govulncheck v1.9.0",
                "db": "https://vuln.go.dev",
                "db_last_modified": "2026-11-12T00:00:00Z",
                "go_version": "go1.27.4",
                "scan_mode": "source",
                "scan_level": "symbol",
            }
        }
        self.sbom = {"SBOM": {"roots": [
            "github.com/henrygd/beszel/internal/cmd/agent",
            "github.com/henrygd/beszel/internal/cmd/hub",
        ]}}
        self.x509 = {
            "status": "PASS",
            "advisory": "GO-2026-6809",
            "go_version": "go1.27.4",
            "fix_disposition": "native in official Go 1.27.4; custom P019/P022 backport not applied",
            "native_source_sha256": module.EXPECTED_X509_SOURCE_SHA256,
            "matches_reviewed_backport_outputs": True,
            "standard_library_test": {"status": "PASS"},
        }
        self.openpgp = {
            "schema": 1,
            "status": "PASS",
            "source_files_scanned": 377,
            "source_imports": [],
            "dependency_profiles": [
                {"name": name, "openpgp_packages": [], "package_count": 1}
                for name in sorted(module.EXPECTED_PROFILES)
            ],
        }
        self.policy = json.loads(module.POLICY_PATH.read_text())

    def report(self, *findings):
        return "\n".join([message(**self.config), message(**self.sbom), *(message(finding=f) for f in findings)])

    def test_clean_go_1274_scan_and_fixed_x_crypto_pass(self):
        result = module.summarize(self.report(), "go1.27.4", self.x509, self.openpgp, self.policy)
        self.assertEqual(result["finding_ids"], [])
        self.assertEqual(result["security_dispositions"][0]["remediation_status"], "REMEDIATED_AND_VERIFIED")
        go6608 = next(item for item in result["security_dispositions"] if item["advisory"] == "GO-2026-6608")
        self.assertEqual(go6608["remediation_status"], "REMEDIATED_AND_VERIFIED")
        go5932 = next(item for item in result["security_dispositions"] if item["advisory"] == "GO-2026-5932")
        self.assertEqual(go5932["remediation_status"], "REMEDIATED_BY_FIXED_MODULE_VERSION")

    def test_rejects_a_go_5932_finding_at_the_fixed_module_version(self):
        finding = {"osv": "GO-2026-5932", "trace": [{"module": "golang.org/x/crypto", "version": "v0.59.0"}]}
        with self.assertRaisesRegex(ValueError, "GO-2026-5932 reappeared"):
            module.summarize(self.report(finding), "go1.27.4", self.x509, self.openpgp, self.policy)

    def test_requires_full_candidate_source_sha(self):
        with self.assertRaisesRegex(ValueError, "full 40-character Git SHA"):
            module.summarize(self.report(), "go1.27.4", self.x509, self.openpgp, self.policy, "short")

    def test_rejects_openpgp_symbol_trace_even_when_fixed_version_is_reported(self):
        finding = {"osv": "GO-2026-5932", "trace": [{"module": "golang.org/x/crypto", "version": "v0.59.0", "package": "golang.org/x/crypto/openpgp", "function": "openpgp.ReadMessage"}]}
        with self.assertRaisesRegex(ValueError, "GO-2026-5932 reappeared"):
            module.summarize(self.report(finding), "go1.27.4", self.x509, self.openpgp, self.policy)

    def test_rejects_remediated_standard_library_advisories_returning(self):
        for advisory in ("GO-2026-6809", "GO-2026-6608", "GO-2026-6859"):
            with self.subTest(advisory=advisory), self.assertRaises(ValueError):
                module.summarize(self.report({"osv": advisory, "trace": []}), "go1.27.4", self.x509, self.openpgp, self.policy)

    def test_rejects_new_unreviewed_advisory(self):
        with self.assertRaisesRegex(ValueError, "unreviewed Go advisory"):
            module.summarize(self.report({"osv": "GO-2099-0001", "trace": []}), "go1.27.4", self.x509, self.openpgp, self.policy)

    def test_rejects_openpgp_import_evidence_failure(self):
        self.openpgp["source_imports"] = [{"path": "golang.org/x/crypto/openpgp"}]
        with self.assertRaisesRegex(ValueError, "an OpenPGP import"):
            module.summarize(self.report(), "go1.27.4", self.x509, self.openpgp, self.policy)


if __name__ == "__main__":
    unittest.main()
