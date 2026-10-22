#!/usr/bin/env python3
"""Record inputs and byte comparison for two isolated native build invocations."""

import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import re
import struct
import subprocess


BINARIES = ("beszel-agent-ios-arm64", "beszel-hub-ios-arm64")
TARGET = {"GOOS": "ios", "GOARCH": "arm64", "GOARM64": "v8.0", "CGO_ENABLED": "1"}
ROOT = Path(__file__).resolve().parents[2]


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def _load_frontend_assets_validator():
    path = ROOT / ".github/scripts/verify-ios-artifacts.py"
    spec = importlib.util.spec_from_file_location("ios_artifact_validator", path)
    if spec is None or spec.loader is None:
        raise ValueError("cannot load the native artifact validator")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def frontend_manifest(directory: Path):
    validator = _load_frontend_assets_validator()
    paths = validator.frontend_assets(directory)
    assets = []
    for path in paths:
        if path.is_symlink() or not path.is_file():
            raise ValueError(f"frontend asset must be a regular file: {path.name}")
        assets.append({
            "path": path.relative_to(directory).as_posix(),
            "bytes": path.stat().st_size,
            "sha256": sha256(path),
        })
    canonical = json.dumps(assets, sort_keys=True, separators=(",", ":")).encode()
    return assets, hashlib.sha256(canonical).hexdigest()


def _macho_regions(path: Path):
    try:
        with path.open("rb") as stream:
            header = stream.read(32)
            if len(header) != 32:
                return []
            words = struct.unpack("<8I", header)
            if words[0] != 0xFEEDFACF or words[4] > 65536:
                return []
            load_commands = stream.read(words[5])
    except (OSError, struct.error):
        return []
    if len(load_commands) != words[5]:
        return []
    regions = [(0, 32, "Mach-O header"), (32, 32 + len(load_commands), "Mach-O load commands")]
    cursor = 0
    for _ in range(words[4]):
        if cursor + 8 > len(load_commands):
            return regions
        command, length = struct.unpack_from("<2I", load_commands, cursor)
        if length < 8 or cursor + length > len(load_commands):
            return regions
        if command == 0x19 and length >= 72:
            segment = load_commands[cursor + 8:cursor + 24].split(b"\0", 1)[0].decode("ascii", "replace")
            fileoff, filesize = struct.unpack_from("<QQ", load_commands, cursor + 40)
            regions.append((fileoff, fileoff + filesize, segment or "Mach-O segment"))
            nsects = struct.unpack_from("<I", load_commands, cursor + 64)[0]
            section_cursor = cursor + 72
            if length == 72 + nsects * 80:
                for _ in range(nsects):
                    sect = load_commands[section_cursor:section_cursor + 80]
                    section_name = sect[:16].split(b"\0", 1)[0].decode("ascii", "replace")
                    segment_name = sect[16:32].split(b"\0", 1)[0].decode("ascii", "replace")
                    size = struct.unpack_from("<Q", sect, 40)[0]
                    file_offset = struct.unpack_from("<I", sect, 48)[0]
                    if size:
                        regions.append((file_offset, file_offset + size,
                                        f"{segment_name or segment},{section_name or 'unnamed section'}"))
                    section_cursor += 80
        cursor += length
    return regions


def _region_for_offset(regions, offset):
    # Prefer a named section to its containing segment or load-command range.
    for start, end, name in reversed(regions):
        if start <= offset < end and "," in name:
            return name
    for start, end, name in regions:
        if start <= offset < end:
            return name
    return "outside mapped Mach-O sections"


def compare_binary(first: Path, second: Path, first_build_id=None, second_build_id=None):
    if first.is_symlink() or second.is_symlink() or not first.is_file() or not second.is_file():
        raise ValueError("both reproducibility outputs must be regular binary files")
    left_size, right_size = first.stat().st_size, second.stat().st_size
    left_sha, right_sha = sha256(first), sha256(second)
    identical = left_size == right_size and left_sha == right_sha
    first_offsets = []
    first_differences = []
    different_bytes = 0
    offset = 0
    regions = _macho_regions(first) if not identical else []
    with first.open("rb") as left, second.open("rb") as right:
        while True:
            a, b = left.read(1024 * 1024), right.read(1024 * 1024)
            if not a and not b:
                break
            common = min(len(a), len(b))
            for index, (one, two) in enumerate(zip(a[:common], b[:common])):
                if one != two:
                    different_bytes += 1
                    if len(first_offsets) < 32:
                        first_offsets.append(offset + index)
                        first_differences.append({
                            "offset": offset + index,
                            "primary_byte": f"{one:02x}",
                            "repeat_byte": f"{two:02x}",
                            "section": _region_for_offset(regions, offset + index),
                        })
            different_bytes += abs(len(a) - len(b))
            if len(first_offsets) < 32 and len(a) != len(b):
                extra = a[common:] if len(a) > common else b[common:]
                primary_extra = a[common:] if len(a) > len(b) else b""
                repeat_extra = b[common:] if len(b) > len(a) else b""
                count = min(len(extra), 32 - len(first_offsets))
                for index in range(count):
                    file_offset = offset + common + index
                    first_offsets.append(file_offset)
                    first_differences.append({
                        "offset": file_offset,
                        "primary_byte": f"{primary_extra[index]:02x}" if index < len(primary_extra) else None,
                        "repeat_byte": f"{repeat_extra[index]:02x}" if index < len(repeat_extra) else None,
                        "section": _region_for_offset(regions, file_offset),
                    })
            offset += max(len(a), len(b))
    return {
        "primary": {"sha256": left_sha, "bytes": left_size, "go_build_id": first_build_id},
        "repeat": {"sha256": right_sha, "bytes": right_size, "go_build_id": second_build_id},
        "byte_identical": identical,
        "differing_byte_count": different_bytes,
        "first_differing_offsets": first_offsets,
        "first_differing_bytes": first_differences,
    }


