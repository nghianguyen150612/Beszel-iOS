#!/usr/bin/env python3
"""Historical P019/P022 helper for Go 1.27.2; P023 verifies the native Go 1.27.3 fix."""

import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys


SUPPORTED_GO = "go1.27.2"
UPSTREAM_CHANGE = "https://go-review.googlesource.com/c/go/+/851047"
UPSTREAM_COMMIT = "7f86f957633ee31a1c17a1eb8452ca7a3d9bc141"
PATCH_SHA256 = "111bf066e27cb60df43881cbfa9b1d22b7682b720628b4b691a960a853b48150"
BASELINE_SHA256 = {
    "src/crypto/x509/root_darwin.go": "e84652a32aa3806e13c4845107b57f5da5a069ec7b91855da4a645d5fa9ea5b5",
    "src/crypto/x509/root_windows.go": "cae38f158aebec723791d96194f30bf1639a4e9e765878bcb5ee4eec5e0a1568",
    "src/crypto/x509/verify.go": "8b3d059f20a75ac8a103d97b473fbe90e01d40654c4548c50d9b0937f54f3c8f",
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
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def load_patch(path: Path) -> tuple[bytes, str]:
    encoded = b"".join(path.read_bytes().split())
    try:
        patch_bytes = base64.b64decode(encoded, validate=True)
    except ValueError as exc:
        raise ValueError("security patch encoding is invalid") from exc
    return patch_bytes, hashlib.sha256(patch_bytes).hexdigest()


def patch_source(goroot: Path, version: str, patch_file: Path) -> dict:
    if version != SUPPORTED_GO:
        raise ValueError(f"unsupported Go version {version!r}; expected {SUPPORTED_GO}")
    if not goroot.is_absolute() or goroot.is_symlink() or not goroot.is_dir():
        raise ValueError("GOROOT must be an absolute, non-symlink directory")
    if patch_file.is_symlink() or not patch_file.is_file():
        raise ValueError("security patch is missing or symlinked")
    patch_bytes, patch_sha = load_patch(patch_file)
    if patch_sha != PATCH_SHA256:
        raise ValueError(f"unexpected GO-2026-6809 patch digest {patch_sha}")

    sources = {}
    for relative, expected in BASELINE_SHA256.items():
        path = goroot / relative
        if path.is_symlink() or not path.is_file():
            raise ValueError(f"Go source missing or symlinked: {relative}")
        actual = sha256(path)
        if actual != expected:
            raise ValueError(f"unknown Go 1.27.2 source for {relative}: {actual}")
        sources[relative] = actual

    # Check the complete official patch before changing any file. The source
    # fingerprints above bind it to the Go 1.27.2 archive reviewed for P019.
    subprocess.run(["patch", "--dry-run", "-p1", "-d", str(goroot)], input=patch_bytes, check=True)
    subprocess.run(["patch", "-p1", "-d", str(goroot)], input=patch_bytes, check=True)

    verify = goroot / "src/crypto/x509/verify.go"
    contents = verify.read_text()
    if "func containsDistrustedSPKIHash" not in contents:
        raise ValueError("upstream verification helper is missing after patch")
    if any(value not in contents for value in DENYLIST_HASHES):
        raise ValueError("the complete upstream certificate denylist is missing")
    for relative, marker in (
        ("src/crypto/x509/root_darwin.go", "containsDistrustedSPKIHash(chain[0])"),
        ("src/crypto/x509/root_windows.go", "slices.DeleteFunc(chains, containsDistrustedSPKIHash)"),
    ):
        if marker not in (goroot / relative).read_text():
            raise ValueError(f"platform verification hook is missing from {relative}")

    patched_sources = {relative: sha256(goroot / relative) for relative in BASELINE_SHA256}
    return {
        "schema": 1,
        "status": "PASS",
        "advisory": "GO-2026-6809",
        "cve": "CVE-2026-111943",
        "go_version": version,
        "upstream_change": UPSTREAM_CHANGE,
        "upstream_commit": UPSTREAM_COMMIT,
        "patch_sha256": patch_sha,
        "denylisted_intermediate_spki_sha256": list(DENYLIST_HASHES),
        "baseline_source_sha256": sources,
        "patched_source_sha256": patched_sources,
        "standard_library_test": "pending",
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--patch", type=Path)
    args = parser.parse_args()
    default_patch = Path(__file__).resolve().parents[2] / "patches/go/1.27.2/GO-2026-6809.patch.b64"
    patch_file = args.patch or default_patch
    try:
        info = json.loads(subprocess.check_output(["go", "env", "-json", "GOROOT", "GOVERSION"], text=True))
        evidence = patch_source(Path(info["GOROOT"]), info["GOVERSION"], patch_file)
        test_env = os.environ.copy()
        for key in (
            "GOOS", "GOARCH", "GOARM64", "CGO_ENABLED", "CC", "GOFLAGS",
            "CGO_CFLAGS", "CGO_CPPFLAGS", "CGO_CXXFLAGS", "CGO_LDFLAGS",
        ):
            test_env.pop(key, None)
        subprocess.run(["go", "test", "crypto/x509", "-count=1"], check=True, env=test_env)
        evidence["standard_library_test"] = {
            "command": "go test crypto/x509 -count=1",
            "status": "PASS",
        }
        args.output.parent.mkdir(parents=True, exist_ok=True)
        if args.output.is_symlink():
            raise ValueError("evidence output path is symlinked")
        args.output.write_text(json.dumps(evidence, indent=2) + "\n")
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError, json.JSONDecodeError) as exc:
        raise SystemExit(f"ERROR: refusing GO-2026-6809 backport: {exc}") from exc
    print(json.dumps(evidence, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
