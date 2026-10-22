#!/usr/bin/env python3
"""Independently verify same-run native artifact ZIPs and extracted payload bytes."""

import argparse
import hashlib
import importlib.util
import io
import json
from pathlib import Path, PurePosixPath
import re
import stat
import urllib.error
import urllib.parse
import urllib.request
import zipfile


ROOT = Path(__file__).resolve().parents[2]
INTEGRATION_BASELINE_SHA = "07f6fd5f87c99348ec33549fe8dfed099464eb7f"
PINNED_UPSTREAM_SHA = "b97abf7662a7c7a2dd7542ea2062405abd22d4dc"
AGENT = "beszel-agent-ios-arm64"
HUB = "beszel-hub-ios-arm64"
PAYLOAD_FILES = (AGENT, HUB, "SHA256SUMS")
NATIVE_FILES = (
    "source-selection.json",
    "runtime-patch.json",
    "go-x509-security-native.json",
    "runtime-agent.json",
    "runtime-hub.json",
    "native-artifacts.json",
    "wifi-linkage.txt",
    "native-build-reproducibility.json",
)
REQUIRED_ARTIFACTS = {"beszel-ios-arm64", "ios-native-validation"}
ALLOWED_ARTIFACTS = REQUIRED_ARTIFACTS | {"ios-wifi-diagnostic-disabled"}
EXPECTED_RUNTIME_WORDS = ["b9400be0", "b4000080", "d503203f", "51000400", "b5ffffc0", "d65f03c0"]
EXPECTED_TARGET = {"GOOS": "ios", "GOARCH": "arm64", "GOARM64": "v8.0", "CGO_ENABLED": "1"}
MAX_JSON_BYTES = 16 * 1024 * 1024
MAX_ARCHIVE_BYTES = 128 * 1024 * 1024
MAX_ARCHIVE_ENTRY_BYTES = 96 * 1024 * 1024


class VerificationError(ValueError):
    pass


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def file_sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def _no_auth_redirect(req, fp, code, msg, headers, newurl):
    """Follow GitHub's signed artifact redirect without forwarding API headers."""
    parsed = urllib.parse.urlsplit(newurl)
    if parsed.scheme != "https" or not parsed.hostname or parsed.username or parsed.password:
        raise VerificationError("GitHub returned an invalid artifact download redirect")
    return urllib.request.Request(newurl, headers={"Accept": "application/zip"}, method="GET")


class NoAuthRedirectHandler(urllib.request.HTTPRedirectHandler):
    redirect_request = staticmethod(_no_auth_redirect)


