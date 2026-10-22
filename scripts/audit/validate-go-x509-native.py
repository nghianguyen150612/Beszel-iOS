#!/usr/bin/env python3
"""Verify Go 1.27.3 contains the native GO-2026-6809 x509 fix and test it."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys


EXPECTED_GO = "go1.27.3"
GO_ARCHIVE_SHA256 = "65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40"
UPSTREAM_CHANGE = "https://go-review.googlesource.com/c/go/+/851047"
UPSTREAM_COMMIT = "7f86f957633ee31a1c17a1eb8452ca7a3d9bc141"
SOURCE_SHA256 = {
    "src/crypto/x509/root_darwin.go": "69b350fe1afa679935ca831c3beea9833d59d8cd62c6f0b4d02f14526dd46ac1",
    "src/crypto/x509/root_windows.go": "ba910c270b9d096f50ccbf9426c03410ad16bc46a86e0ba4237d27c3381d28d0",
    "src/crypto/x509/verify.go": "a239364c8fd240db693be8ec4f82e84958d9b4e4f265ff3923499e2161a34d86",
}
DENYLIST_HASHES = (
    "ebed6475c2c2d80fcb199f3fe8dd6e6b3ce60b77eb15d4e42c3dec0c6f5b5105",
    "49dd353dc2b919a3770645117d0e60c16115e7d1bc259eefe734b00ea434fe5f",
    "e0254c7fcdc265e2f0f8a428dabb75ab4ae039b4872d62f5ee57f0877d5a8a41",
    "5c0660a2b164910d8ff589c18e14552e9e0ec237dcefe2cb736d41a4811f81a0",
    "43bffa19d6ef651451f6841e019931e96f6f2eaeff28ab4b6f1ce4576de2717b",
    "63ecefee431d49523bc1136c4e72d0206a477086caf606cd5cf025686c109cf8",
    "ec4ffa1ad0ac88081314b7aaffef6a91f97cd57291bb5ba686c5415f5eb4f6b6",
    "92c063d67e8f14db71d288e11266ffdaf69cbf115f7bab14f3bc2f68d25e0e30",
)


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def verify_sources(goroot: Path, version: str, expected: dict[str, str] = SOURCE_SHA256) -> dict:
    if version != EXPECTED_GO:
        raise ValueError(f"unsupported Go version {version!r}; expected {EXPECTED_GO}")
    if not goroot.is_absolute() or goroot.is_symlink() or not goroot.is_dir():
        raise ValueError("GOROOT must be an absolute, non-symlink directory")
    observed = {}
    for relative, digest in expected.items():
        path = goroot / relative
        if path.is_symlink() or not path.is_file():
            raise ValueError(f"Go source missing or symlinked: {relative}")
        actual = sha256(path)
        if actual != digest:
            raise ValueError(f"unknown Go 1.27.3 x509 source for {relative}: {actual}")
        observed[relative] = actual

    verify = (goroot / "src/crypto/x509/verify.go").read_text()
    if "func containsDistrustedSPKIHash" not in verify:
        raise ValueError("native Go x509 verification helper is missing")
    if any(value not in verify for value in DENYLIST_HASHES):
        raise ValueError("native Go x509 certificate denylist is incomplete")
    hooks = {
        "src/crypto/x509/root_darwin.go": "containsDistrustedSPKIHash(chain[0])",
        "src/crypto/x509/root_windows.go": "slices.DeleteFunc(chains, containsDistrustedSPKIHash)",
    }
    for relative, marker in hooks.items():
        if marker not in (goroot / relative).read_text():
            raise ValueError(f"native Go x509 platform hook is missing: {relative}")
    return observed


def validate(goroot: Path, version: str) -> dict:
    sources = verify_sources(goroot, version)
    return {
        "schema": 1,
        "status": "PASS",
        "advisory": "GO-2026-6809",
        "go_version": version,
        "fix_disposition": "native in official Go 1.27.3; custom P019/P022 backport not applied",
        "upstream_change": UPSTREAM_CHANGE,
        "upstream_commit": UPSTREAM_COMMIT,
        "official_linux_amd64_archive_sha256": GO_ARCHIVE_SHA256,
        "native_source_sha256": sources,
        "matches_reviewed_backport_outputs": True,
        "standard_library_test": "pending",
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    try:
        info = json.loads(subprocess.check_output(["go", "env", "-json", "GOROOT", "GOVERSION"], text=True))
        evidence = validate(Path(info["GOROOT"]), info["GOVERSION"])
        test_env = os.environ.copy()
        for key in (
            "GOOS", "GOARCH", "GOARM", "GOARM64", "CGO_ENABLED", "CC", "GOFLAGS",
            "CGO_CFLAGS", "CGO_CPPFLAGS", "CGO_CXXFLAGS", "CGO_LDFLAGS",
        ):
            test_env.pop(key, None)
        test_env["GOTOOLCHAIN"] = "local"
        subprocess.run(["go", "test", "crypto/x509", "-count=1"], check=True, env=test_env)
        evidence["standard_library_test"] = {"command": "go test crypto/x509 -count=1", "status": "PASS"}
        if args.output.is_symlink():
            raise ValueError("evidence output path is symlinked")
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(evidence, indent=2) + "\n")
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError, json.JSONDecodeError) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1
    print(json.dumps(evidence, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
