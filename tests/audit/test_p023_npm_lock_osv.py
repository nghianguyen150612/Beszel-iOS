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
            "withdrawn": module.REVIEWED_WITHDRAWAL,
            "severity": [{"type": "CVSS_V3", "score": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H"}],
            "affected": [{
                "package": {"name": "braces", "ecosystem": "npm"},
                "ranges": [{"type": "SEMVER", "events": [{"introduced": "0"}, {"last_affected": "3.0.3"}]}],
            }],
        }
        self.response = [{"results": [{"vulns": []}]}]

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
        # npm 11 prunes the superseded transitive graph after the compatible
        # glob override; keep an exact count so the OSV scan covers every
        # install node in the reviewed lockfile.
        self.assertEqual(nodes, 481)
        self.assertEqual(skipped, 0)
        self.assertEqual(braces, [{"package": {"name": "braces", "ecosystem": "npm"}, "version": "3.0.3"}])

    def test_retains_exact_reviewed_finding_and_lock_provenance(self):
        summary, raw = self.scan()
        self.assertEqual(summary["status"], "PASS_NO_ACTIVE_ADVISORIES")
        self.assertEqual(summary["finding_ids"], [])
        self.assertEqual(summary["candidate_source_sha"], "a" * 40)
        self.assertEqual(summary["findings"], [])
        self.assertEqual(summary["verified_withdrawals"][0]["withdrawn"], module.REVIEWED_WITHDRAWAL)
        self.assertEqual(len(raw["queries"]), 1)

    def test_reappearing_or_unreviewed_findings_fail_closed(self):
        for vuln_id in (module.REVIEWED_ID, "GHSA-aaaa-bbbb-cccc"):
            with self.subTest(vuln_id=vuln_id), self.assertRaisesRegex(ValueError, "active or unreviewed"):
                self.scan(response=[{"results": [{"vulns": [{"id": vuln_id}]}]}])

    def test_missing_finding_requires_exact_withdrawal_timestamp(self):
        advisory = json.loads(json.dumps(self.advisory))
        advisory["withdrawn"] = "2026-10-24T00:00:00Z"
        with self.assertRaisesRegex(ValueError, "verified OSV withdrawal"):
            self.scan(advisory=advisory)

    def test_changed_affected_range_or_wrong_source_sha_fails_closed(self):
        advisory = json.loads(json.dumps(self.advisory))
        advisory["affected"][0]["ranges"][0]["events"][-1]["last_affected"] = "3.0.2"
        with self.assertRaisesRegex(ValueError, "previously reviewed braces@3.0.3"):
            self.scan(advisory=advisory)
        with self.assertRaisesRegex(ValueError, "full 40-character Git SHA"):
            self.scan(source_sha="short")

    def test_missing_exact_version_fails_closed(self):
        with self.assertRaisesRegex(ValueError, "no exact version"):
            module.lock_queries({"lockfileVersion": 3, "packages": {"": {}, "node_modules/braces": {"dev": True}}})


if __name__ == "__main__":
    unittest.main()
