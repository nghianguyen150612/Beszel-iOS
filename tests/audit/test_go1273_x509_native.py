"""Tests for the fail-closed Go 1.27.3 native x509 source validator."""

import importlib.util
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts/audit/validate-go-x509-native.py"
spec = importlib.util.spec_from_file_location("x509_native", SCRIPT)
x509 = importlib.util.module_from_spec(spec)
spec.loader.exec_module(x509)


class NativeX509Tests(unittest.TestCase):
    def test_pins_native_source_to_reviewed_go_release_outputs(self):
        self.assertEqual(x509.EXPECTED_GO, "go1.27.3")
        self.assertEqual(x509.SOURCE_SHA256, {
            "src/crypto/x509/root_darwin.go": "69b350fe1afa679935ca831c3beea9833d59d8cd62c6f0b4d02f14526dd46ac1",
            "src/crypto/x509/root_windows.go": "ba910c270b9d096f50ccbf9426c03410ad16bc46a86e0ba4237d27c3381d28d0",
            "src/crypto/x509/verify.go": "a239364c8fd240db693be8ec4f82e84958d9b4e4f265ff3923499e2161a34d86",
        })

    def test_rejects_any_other_go_version(self):
        with tempfile.TemporaryDirectory() as tmp, self.assertRaisesRegex(ValueError, "expected go1.27.3"):
            x509.verify_sources(Path(tmp), "go1.27.2")

    def test_rejects_missing_or_unknown_sources(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for relative in x509.SOURCE_SHA256:
                path = root / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("unknown source\n")
            with self.assertRaisesRegex(ValueError, "unknown Go 1.27.3 x509 source"):
                x509.verify_sources(root, "go1.27.3")


if __name__ == "__main__":
    unittest.main()
