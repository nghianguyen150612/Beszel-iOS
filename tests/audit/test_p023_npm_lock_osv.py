"""P023 full npm-lock OSV fallback tests."""

import importlib.util
import json
from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts/audit/scan-p023-npm-lock-osv.py"
spec = importlib.util.spec_from_file_location("p023_npm_lock_osv", SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class P023NpmLockOsvTests(unittest.TestCase):
    def setUp(self):
        self.lock = {
            "lockfileVersion": 3,
            "packages": {
                "": {"name": "beszel", "version": "0.21.0"},
                "node_modules/braces": {"version": "3.0.3", "dev": True},
            },
        }
        self.advisory = {
            "id": module.REVIEWED_ID,
            "aliases": [module.REVIEWED_ALIAS],
            "severity": [{"type": "CVSS_V3", "score": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H"}],
            "affected": [{
                "package": {"name": "braces", "ecosystem": "npm"},
                "ranges": [{"type": "SEMVER", "events": [{"introduced": "0"}, {"last_affected": "3.0.3"}]}],
            }],
        }
        self.response = [{"results": [{"vulns": [{"id": module.REVIEWED_ID, "modified": "2026-10-02T22:45:04Z"}]}]}]

    def scan(self, response=None, advisory=None, source_sha="a" * 40):
        return module.scan(
            json.dumps(self.lock).encode(), source_sha,
            self.response if response is None else response,
            self.advisory if advisory is None else advisory,
        )

    def test_current_npm_lock_has_build_only_braces_path(self):
        lock = json.loads((ROOT / "internal/site/package-lock.json").read_text())
        queries, nodes, skipped = module.lock_queries(lock)
        braces = [q for q in queries if q["package"]["name"] == "braces"]
        self.assertEqual(nodes, 510)
        self.assertEqual(skipped, 0)
        self.assertEqual(braces, [{"package": {"name": "braces", "ecosystem": "npm"}, "version": "3.0.3"}])

    def test_retains_exact_reviewed_finding_and_lock_provenance(self):
        summary, raw = self.scan()
        self.assertEqual(summary["status"], "PASS_WITH_REVIEWED_ADVISORY")
        self.assertEqual(summary["finding_ids"], [module.REVIEWED_ID])
        self.assertEqual(summary["candidate_source_sha"], "a" * 40)
        self.assertEqual(summary["findings"][0]["install_nodes"], [{"path": "node_modules/braces", "dev": True}])
        self.assertEqual(len(raw["queries"]), 1)

    def test_unreviewed_or_missing_findings_fail_closed(self):
        for vuln_id in ("GHSA-aaaa-bbbb-cccc", None):
            with self.subTest(vuln_id=vuln_id), self.assertRaises(ValueError):
                response = [{"results": [{"vulns": [] if vuln_id is None else [{"id": vuln_id}]}]}]
                self.scan(response=response)

    def test_new_fix_or_wrong_source_sha_requires_reassessment(self):
        with self.assertRaisesRegex(ValueError, "fixed version"):
            advisory = json.loads(json.dumps(self.advisory))
            advisory["affected"][0]["ranges"][0]["events"].append({"fixed": "3.0.4"})
            self.scan(advisory=advisory)
        with self.assertRaisesRegex(ValueError, "full 40-character Git SHA"):
            self.scan(source_sha="short")

    def test_missing_exact_version_fails_closed(self):
        with self.assertRaisesRegex(ValueError, "no exact version"):
            module.lock_queries({"lockfileVersion": 3, "packages": {"": {}, "node_modules/braces": {"dev": True}}})


if __name__ == "__main__":
    unittest.main()
