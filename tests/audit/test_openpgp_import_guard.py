"""The OpenPGP import guard catches build-tagged imports without false positives from comments."""

import json
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
GUARD = ROOT / ".github/scripts/check-openpgp-imports.go"


class OpenPGPImportGuardTests(unittest.TestCase):
    def run_guard(self, source):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "sample.go").write_text(source)
            result = subprocess.run(
                ["go", "run", str(GUARD), "--root", str(root), "--skip-deps"],
                cwd=ROOT,
                capture_output=True,
                text=True,
            )
        return result, json.loads(result.stdout)

    def test_flags_import_under_build_tag(self):
        result, evidence = self.run_guard(
            '//go:build special\npackage sample\nimport _ "golang.org/x/crypto/openpgp/armor"\n'
        )
        self.assertEqual(result.returncode, 1)
        self.assertEqual(evidence["status"], "FAIL")
        self.assertEqual(evidence["source_imports"][0]["path"], "golang.org/x/crypto/openpgp/armor")

    def test_ignores_comment_text(self):
        result, evidence = self.run_guard(
            '//go:build special\npackage sample\n// import "golang.org/x/crypto/openpgp"\n'
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(evidence["status"], "PASS")
        self.assertEqual(evidence["source_imports"], [])


if __name__ == "__main__":
    unittest.main()
