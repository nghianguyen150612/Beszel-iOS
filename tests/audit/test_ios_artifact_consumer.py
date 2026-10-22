"""Synthetic tests for the independent same-run artifact consumer."""

import hashlib
import importlib.util
import io
import json
from pathlib import Path
import tempfile
import unittest
import urllib.request
import warnings
import zipfile
from unittest.mock import patch


ROOT = Path(__file__).resolve().parents[2]


def module(name):
    spec = importlib.util.spec_from_file_location(name, ROOT / ".github/scripts" / (name + ".py"))
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


consumer = module("verify-ios-artifact-consumer")
repro = module("create-ios-reproducibility-manifest")


def archive(files):
    stream = io.BytesIO()
    with zipfile.ZipFile(stream, "w", compression=zipfile.ZIP_DEFLATED) as zipped:
        for name, data in files.items():
            zipped.writestr(name, data)
    return stream.getvalue()


class ArtifactZipTests(unittest.TestCase):
    def test_exact_archive_file_set_and_checksums(self):
        agent = b"synthetic agent bytes"
        hub = b"synthetic hub bytes"
        sums = (f"{hashlib.sha256(agent).hexdigest()}  {consumer.AGENT}\n"
                f"{hashlib.sha256(hub).hexdigest()}  {consumer.HUB}\n").encode()
        files = {consumer.AGENT: agent, consumer.HUB: hub, "SHA256SUMS": sums}
        decoded = consumer.zip_files(archive(files), consumer.PAYLOAD_FILES)
        self.assertEqual(consumer.validate_checksum_manifest(decoded)[consumer.AGENT], hashlib.sha256(agent).hexdigest())

    def test_rejects_missing_extra_nested_duplicate_and_unsafe_zip_entries(self):
        expected = consumer.PAYLOAD_FILES
        valid_data = {name: b"x" for name in expected}
        for files in (
            {consumer.AGENT: b"x"},
            {**valid_data, "unexpected": b"x"},
            {**valid_data, "nested/file": b"x"},
        ):
            with self.subTest(files=sorted(files)), self.assertRaises(consumer.VerificationError):
                consumer.zip_files(archive(files), expected)
        stream = io.BytesIO()
        with warnings.catch_warnings():
            warnings.simplefilter("ignore", UserWarning)
            with zipfile.ZipFile(stream, "w") as zipped:
                for name in expected:
                    zipped.writestr(name, b"x")
                zipped.writestr(consumer.AGENT, b"duplicate")
        with self.assertRaises(consumer.VerificationError):
            consumer.zip_files(stream.getvalue(), expected)
        with self.assertRaises(consumer.VerificationError):
            consumer.zip_files(archive({"../escape": b"x"}), expected)

    def test_downloaded_files_must_match_the_archive_bytes(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            files = {name: name.encode() for name in consumer.PAYLOAD_FILES}
            for name, data in files.items():
                (root / name).write_bytes(data)
            self.assertEqual(set(consumer.extracted_files(root, files, files)), set(files))
            (root / consumer.AGENT).write_bytes(b"changed after download")
            with self.assertRaisesRegex(consumer.VerificationError, "differ"):
                consumer.extracted_files(root, files, files)

    def test_checksum_manifest_rejects_wrong_or_duplicate_binary_hashes(self):
        files = {consumer.AGENT: b"a", consumer.HUB: b"h"}
        files["SHA256SUMS"] = (f"{'0' * 64}  {consumer.AGENT}\n"
                               f"{hashlib.sha256(files[consumer.HUB]).hexdigest()}  {consumer.HUB}\n").encode()
        with self.assertRaisesRegex(consumer.VerificationError, "does not match"):
            consumer.validate_checksum_manifest(files)

    def test_download_redirect_does_not_forward_authorization(self):
        request = urllib.request.Request("https://api.github.com/artifact", headers={"Authorization": "Bearer hidden"})
        redirected = consumer.NoAuthRedirectHandler().redirect_request(
            request, None, 302, "Found", {}, "https://artifact-store.example/archive.zip?sig=hidden")
        self.assertIsNotNone(redirected)
        self.assertIsNone(redirected.get_header("Authorization"))
        self.assertEqual(redirected.get_header("Accept"), "application/zip")
        with self.assertRaises(consumer.VerificationError):
            consumer.NoAuthRedirectHandler().redirect_request(
                request, None, 302, "Found", {}, "http://artifact-store.example/archive.zip")

    def test_independent_zip_hash_must_match_github_metadata(self):
        data = archive({"file": b"native bytes"})
        artifact = {"id": 123, "name": "native", "digest": f"sha256:{hashlib.sha256(data).hexdigest()}"}
        with patch.object(consumer, "_http_bytes", return_value=(200, data)):
            observed, digest = consumer.download_artifact_zip("https://api.github.com/repo", artifact, "test-token")
        self.assertEqual(observed, data)
        self.assertEqual(digest, hashlib.sha256(data).hexdigest())
        artifact["digest"] = "sha256:" + "0" * 64
        with patch.object(consumer, "_http_bytes", return_value=(200, data)), self.assertRaises(consumer.VerificationError):
            consumer.download_artifact_zip("https://api.github.com/repo", artifact, "test-token")

    def test_hosted_extracted_bytes_remain_verifiable_when_raw_zip_is_forbidden(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            files = {name: ("synthetic " + name).encode() for name in consumer.PAYLOAD_FILES}
            for name, data in files.items():
                (root / name).write_bytes(data)
            artifact = {"id": 124, "name": "beszel-ios-arm64", "digest": "sha256:" + "a" * 64}
            with patch.object(consumer, "download_artifact_zip",
                              side_effect=consumer.VerificationError("GitHub artifact request failed with HTTP 403")):
                result = consumer.load_artifact_for_consumer(
                    "https://api.github.com/repo", artifact, "test-token", root, consumer.PAYLOAD_FILES)
            self.assertEqual(result["zip_download_status"], "blocked_http_403")
            self.assertIsNone(result["sha256"])
            self.assertEqual(set(result["payload_files"]), set(consumer.PAYLOAD_FILES))
            self.assertEqual(result["file_digests"][consumer.AGENT]["sha256"], hashlib.sha256(files[consumer.AGENT]).hexdigest())

    def test_run_artifact_listing_is_bound_to_source_and_expected_names(self):
        source_sha = "a" * 40
        required = [
            {"id": 1, "name": "beszel-ios-arm64", "digest": "sha256:" + "1" * 64, "expired": False},
            {"id": 2, "name": "ios-native-validation", "digest": "sha256:" + "2" * 64, "expired": False},
            {"id": 3, "name": "ios-wifi-diagnostic-disabled", "digest": "sha256:" + "3" * 64, "expired": False},
        ]
        pages = [
            {"id": 99, "head_sha": source_sha, "event": "pull_request", "head_branch": "ios/p025",
             "path": ".github/workflows/ios-build.yml", "name": "iOS Build"},
            {"total_count": len(required), "artifacts": required},
        ]
        with patch.object(consumer, "_json_api", side_effect=pages):
            _, run, found = consumer.github_run_artifacts("owner/repo", 99, "test-token", source_sha)
        self.assertEqual(run["head_sha"], source_sha)
        self.assertEqual(set(found), set(item["name"] for item in required))

        with patch.object(consumer, "_json_api", side_effect=[pages[0], {"total_count": 4, "artifacts": [*required, {"id": 4, "name": "other", "digest": "sha256:" + "4" * 64, "expired": False}]}]):
            with self.assertRaises(consumer.VerificationError):
                consumer.github_run_artifacts("owner/repo", 99, "test-token", source_sha)

    def test_native_x509_record_requires_exact_official_toolchain_evidence(self):
        validator = consumer._load_script(ROOT / "scripts/audit/validate-go-x509-native.py", "go_x509_test_fixture")
        evidence = {"status": "PASS", "advisory": "GO-2026-6809", "go_version": "go1.27.3",
                    "native_source_sha256": validator.SOURCE_SHA256,
                    "matches_reviewed_backport_outputs": True,
                    "fix_disposition": "native in official Go 1.27.3; custom P019/P022 backport not applied",
                    "standard_library_test": {"status": "PASS"}}
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "x509.json"
            path.write_text(json.dumps(evidence))
            self.assertEqual(consumer._validate_x509_report(path)["status"], "PASS")
            evidence["fix_disposition"] = "custom backport"
            path.write_text(json.dumps(evidence))
            with self.assertRaises(consumer.VerificationError):
                consumer._validate_x509_report(path)

    def test_source_selection_record_checks_ios_adapters_and_frontend_embed(self):
        prefix = "github.com/henrygd/beszel/"
        selected = {
            "ios": {
                prefix + "internal/cmd/agent": {"GoFiles": ["agent.go"]},
                prefix + "internal/cmd/hub": {"GoFiles": ["hub.go"]},
                prefix + "agent": {"GoFiles": ["system_platform_ios.go", "memory_platform_ios.go", "ios_memory.go", "ios_metadata.go", "ios_disk.go"]},
                prefix + "agent/battery": {"GoFiles": ["battery.go", "battery_ios.go"]},
                prefix + "agent/wifi": {"GoFiles": ["wifi.go", "wifi_ios.go"]},
                prefix + "internal/site": {"GoFiles": ["embed.go"], "EmbedFiles": ["dist/index.html", "dist/assets/app.js", "dist/assets/app.css"]},
            },
            "darwin": {},
            "linux": {},
            "wifi_platforms": {
                "ios": ["wifi.go", "wifi_ios.go"],
                "darwin": ["wifi.go", "wifi_darwin.go"],
                "linux": ["wifi.go", "wifi_linux.go"],
                "windows": ["wifi.go", "wifi_windows.go"],
                "freebsd": ["wifi.go", "wifi_unsupported.go"],
            },
        }
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "source-selection.json"
            path.write_text(json.dumps(selected))
            self.assertTrue(consumer._validate_source_selection(path)["inactive_ios_helper_excluded"])
            selected["wifi_platforms"]["ios"] = ["wifi.go", "wifi_darwin.go"]
            path.write_text(json.dumps(selected))
            with self.assertRaisesRegex(consumer.VerificationError, "ios Wi-Fi implementation"):
                consumer._validate_source_selection(path)

    def test_reproducibility_manifest_is_bound_to_source_assets_and_native_bytes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            frontend = root / "dist"
            (frontend / "assets").mkdir(parents=True)
            (frontend / "index.html").write_text('<script src="./assets/app.js"></script><link rel="stylesheet" href="./assets/app.css">')
            (frontend / "assets/app.js").write_text("console.log('candidate');")
            (frontend / "assets/app.css").write_text("body { color: black; }")
            payload = root / "payload"
            payload.mkdir()
            primary = root / "primary"
            repeat = root / "repeat"
            primary.mkdir(); repeat.mkdir()
            for name in repro.BINARIES:
                data = b"same synthetic native binary " + name.encode()
                (primary / name).write_bytes(data)
                (repeat / name).write_bytes(data)
                (payload / name).write_bytes(data)
            source_sha, tree_sha, run_id = "a" * 40, "b" * 40, 321
            asset_entries, asset_digest = consumer._frontend_manifest(frontend)
            inputs = {
                "frontend_lockfile_sha256": consumer.file_sha256(ROOT / "internal/site/bun.lock"),
                "build_workflow_sha256": consumer.file_sha256(ROOT / ".github/workflows/ios-build.yml"),
                "runtime_patch_script_sha256": consumer.file_sha256(ROOT / ".github/scripts/patch-go-ios-arm64-runtime.py"),
                "patched_runtime_asm_sha256": "c" * 64,
                "frontend_asset_count": len(asset_entries),
                "frontend_asset_manifest_sha256": asset_digest,
                "frontend_assets": asset_entries,
            }
            builds = {}
            for name in repro.BINARIES:
                digest = hashlib.sha256((payload / name).read_bytes()).hexdigest()
                size = (payload / name).stat().st_size
                builds[name] = {"primary": {"sha256": digest, "bytes": size},
                                "repeat": {"sha256": digest, "bytes": size}, "byte_identical": True}
            native = root / "native"
            native.mkdir()
            report = {"schema_version": 1, "source_sha": source_sha, "source_tree_sha": tree_sha,
                      "workflow_run_id": run_id,
                      "toolchain": {"go": "go1.27.3", "xcodebuild": "Xcode 16.4 Build version 16F6", "iphoneos_sdk": "18.5"},
                      "target": consumer.EXPECTED_TARGET, "build_flags": ["-trimpath", "-ldflags=-s -w"], "inputs": inputs,
                      "source_equivalent": True, "byte_identical": True, "classification": "byte_identical",
                      "build_isolation": "isolated test builds", "builds": builds}
            (native / "native-build-reproducibility.json").write_text(json.dumps(report))
            validated, assets, observed_digest = consumer._validate_reproducibility(
                native, frontend, ROOT, source_sha, tree_sha, run_id, {"patched_sha256": "c" * 64}, payload)
            self.assertTrue(validated["byte_identical"])
            self.assertEqual(validated["frontend_asset_manifest_sha256"], observed_digest)
            self.assertEqual(len(assets), len(asset_entries))

            report["source_tree_sha"] = "d" * 40
            (native / "native-build-reproducibility.json").write_text(json.dumps(report))
            with self.assertRaises(consumer.VerificationError):
                consumer._validate_reproducibility(native, frontend, ROOT, source_sha, tree_sha, run_id,
                                                   {"patched_sha256": "c" * 64}, payload)


if __name__ == "__main__":
    unittest.main()