def _http_bytes(url: str, token: str | None = None, max_bytes: int = MAX_JSON_BYTES):
    headers = {"Accept": "application/vnd.github+json", "X-GitHub-Api-Version": "2022-11-28"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    request = urllib.request.Request(url, headers=headers, method="GET")
    opener = urllib.request.build_opener(NoAuthRedirectHandler())
    try:
        with opener.open(request, timeout=90) as response:
            content_length = response.headers.get("Content-Length")
            if content_length and int(content_length) > max_bytes:
                raise VerificationError("GitHub artifact response exceeds the configured size limit")
            body = response.read(max_bytes + 1)
            if len(body) > max_bytes:
                raise VerificationError("GitHub artifact response exceeds the configured size limit")
            return response.status, body
    except VerificationError:
        raise
    except urllib.error.HTTPError as exc:
        raise VerificationError(f"GitHub artifact request failed with HTTP {exc.code}") from None
    except (urllib.error.URLError, TimeoutError, OSError) as exc:
        raise VerificationError(f"GitHub artifact request failed ({type(exc).__name__})") from None


def _json_api(url: str, token: str):
    status, body = _http_bytes(url, token)
    if status != 200:
        raise VerificationError(f"GitHub API returned HTTP {status}")
    try:
        return json.loads(body)
    except (UnicodeDecodeError, json.JSONDecodeError):
        raise VerificationError("GitHub API returned invalid JSON") from None


def github_run_artifacts(repository: str, run_id: int, token: str, expected_source_sha: str):
    if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repository):
        raise VerificationError("repository must use owner/name form")
    if run_id <= 0 or not re.fullmatch(r"[0-9a-f]{40}", expected_source_sha):
        raise VerificationError("workflow run or source identity is invalid")
    owner, repo = repository.split("/", 1)
    base = f"https://api.github.com/repos/{urllib.parse.quote(owner)}/{urllib.parse.quote(repo)}"
    run = _json_api(f"{base}/actions/runs/{run_id}", token)
    if run.get("id") != run_id or run.get("head_sha") != expected_source_sha:
        raise VerificationError("workflow run source SHA does not match the checked out candidate")
    if run.get("path") != ".github/workflows/ios-build.yml" or run.get("name") != "iOS Build":
        raise VerificationError("workflow run is not the native iOS build workflow")
    page = _json_api(f"{base}/actions/runs/{run_id}/artifacts?per_page=100", token)
    artifacts = page.get("artifacts")
    if not isinstance(artifacts, list) or page.get("total_count") != len(artifacts):
        raise VerificationError("workflow artifact listing is incomplete")
    by_name = {}
    for artifact in artifacts:
        name = artifact.get("name")
        if name not in ALLOWED_ARTIFACTS or name in by_name:
            raise VerificationError("workflow run has an unexpected or duplicate artifact name")
        if artifact.get("expired") or not isinstance(artifact.get("id"), int):
            raise VerificationError(f"workflow artifact is expired or has no ID: {name}")
        digest = artifact.get("digest")
        if name in REQUIRED_ARTIFACTS and not re.fullmatch(r"sha256:[0-9a-f]{64}", digest or ""):
            raise VerificationError(f"workflow artifact has no SHA-256 digest: {name}")
        by_name[name] = artifact
    if not REQUIRED_ARTIFACTS.issubset(by_name):
        raise VerificationError("workflow run is missing a required native artifact")
    return base, run, by_name


def download_artifact_zip(base: str, artifact: dict, token: str):
    if not isinstance(artifact.get("id"), int):
        raise VerificationError("artifact ID is invalid")
    status, archive = _http_bytes(f"{base}/actions/artifacts/{artifact['id']}/zip", token, MAX_ARCHIVE_BYTES)
    if status != 200:
        raise VerificationError(f"artifact download returned HTTP {status}")
    digest = sha256(archive)
    reported = artifact.get("digest", "")
    if reported != f"sha256:{digest}":
        raise VerificationError(f"independent ZIP SHA-256 differs from GitHub metadata for {artifact['name']}")
    return archive, digest


def zip_files(archive: bytes, expected_names):
    expected = set(expected_names)
    try:
        with zipfile.ZipFile(io.BytesIO(archive)) as zipped:
            files = {}
            total_size = 0
            for info in zipped.infolist():
                name = info.filename
                path = PurePosixPath(name)
                mode = (info.external_attr >> 16) & 0xFFFF
                if (info.is_dir() or name in files or "\\" in name or path.is_absolute()
                        or path.as_posix() != name or any(part in {"", ".", ".."} for part in path.parts)
                        or stat.S_ISLNK(mode) or info.flag_bits & 0x1):
                    raise VerificationError("artifact ZIP contains a duplicate, directory, link, or unsafe entry")
                if info.file_size > MAX_ARCHIVE_ENTRY_BYTES:
                    raise VerificationError("artifact ZIP entry exceeds the configured size limit")
                total_size += info.file_size
                if total_size > MAX_ARCHIVE_BYTES:
                    raise VerificationError("artifact ZIP contents exceed the configured size limit")
                if name not in expected:
                    raise VerificationError("artifact ZIP contains an unexpected file")
                files[name] = zipped.read(info)
    except VerificationError:
        raise
    except (OSError, zipfile.BadZipFile, RuntimeError, EOFError):
        raise VerificationError("artifact ZIP cannot be read") from None
    if set(files) != expected:
        raise VerificationError("artifact ZIP file layout is incomplete")
    return files


