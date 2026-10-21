import base64
import importlib.util
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts/audit/patch-go-x509-security.py"
PATCH = ROOT / "patches/go/1.27.2/GO-2026-6809.patch.b64"
spec = importlib.util.spec_from_file_location("p019_go_x509_backport", SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class P019GoX509BackportTests(unittest.TestCase):
    def test_patch_digest_is_the_reviewed_go_change(self):
        patch_bytes, digest = module.load_patch(PATCH)
        self.assertEqual(digest, module.PATCH_SHA256)
        self.assertIn(module.UPSTREAM_COMMIT.encode(), patch_bytes)

    def test_unknown_toolchain_version_fails_before_source_access(self):
        with self.assertRaisesRegex(ValueError, "unsupported Go version"):
            module.patch_source(Path("/not/a/real/goroot"), "go1.27.3", PATCH)

    def test_modified_patch_fails_before_source_access(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            bad_patch = directory / "bad.patch"
            bad_patch.write_bytes(base64.b64encode(b"not the reviewed patch"))
            with self.assertRaisesRegex(ValueError, "unexpected GO-2026-6809 patch digest"):
                module.patch_source(directory, "go1.27.2", bad_patch)


if __name__ == "__main__":
    unittest.main()
