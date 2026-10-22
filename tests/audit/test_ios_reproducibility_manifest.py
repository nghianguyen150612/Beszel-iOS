"""Deterministic tests for the native build reproducibility report."""

import hashlib
import importlib.util
from pathlib import Path
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location(
    "ios_reproducibility_manifest", ROOT / ".github/scripts/create-ios-reproducibility-manifest.py")
manifest_module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(manifest_module)


class ReproducibilityManifestTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.dist = self.root / "internal/site/dist"
        (self.dist / "assets").mkdir(parents=True)
        (self.dist / "index.html").write_text('<script src="./assets/app.js"></script><link rel="stylesheet" href="./assets/app.css">')
        (self.dist / "assets/app.js").write_text("console.log('same output');")
        (self.dist / "assets/app.css").write_text("body { color: black; }")
        self.lock = self.root / "internal/site/bun.lock"
        self.lock.parent.mkdir(parents=True, exist_ok=True)
        self.lock.write_text("frozen lockfile")
        self.workflow = self.root / ".github/workflows/ios-build.yml"
        self.workflow.parent.mkdir(parents=True)
        self.workflow.write_text("pinned build workflow")
        self.patch = self.root / ".github/scripts/patch-go-ios-arm64-runtime.py"
        self.patch.parent.mkdir(parents=True)
        self.patch.write_text("reviewed runtime patch")
        self.runtime = self.root / "asm_arm64.s"
        self.runtime.write_text("patched YIELD body")
        self.first = self.root / "first"
        self.repeat = self.root / "repeat"
        self.first.mkdir(); self.repeat.mkdir()
        for name in manifest_module.BINARIES:
            data = b"synthetic Mach-O bytes " + name.encode()
            (self.first / name).write_bytes(data)
            (self.repeat / name).write_bytes(data)

    def create(self, **kwargs):
        values = {
            "source_sha": "a" * 40,
            "source_tree_sha": "b" * 40,
            "go_version": "go1.27.3",
            "frontend_lockfile": self.lock,
            "workflow": self.workflow,
            "runtime_patch_script": self.patch,
            "patched_runtime_asm": self.runtime,
            "frontend_dist": self.dist,
            "primary_build": self.first,
            "repeat_build": self.repeat,
            "toolchain": {"xcodebuild": "Xcode 26", "iphoneos_sdk": "26.0"},
            "target": manifest_module.TARGET,
            "workflow_run_id": 777,
        }
        values.update(kwargs)
        return manifest_module.build_manifest(**values)

    def test_identical_clean_cache_builds_record_exact_inputs_and_hashes(self):
        result = self.create()
        self.assertEqual(result["classification"], "byte_identical")
        self.assertTrue(result["source_equivalent"])
        self.assertTrue(result["byte_identical"])
        self.assertEqual(result["source_sha"], "a" * 40)
        self.assertEqual(result["workflow_run_id"], 777)
        self.assertEqual(result["target"], manifest_module.TARGET)
        self.assertEqual(result["inputs"]["frontend_asset_count"], 3)
        expected = hashlib.sha256((self.first / manifest_module.BINARIES[0]).read_bytes()).hexdigest()
        self.assertEqual(result["builds"][manifest_module.BINARIES[0]]["primary"]["sha256"], expected)

    def test_differing_bytes_are_reported_with_offsets_without_false_pass(self):
        path = self.repeat / manifest_module.BINARIES[0]
        path.write_bytes(path.read_bytes().replace(b"Mach-O", b"mach-O"))
        result = self.create()
        observed = result["builds"][manifest_module.BINARIES[0]]
        self.assertEqual(result["classification"], "differences_detected")
        self.assertFalse(observed["byte_identical"])
        self.assertEqual(observed["differing_byte_count"], 1)
        self.assertTrue(observed["first_differing_offsets"])
        self.assertEqual(observed["first_differing_bytes"][0]["primary_byte"], "4d")
        self.assertEqual(observed["first_differing_bytes"][0]["repeat_byte"], "6d")
        self.assertEqual(result["nondeterminism_disposition"].split(";")[0], "unclassified")

    def test_wrong_source_or_target_is_rejected(self):
        with self.assertRaises(ValueError):
            self.create(source_sha="short")
        with self.assertRaises(ValueError):
            self.create(target={"GOOS": "ios", "GOARCH": "arm64", "GOARM64": "v8.1", "CGO_ENABLED": "1"})


if __name__ == "__main__":
    unittest.main()