def directory_file_bytes(directory: Path, expected_names):
    if directory.is_symlink() or not directory.is_dir():
        raise VerificationError("downloaded artifact directory is missing or unsafe")
    entries = list(directory.iterdir())
    if {path.name for path in entries} != set(expected_names) or len(entries) != len(expected_names):
        raise VerificationError("downloaded artifact has an unexpected file layout")
    result = {}
    total_size = 0
    for path in entries:
        if path.is_symlink() or not path.is_file():
            raise VerificationError("downloaded artifact contains a non-regular file")
        data = path.read_bytes()
        total_size += len(data)
        if len(data) > MAX_ARCHIVE_ENTRY_BYTES or total_size > MAX_ARCHIVE_BYTES:
            raise VerificationError("downloaded artifact contents exceed the configured size limit")
        result[path.name] = data
    return result


def file_digests(files: dict):
    return {name: {"bytes": len(data), "sha256": sha256(data)} for name, data in sorted(files.items())}


def extracted_files(directory: Path, expected_names, zipped_files: dict):
    result = directory_file_bytes(directory, expected_names)
    for name, data in result.items():
        if data != zipped_files[name]:
            raise VerificationError(f"actions/download-artifact bytes differ from the independently downloaded ZIP: {name}")
    return file_digests(result)


def load_artifact_for_consumer(base: str, artifact: dict, token: str,
                               downloaded_dir: Path, expected_names):
    try:
        archive, archive_sha = download_artifact_zip(base, artifact, token)
    except VerificationError as exc:
        if "HTTP 403" not in str(exc):
            raise
        # A hosted actions/download-artifact step has already retrieved the
        # exact same-run artifact. Keep verifying those bytes and record that
        # independent raw-ZIP hashing was blocked instead of substituting the
        # API-reported ZIP digest.
        files = directory_file_bytes(downloaded_dir, expected_names)
        return {
            "sha256": None,
            "zip_download_status": "blocked_http_403",
            "file_digests": file_digests(files),
            "payload_files": files if artifact["name"] == "beszel-ios-arm64" else None,
            "native_files": files if artifact["name"] == "ios-native-validation" else None,
        }
    files = zip_files(archive, expected_names)
    extracted_files(downloaded_dir, expected_names, files)
    return {
        "sha256": archive_sha,
        "zip_download_status": "independent_zip_verified",
        "file_digests": file_digests(files),
        "payload_files": files if artifact["name"] == "beszel-ios-arm64" else None,
        "native_files": files if artifact["name"] == "ios-native-validation" else None,
    }


def validate_checksum_manifest(files: dict):
    if set(files) != set(PAYLOAD_FILES):
        raise VerificationError("payload archive must contain Agent, Hub, and SHA256SUMS only")
    entries = {}
    for line in files["SHA256SUMS"].decode("utf-8").splitlines():
        match = re.fullmatch(r"([0-9a-f]{64}) [ *](beszel-(?:agent|hub)-ios-arm64)", line)
        if not match or match[2] in entries:
            raise VerificationError("SHA256SUMS has an invalid or duplicate entry")
        entries[match[2]] = match[1]
    if set(entries) != {AGENT, HUB}:
        raise VerificationError("SHA256SUMS must list Agent and Hub exactly once")
    for name in (AGENT, HUB):
        if entries[name] != sha256(files[name]):
            raise VerificationError(f"SHA256SUMS does not match exact payload bytes: {name}")
    return entries


def _load_script(path: Path, module_name: str):
    spec = importlib.util.spec_from_file_location(module_name, path)
    if spec is None or spec.loader is None:
        raise VerificationError(f"cannot load validation script: {path.name}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def _json_file(path: Path):
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeDecodeError, json.JSONDecodeError):
        raise VerificationError(f"invalid JSON artifact file: {path.name}") from None
    if not isinstance(value, dict):
        raise VerificationError(f"JSON artifact must be an object: {path.name}")
    return value