def build_manifest(*, source_sha: str, source_tree_sha: str, go_version: str,
                   frontend_lockfile: Path, workflow: Path, runtime_patch_script: Path,
                   patched_runtime_asm: Path, frontend_dist: Path, primary_build: Path,
                   repeat_build: Path, toolchain: dict, target: dict,
                   build_ids=None, workflow_run_id=None):
    if not re.fullmatch(r"[0-9a-f]{40}", source_sha) or not re.fullmatch(r"[0-9a-f]{40}", source_tree_sha):
        raise ValueError("source and tree identity must be full lowercase Git SHA-1 values")
    if go_version != "go1.27.3":
        raise ValueError("reproducibility assessment requires Go 1.27.3")
    if target != TARGET:
        raise ValueError("reproducibility assessment requires iOS ARM64 v8.0 with cgo")
    assets, assets_sha = frontend_manifest(frontend_dist)
    builds = {}
    for name in BINARIES:
        ids = (build_ids or {}).get(name, {})
        builds[name] = compare_binary(primary_build / name, repeat_build / name,
                                      ids.get("primary"), ids.get("repeat"))
    byte_identical = all(item["byte_identical"] for item in builds.values())
    return {
        "schema_version": 1,
        "source_sha": source_sha,
        "source_tree_sha": source_tree_sha,
        "workflow_run_id": workflow_run_id,
        "toolchain": {"go": go_version, **toolchain},
        "target": target,
        "build_flags": ["-trimpath", "-ldflags=-s -w"],
        "build_isolation": "separate output directories and Go build caches on the same hosted runner",
        "inputs": {
            "frontend_lockfile_sha256": sha256(frontend_lockfile),
            "build_workflow_sha256": sha256(workflow),
            "runtime_patch_script_sha256": sha256(runtime_patch_script),
            "patched_runtime_asm_sha256": sha256(patched_runtime_asm),
            "frontend_asset_count": len(assets),
            "frontend_asset_manifest_sha256": assets_sha,
            "frontend_assets": assets,
        },
        "builds": builds,
        "source_equivalent": True,
        "byte_identical": byte_identical,
        "classification": "byte_identical" if byte_identical else "differences_detected",
        "nondeterminism_disposition": "none_observed" if byte_identical else "unclassified; inspect differing offsets, Go build IDs, Mach-O sections, and asset/input fingerprints",
    }


def _read_build_id(path: Path):
    try:
        result = subprocess.run(["go", "tool", "buildid", str(path)], check=True,
                                capture_output=True, text=True)
    except (OSError, subprocess.CalledProcessError):
        return None
    value = result.stdout.strip()
    return value or None


def _git_value(root: Path, revision: str):
    return subprocess.check_output(["git", "-C", str(root), "rev-parse", revision], text=True).strip()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT)
    parser.add_argument("--primary-build", required=True, type=Path)
    parser.add_argument("--repeat-build", required=True, type=Path)
    parser.add_argument("--runtime-asm", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--xcode-version", required=True)
    parser.add_argument("--ios-sdk-version", required=True)
    parser.add_argument("--workflow-run-id", type=int)
    args = parser.parse_args()
    root = args.root.resolve()
    try:
        go_version = subprocess.check_output(["go", "env", "GOVERSION"], text=True).strip()
        target_values = subprocess.check_output(
            ["go", "env", "GOOS", "GOARCH", "GOARM64", "CGO_ENABLED"], text=True).splitlines()
        if len(target_values) != 4:
            raise ValueError("go env returned an invalid target tuple")
        target = dict(zip(TARGET, target_values))
        ids = {name: {
            "primary": _read_build_id(args.primary_build / name),
            "repeat": _read_build_id(args.repeat_build / name),
        } for name in BINARIES}
        report = build_manifest(
            source_sha=_git_value(root, "HEAD"),
            source_tree_sha=_git_value(root, "HEAD^{tree}"),
            go_version=go_version,
            frontend_lockfile=root / "internal/site/bun.lock",
            workflow=root / ".github/workflows/ios-build.yml",
            runtime_patch_script=root / ".github/scripts/patch-go-ios-arm64-runtime.py",
            patched_runtime_asm=args.runtime_asm,
            frontend_dist=root / "internal/site/dist",
            primary_build=args.primary_build,
            repeat_build=args.repeat_build,
            target=target,
            toolchain={"xcodebuild": args.xcode_version, "iphoneos_sdk": args.ios_sdk_version},
            build_ids=ids,
            workflow_run_id=args.workflow_run_id,
        )
        args.output.write_text(json.dumps(report, indent=2) + "\n")
    except (OSError, ValueError, subprocess.CalledProcessError, json.JSONDecodeError) as exc:
        raise SystemExit(f"ERROR: cannot create reproducibility record: {exc}") from exc
    print(f"Reproducibility classification: {report['classification']}")
    for name, result in report["builds"].items():
        print(f"{name}: primary={result['primary']['sha256']} repeat={result['repeat']['sha256']} differing_bytes={result['differing_byte_count']}")


if __name__ == "__main__":
    main()
