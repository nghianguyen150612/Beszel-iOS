#!/usr/bin/env python3
"""Validate a synthetic or real iOS release payload without publishing it."""

import argparse
import importlib.util
import json
from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[2]
ARTIFACTS_VALIDATOR = ROOT / ".github" / "scripts" / "verify-ios-artifacts.py"
NAMES = ("beszel-agent-ios-arm64", "beszel-hub-ios-arm64")


def load_artifacts_validator():
    spec = importlib.util.spec_from_file_location("ios_artifacts_validator", ARTIFACTS_VALIDATOR)
    if spec is None or spec.loader is None:
        raise ValueError("cannot load native artifact validator")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def source_version(repo_root: Path) -> str:
    text = (repo_root / "beszel.go").read_text()
    match = re.search(r'^\s*Version\s*=\s*"([0-9]+\.[0-9]+\.[0-9]+)"\s*$', text, re.MULTILINE)
    if not match:
        raise ValueError("cannot read a stable numeric Beszel version from beszel.go")
    return match.group(1)


def validate_tag(tag: str, repo_root: Path = ROOT) -> str:
    version = source_version(repo_root)
    if not re.fullmatch(rf"v{re.escape(version)}-ios\.[1-9][0-9]*", tag):
        raise ValueError(f"tag must match v{version}-ios.<positive integer>")
    return version


def validate_payload(directory: Path, metadata_path: Path, tag: str, source_sha: str, repo_root: Path = ROOT):
    validate_tag(tag, repo_root)
    if not re.fullmatch(r"[0-9a-f]{40}", source_sha):
        raise ValueError("source SHA must be a full lowercase Git commit SHA")

    native = load_artifacts_validator()
    verified = native.verify(directory)
    metadata = json.loads(metadata_path.read_text())
    if not isinstance(metadata, dict) or set(metadata) != set(NAMES):
        raise ValueError("native validation metadata must describe Agent and Hub exactly")

    for name in NAMES:
        observed = verified[name]
        recorded = metadata[name]
        if not isinstance(recorded, dict):
            raise ValueError(f"{name}: native metadata entry must be an object")
        for key in ("cpu", "filetype", "command", "platform", "minos", "sha256", "bytes"):
            if recorded.get(key) != observed.get(key):
                raise ValueError(f"{name}: native metadata mismatch for {key}")
        build = recorded.get("build_info")
        settings = build.get("settings", {}) if isinstance(build, dict) else {}
        if not isinstance(build, dict) or build.get("go") != "go1.27.3":
            raise ValueError(f"{name}: native metadata does not prove Go 1.27.3")
        if not isinstance(settings, dict):
            raise ValueError(f"{name}: native build settings must be an object")
        expected_settings = {
            "GOOS": "ios",
            "GOARCH": "arm64",
            "GOARM64": "v8.0",
            "CGO_ENABLED": "1",
            "vcs.revision": source_sha,
            "vcs.modified": "false",
        }
        for key, value in expected_settings.items():
            if settings.get(key) != value:
                raise ValueError(f"{name}: native build provenance mismatch for {key}")
        if name == NAMES[1] and (not isinstance(recorded.get("embedded_asset_count"), int) or recorded["embedded_asset_count"] < 1):
            raise ValueError("Hub metadata is missing the embedded frontend asset count")

    return {
        "schema": 1,
        "tag": tag,
        "source_sha": source_sha,
        "release_created": False,
        "release_published": False,
        "payload_files": [*NAMES, "SHA256SUMS"],
        "binaries": {name: {"sha256": verified[name]["sha256"], "bytes": verified[name]["bytes"]} for name in NAMES},
        "native_metadata": "matched exact payload bytes and source VCS revision",
        "go": "go1.27.3",
        "target": {"GOOS": "ios", "GOARCH": "arm64", "GOARM64": "v8.0", "CGO_ENABLED": "1", "minimum_ios": "12.0"},
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path, help="directory containing the three release payload files")
    parser.add_argument("--metadata", required=True, type=Path, help="same-run ios-native-validation/native-artifacts.json")
    parser.add_argument("--tag", required=True, help="synthetic or pushed iOS release tag; never creates a tag")
    parser.add_argument("--source-sha", required=True, help="source commit SHA whose binaries are being validated")
    parser.add_argument("--repo-root", type=Path, default=ROOT, help=argparse.SUPPRESS)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    try:
        result = validate_payload(args.directory, args.metadata, args.tag, args.source_sha, args.repo_root)
    except (OSError, ValueError, json.JSONDecodeError, KeyError, TypeError) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1
    rendered = json.dumps(result, indent=2) + "\n"
    print(rendered, end="")
    if args.output:
        args.output.write_text(rendered)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