def _validate_runtime_report(path: Path):
    report = _json_file(path)
    words = report.get("instructions")
    if (report.get("status") != "PASS" or report.get("counter_access") is not False
            or report.get("symbol") not in {"runtime.procyieldAsm", "runtime.procyieldAsm.abi0"}
            or not isinstance(words, list) or words[:6] != EXPECTED_RUNTIME_WORDS
            or any(word != "00000000" for word in words[6:]) or len(words) not in {6, 8}
            or not re.fullmatch(r"[0-9a-f]{64}", report.get("binary_sha256", ""))
            or not isinstance(report.get("binary_bytes"), int) or report["binary_bytes"] <= 0):
        raise VerificationError(f"Apple A7 runtime instruction report failed validation: {path.name}")
    return {key: report[key] for key in ("symbol", "instructions", "counter_access", "binary_sha256", "binary_bytes", "status")}


def _validate_x509_report(path: Path):
    report = _json_file(path)
    validator = _load_script(ROOT / "scripts/audit/validate-go-x509-native.py", "go_x509_native_validator")
    test_result = report.get("standard_library_test")
    if (report.get("status") != "PASS" or report.get("advisory") != "GO-2026-6809"
            or report.get("go_version") != validator.EXPECTED_GO
            or report.get("native_source_sha256") != validator.SOURCE_SHA256
            or report.get("matches_reviewed_backport_outputs") is not True
            or not isinstance(test_result, dict) or test_result.get("status") != "PASS"
            or report.get("fix_disposition") != "native in official Go 1.27.3; custom P019/P022 backport not applied"):
        raise VerificationError("native x509 validation evidence does not match the reviewed Go 1.27.3 fix")
    return {
        "status": report["status"],
        "advisory": report["advisory"],
        "go_version": report["go_version"],
        "native_source_sha256": report["native_source_sha256"],
        "fix_disposition": report["fix_disposition"],
        "standard_library_test": test_result,
    }


def _validate_source_selection(path: Path):
    report = _json_file(path)
    module = "github.com/henrygd/beszel/"
    if set(report) != {"ios", "darwin", "linux", "wifi_platforms"}:
        raise VerificationError("source-selection report does not cover the required operating systems")
    ios = report["ios"]
    if not isinstance(ios, dict):
        raise VerificationError("iOS source-selection report is malformed")
    package = lambda suffix: ios.get(module + suffix)
    expected_files = {
        "internal/cmd/agent": {"agent.go"},
        "internal/cmd/hub": {"hub.go"},
        "agent": {"system_platform_ios.go", "memory_platform_ios.go", "ios_memory.go", "ios_metadata.go", "ios_disk.go"},
        "agent/battery": {"battery_ios.go"},
        "agent/wifi": {"wifi_ios.go"},
        "internal/site": {"embed.go"},
    }
    for suffix, required in expected_files.items():
        entry = package(suffix)
        if not isinstance(entry, dict):
            raise VerificationError(f"source-selection report omits iOS package {suffix}")
        selected = set(entry.get("GoFiles", [])) | set(entry.get("CgoFiles", [])) | set(entry.get("SFiles", []))
        if not required.issubset(selected):
            raise VerificationError(f"source-selection report has unexpected iOS source selection: {suffix}")
    if module + "agent/wifi/ioshelper" in ios:
        raise VerificationError("inactive iOS Wi-Fi helper appears in the Agent/Hub build dependency graph")
    embed_files = package("internal/site").get("EmbedFiles", [])
    if ("dist/index.html" not in embed_files
            or not any(name.startswith("dist/assets/") and name.endswith(".js") for name in embed_files)
            or not any(name.startswith("dist/assets/") and name.endswith(".css") for name in embed_files)):
        raise VerificationError("source-selection report does not include production frontend assets")

    wifi_platforms = report["wifi_platforms"]
    expected_wifi = {
        "ios": "wifi_ios.go",
        "darwin": "wifi_darwin.go",
        "linux": "wifi_linux.go",
        "windows": "wifi_windows.go",
        "freebsd": "wifi_unsupported.go",
    }
    if not isinstance(wifi_platforms, dict) or set(wifi_platforms) != set(expected_wifi):
        raise VerificationError("source-selection report omits a required Wi-Fi target")
    for target, expected in expected_wifi.items():
        if set(wifi_platforms[target]) != {"wifi.go", expected}:
            raise VerificationError(f"source-selection report has unexpected {target} Wi-Fi implementation")
    return {
        "sha256": file_sha256(path),
        "targets": ["ios", "darwin", "linux"],
        "wifi_targets": sorted(expected_wifi),
        "inactive_ios_helper_excluded": True,
        "production_frontend_selected": True,
    }


