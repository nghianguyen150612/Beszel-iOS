"""Synthetic release validation tests. These tests never create tags or releases."""

import hashlib
import importlib.util
import json
from pathlib import Path
import struct
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / ".github" / "scripts" / "verify-ios-release-payload.py"
spec = importlib.util.spec_from_file_location("ios_release_payload", SCRIPT)
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


def macho_fixture(payload: bytes) -> bytes:
    commands = struct.pack("<6I", 0x32, 24, 2, 12 << 16, 18 << 16, 0)
    commands += struct.pack("<II16sQQQQIIII", 0x19, 72, b"__TEXT", 0x100000000, 4096, 152, len(payload), 7, 5, 0, 0)
    commands += struct.pack("<IIQQ", 0x80000028, 24, 152, 0)
    header = struct.pack("<8I", 0xFEEDFACF, 0x0100000C, 0, 2, 3, len(commands), 0, 0)
    return header + commands + payload


class IOSReleasePayloadTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.payload = self.root / "payload"
        self.payload.mkdir()
        self.source_sha = "a" * 40
        self.tag = "v0.21.0-ios.1"
        self.binaries = {}
        for name in release.NAMES:
            self.binaries[name] = macho_fixture(name.encode())
            (self.payload / name).write_bytes(self.binaries[name])
        sums = "".join(f"{hashlib.sha256(data).hexdigest()}  {name}\n" for name, data in self.binaries.items())
        (self.payload / "SHA256SUMS").write_text(sums)
        verified = release.load_artifacts_validator().verify(self.payload)
        self.metadata = {}
        for name, result in verified.items():
            self.metadata[name] = {
                **result,
                "build_info": {
                    "go": "go1.27.3",
                    "settings": {
                        "GOOS": "ios", "GOARCH": "arm64", "GOARM64": "v8.0", "CGO_ENABLED": "1",
                        "vcs.revision": self.source_sha, "vcs.modified": "false",
                    },
                },
            }
        self.metadata[release.NAMES[1]]["embedded_asset_count"] = 1
        self.metadata_path = self.root / "native-artifacts.json"
        self.save_metadata()
        (self.root / "beszel.go").write_text('package beszel\nconst (\n Version = "0.21.0"\n)\n')

    def tearDown(self):
        self.temp.cleanup()

    def save_metadata(self):
        self.metadata_path.write_text(json.dumps(self.metadata))

    def validate(self, tag=None, source_sha=None):
        return release.validate_payload(self.payload, self.metadata_path, tag or self.tag, source_sha or self.source_sha, self.root)

    def test_valid_synthetic_candidate_matches_payload_and_provenance(self):
        result = self.validate()
        self.assertFalse(result["release_created"])
        self.assertFalse(result["release_published"])
        self.assertEqual(result["source_sha"], self.source_sha)

    def test_validates_proposed_tag_against_repository_version(self):
        self.assertEqual(release.validate_tag("v0.21.0-ios.1", ROOT), "0.21.0")

    def test_rejects_wrong_or_non_candidate_tag(self):
        for tag in ("v0.22.0-ios.1", "v0.21.0-ios.0", "v0.21.0-ios.01", "v0.21.0", "candidate-v0.21.0-ios.1"):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                self.validate(tag=tag)

    def test_rejects_source_revision_mismatch(self):
        with self.assertRaisesRegex(ValueError, "vcs.revision"):
            self.validate(source_sha="b" * 40)

    def test_rejects_payload_checksum_mismatch(self):
        (self.payload / release.NAMES[0]).write_bytes(b"stale synthetic binary")
        with self.assertRaisesRegex(ValueError, "checksum mismatch"):
            self.validate()

    def test_rejects_missing_or_extra_payload_assets(self):
        (self.payload / release.NAMES[0]).unlink()
        with self.assertRaises(ValueError):
            self.validate()
        (self.payload / release.NAMES[0]).write_bytes(self.binaries[release.NAMES[0]])
        (self.payload / "stale.zip").write_bytes(b"stale")
        with self.assertRaisesRegex(ValueError, "exactly Agent, Hub and SHA256SUMS"):
            self.validate()

    def test_rejects_metadata_for_different_binary(self):
        self.metadata[release.NAMES[0]]["sha256"] = "0" * 64
        self.save_metadata()
        with self.assertRaisesRegex(ValueError, "metadata mismatch for sha256"):
            self.validate()


if __name__ == "__main__":
    unittest.main()