def _frontend_manifest(frontend: Path):
    validator = _load_script(ROOT / ".github/scripts/verify-ios-artifacts.py", "ios_artifacts_validator")
    try:
        paths = validator.frontend_assets(frontend)
    except (OSError, ValueError) as exc:
        raise VerificationError(f"frontend assets failed source validation: {exc}") from None
    assets = []
    for path in paths:
        if path.is_symlink() or not path.is_file():
            raise VerificationError("frontend tree contains a non-regular file")
        assets.append({"path": path.relative_to(frontend).as_posix(), "bytes": path.stat().st_size, "sha256": file_sha256(path)})
    canonical = json.dumps(assets, sort_keys=True, separators=(",", ":")).encode()
    return assets, sha256(canonical)


def _validate_reproducibility(native_dir: Path, frontend: Path, source_root: Path,
                              source_sha: str, source_tree_sha: str, run_id: int,
                              runtime_patch: dict, payload_dir: Path):
    report_path = native_dir / "native-build-reproducibility.json"
    report = _json_file(report_path)
    assets, asset_sha = _frontend_manifest(frontend)
    input_digests = {
        "frontend_lockfile_sha256": file_sha256(source_root / "internal/site/bun.lock"),
        "build_workflow_sha256": file_sha256(source_root / ".github/workflows/ios-build.yml"),
        "runtime_patch_script_sha256": file_sha256(source_root / ".github/scripts/patch-go-ios-arm64-runtime.py"),
    }
    inputs = report.get("inputs")
    toolchain = report.get("toolchain")
    if (report.get("source_sha") != source_sha or report.get("source_tree_sha") != source_tree_sha
            or report.get("workflow_run_id") != run_id or not isinstance(toolchain, dict)
            or toolchain.get("go") != "go1.27.3"
            or not isinstance(toolchain.get("xcodebuild"), str)
            or not toolchain["xcodebuild"].startswith("Xcode ")
            or "Build version " not in toolchain["xcodebuild"]
            or not re.fullmatch(r"[0-9]+\.[0-9]+(?:\.[0-9]+)?", toolchain.get("iphoneos_sdk", ""))
            or report.get("target") != EXPECTED_TARGET or report.get("source_equivalent") is not True
            or report.get("build_flags") != ["-trimpath", "-ldflags=-s -w"]
            or not isinstance(inputs, dict)):
        raise VerificationError("reproducibility manifest source, toolchain, or target does not match")
    for key, expected in input_digests.items():
        if inputs.get(key) != expected:
            raise VerificationError(f"reproducibility input digest mismatch: {key}")
    if (inputs.get("frontend_asset_count") != len(assets)
            or inputs.get("frontend_asset_manifest_sha256") != asset_sha
            or inputs.get("frontend_assets") != assets
            or inputs.get("patched_runtime_asm_sha256") != runtime_patch.get("patched_sha256")):
        raise VerificationError("frontend or patched runtime input differs from the native build record")
    builds = report.get("builds")
    if not isinstance(builds, dict) or set(builds) != {AGENT, HUB}:
        raise VerificationError("reproducibility manifest is missing Agent/Hub comparisons")
    for name in (AGENT, HUB):
        item = builds[name]
        if not isinstance(item, dict) or not isinstance(item.get("primary"), dict) or not isinstance(item.get("repeat"), dict):
            raise VerificationError(f"reproducibility comparison is invalid: {name}")
        if (item["primary"].get("sha256") != sha256((payload_dir / name).read_bytes())
                or not isinstance(item["primary"].get("bytes"), int)
                or not isinstance(item["repeat"].get("sha256"), str)
                or not re.fullmatch(r"[0-9a-f]{64}", item["repeat"]["sha256"])):
            raise VerificationError(f"reproducibility hashes do not identify the uploaded primary binary: {name}")
        same = item["primary"]["sha256"] == item["repeat"]["sha256"] and item["primary"]["bytes"] == item["repeat"].get("bytes")
        if item.get("byte_identical") is not same:
            raise VerificationError(f"reproducibility status contradicts recorded hashes: {name}")
    expected_identical = all(builds[name]["byte_identical"] for name in (AGENT, HUB))
    if report.get("byte_identical") is not expected_identical:
        raise VerificationError("overall reproducibility status contradicts per-binary comparisons")
    expected_classification = "byte_identical" if expected_identical else "differences_detected"
    if report.get("classification") != expected_classification:
        raise VerificationError("reproducibility classification contradicts per-binary comparisons")
    return {
        "classification": report.get("classification"),
        "byte_identical": expected_identical,
        "source_equivalent": True,
        "build_isolation": report.get("build_isolation"),
        "builds": builds,
        "input_sha256": input_digests,
        "patched_runtime_asm_sha256": inputs["patched_runtime_asm_sha256"],
        "frontend_asset_manifest_sha256": asset_sha,
    }, assets, asset_sha


def verify_candidate_artifacts(payload_dir: Path, native_dir: Path, frontend: Path,
                               source_root: Path, source_sha: str, source_tree_sha: str,
                               run_id: int, artifact_info: dict, archive_info: dict,
                               workflow_run: dict, repository: str):
    payload_archive = archive_info["beszel-ios-arm64"]
    native_archive = archive_info["ios-native-validation"]
    payload_hashes = validate_checksum_manifest(payload_archive["payload_files"])
    if set(native_archive["native_files"]) != set(NATIVE_FILES):
        raise VerificationError("native validation archive has an unexpected file layout")
    if file_digests(directory_file_bytes(payload_dir, PAYLOAD_FILES)) != payload_archive["file_digests"]:
        raise VerificationError("payload byte digest changed after artifact download")
    if file_digests(directory_file_bytes(native_dir, NATIVE_FILES)) != native_archive["file_digests"]:
        raise VerificationError("native validation byte digest changed after artifact download")

    artifacts_validator = _load_script(ROOT / ".github/scripts/verify-ios-artifacts.py", "ios_artifacts_validator")
    try:
        native_bytes = artifacts_validator.verify(payload_dir, frontend, inspect_go=True, expected_commit=source_sha)
    except Exception as exc:
        raise VerificationError(f"downloaded native payload failed independent inspection ({type(exc).__name__})") from None

    native_metadata_path = native_dir / "native-artifacts.json"
    metadata = _json_file(native_metadata_path)
    release_validator = _load_script(ROOT / ".github/scripts/verify-ios-release-payload.py", "ios_release_payload_validator")
    try:
        release_result = release_validator.validate_payload(
            payload_dir, native_metadata_path, "v0.21.0-ios.1", source_sha, source_root)
    except (OSError, ValueError, KeyError, TypeError) as exc:
        raise VerificationError(f"same-run native provenance does not match exact payload ({type(exc).__name__})") from None

    patch = _json_file(native_dir / "runtime-patch.json")
    patch_source = (source_root / ".github/scripts/patch-go-ios-arm64-runtime.py").read_text(encoding="utf-8")
    source_match = re.search(r'^SUPPORTED_SOURCE_SHA256 = "([0-9a-f]{64})"$', patch_source, re.MULTILINE)
    if (patch.get("status") != "PASS" or patch.get("go") != "go1.27.3"
            or patch.get("replacement") != "zero-guarded YIELD loop; no CNTFRQ_EL0/CNTVCT_EL0 reads"
            or not source_match or patch.get("source_sha256") != source_match.group(1)
            or not re.fullmatch(r"[0-9a-f]{64}", patch.get("patched_sha256", ""))):
        raise VerificationError("native runtime patch report is not the reviewed Go 1.27.3 A7 patch")
    runtime_reports = {
        "agent": _validate_runtime_report(native_dir / "runtime-agent.json"),
        "hub": _validate_runtime_report(native_dir / "runtime-hub.json"),
    }
    source_selection = _validate_source_selection(native_dir / "source-selection.json")
    x509_report = _validate_x509_report(native_dir / "go-x509-security-native.json")
    linkage = (native_dir / "wifi-linkage.txt").read_text(encoding="utf-8", errors="replace")
    if re.search(r"MobileWiFi|CoreWLAN|Apple80211|_WiFi(?:Manager|Device|Network)", linkage):
        raise VerificationError("private MobileWiFi linkage appears in native validation evidence")

    reproducibility, assets, asset_sha = _validate_reproducibility(
        native_dir, frontend, source_root, source_sha, source_tree_sha, run_id, patch, payload_dir)
    if metadata.get(HUB, {}).get("embedded_asset_count") != len(assets):
        raise VerificationError("native metadata frontend asset count does not match independent source build")
    binary_files = {
        name: {"sha256": native_bytes[name]["sha256"], "bytes": native_bytes[name]["bytes"]}
        for name in PAYLOAD_FILES[:2]
    }
    for name, value in binary_files.items():
        if payload_hashes[name] != value["sha256"]:
            raise VerificationError(f"payload SHA256SUMS mismatch after native inspection: {name}")

    return {
        "schema_version": 1,
        "verification_status": "PASS",
        "repository": repository,
        "workflow_run": {
            "run_id": run_id,
            "event": workflow_run.get("event"),
            "head_branch": workflow_run.get("head_branch"),
            "head_sha": source_sha,
            "head_tree_sha": source_tree_sha,
        },
        "candidate_identity": {
            "integration_baseline_sha": INTEGRATION_BASELINE_SHA,
            "feature_branch": "ios/p025-artifact-rc-freeze",
            "source_sha": source_sha,
            "tree_sha": source_tree_sha,
            "pinned_upstream_sha": PINNED_UPSTREAM_SHA,
            "go_version": "1.27.3",
            "frontend_lockfile_sha256": file_sha256(source_root / "internal/site/bun.lock"),
            "build_workflow_sha256": file_sha256(source_root / ".github/workflows/ios-build.yml"),
            "runtime_patch_script_sha256": file_sha256(source_root / ".github/scripts/patch-go-ios-arm64-runtime.py"),
            "artifact_workflow_run_id": run_id,
            "proposed_release_version": "v0.21.0-ios.1",
            "physical_acceptance_status": "pending_p026",
            "release_authorization_status": "not_authorized",
        },
        "source": {"commit_sha": source_sha, "tree_sha": source_tree_sha},
        "artifacts": {
            name: {
                "artifact_id": artifact_info[name]["id"],
                "github_reported_archive_sha256": artifact_info[name]["digest"].removeprefix("sha256:"),
                "independently_downloaded_archive_sha256": archive_info[name]["sha256"],
                "archive_download_status": archive_info[name]["zip_download_status"],
                "archive_digest_matches_github": True if archive_info[name]["sha256"] is not None else None,
                "files": archive_info[name]["file_digests"],
            }
            for name in sorted(REQUIRED_ARTIFACTS)
        },
        "payload": {
            "hosted_actions_downloaded_bytes_verified": True,
            "sha256_manifest_valid": True,
            "binaries": binary_files,
            "native_validation_manifest_sha256": file_sha256(native_metadata_path),
            "mach_o_and_go_metadata": native_bytes,
            "release_payload_layout": release_result["payload_files"],
        },
        "runtime": {
            "go": "go1.27.3",
            "minimum_ios": "12.0",
            "target": EXPECTED_TARGET,
            "patch_source_sha256": file_sha256(source_root / ".github/scripts/patch-go-ios-arm64-runtime.py"),
            "patched_runtime_asm_sha256": patch["patched_sha256"],
            "a7_instruction_reports": runtime_reports,
            "native_x509": x509_report,
            "private_mobilewifi_linkage": False,
        },
        "source_selection": source_selection,
        "frontend": {
            "lockfile_sha256": file_sha256(source_root / "internal/site/bun.lock"),
            "asset_count": len(assets),
            "asset_manifest_sha256": asset_sha,
            "hub_assets_match_independent_source_build": True,
        },
        "reproducibility": reproducibility,
        "release_authorization": "not_authorized",
        "proposed_release_version": "v0.21.0-ios.1",
        "tag_created": False,
        "release_published": False,
        "physical_device_accessed": False,
    }


def verify_run(args):
    token = args.token
    if not token:
        raise VerificationError("read-only GitHub token is unavailable")
    if args.source_root.resolve() != ROOT:
        raise VerificationError("source root must be the checked-out candidate repository")
    source_root = args.source_root.resolve()
    source_sha = args.source_sha
    source_tree_sha = args.source_tree_sha
    if not re.fullmatch(r"[0-9a-f]{40}", source_sha) or not re.fullmatch(r"[0-9a-f]{40}", source_tree_sha):
        raise VerificationError("candidate source and tree must be full Git SHA-1 values")
    base, workflow_run, artifacts = github_run_artifacts(args.repository, args.run_id, token, source_sha)
    if workflow_run.get("conclusion") not in {None, "success"}:
        raise VerificationError("native build workflow did not succeed")

    archives = {}
    for name in sorted(REQUIRED_ARTIFACTS):
        expected = PAYLOAD_FILES if name == "beszel-ios-arm64" else NATIVE_FILES
        directory = args.payload if name == "beszel-ios-arm64" else args.native
        archives[name] = load_artifact_for_consumer(base, artifacts[name], token, directory, expected)
    if len(archives) != len(REQUIRED_ARTIFACTS):
        raise VerificationError("not all required native artifacts were downloaded")
    result = verify_candidate_artifacts(
        args.payload, args.native, args.frontend, source_root, source_sha, source_tree_sha,
        args.run_id, artifacts, archives, workflow_run, args.repository)
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repository", required=True)
    parser.add_argument("--run-id", type=int, required=True)
    parser.add_argument("--source-sha", required=True)
    parser.add_argument("--source-tree-sha", required=True)
    parser.add_argument("--payload", type=Path, required=True)
    parser.add_argument("--native", type=Path, required=True)
    parser.add_argument("--frontend", type=Path, required=True)
    parser.add_argument("--source-root", type=Path, default=ROOT)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--token", default=None, help=argparse.SUPPRESS)
    args = parser.parse_args()
    import os
    args.token = args.token or os.environ.get("GH_TOKEN")
    try:
        result = verify_run(args)
        args.output.write_text(json.dumps(result, indent=2) + "\n")
    except (OSError, VerificationError, ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
        raise SystemExit(f"ERROR: independent artifact verification failed: {exc}") from None
    print(f"PASS: independently verified workflow run {args.run_id} source {args.source_sha}")
    print(f"PASS: Agent SHA-256 {result['payload']['binaries'][AGENT]['sha256']}")
    print(f"PASS: Hub SHA-256 {result['payload']['binaries'][HUB]['sha256']}")


if __name__ == "__main__":
    main()
