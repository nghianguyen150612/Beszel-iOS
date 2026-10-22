#!/usr/bin/env python3
"""Static checks for separation between iOS validation and publication events."""

from fnmatch import fnmatchcase
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
WORKFLOWS = ROOT / ".github" / "workflows"

passed = 0
failed = 0


def check(description: str, condition: bool) -> None:
    global passed, failed
    if condition:
        print(f"PASS: {description}")
        passed += 1
    else:
        print(f"FAIL: {description}", file=sys.stderr)
        failed += 1


def section(text: str, header: str, indent: int) -> str:
    lines = text.splitlines()
    start = next((i for i, line in enumerate(lines) if line == " " * indent + header), None)
    if start is None:
        return ""
    selected = [lines[start]]
    for line in lines[start + 1 :]:
        if line.strip() and len(line) - len(line.lstrip()) <= indent:
            break
        selected.append(line)
    return "\n".join(selected)


def read(name: str) -> str:
    return (WORKFLOWS / name).read_text()


ios_build = read("ios-build.yml")
ios_on = section(ios_build, "on:", 0)
ios_release = section(ios_build, "release-ios:", 2)
ios_release_validation = section(ios_build, "release-ios-validate:", 2)
ios_build_job = section(ios_build, "build-ios:", 2)
ios_push = section(ios_on, "push:", 2)
ios_tags = section(ios_push, "tags:", 4)
check("iOS build supports an explicit validation dispatch", "workflow_dispatch:" in ios_on)
check("iOS release tags use a distinct suffix filter", '"v*-ios.*"' in ios_tags)
check(
    "read-only release validation is reachable only on a release-tag push",
    "github.event_name == 'push'" in ios_release_validation
    and "github.ref_type == 'tag'" in ios_release_validation
    and "startsWith(github.ref, 'refs/tags/v')" in ios_release_validation
    and "contains(github.ref_name, '-ios.')" in ios_release_validation,
)
check("publisher requires successful read-only release validation", "needs: release-ios-validate" in ios_release and "github.event_name == 'push'" in ios_release and "github.ref_type == 'tag'" in ios_release)
check("release validation cannot run without a successful native build", "needs: build-ios" in ios_release_validation)
check("top-level iOS build permission is read-only", "contents: read" in section(ios_build, "permissions:", 0))
check("native build job receives no release credentials", "secrets." not in ios_build_job and "GH_TOKEN:" not in ios_build_job)
check("release validation job has read-only repository permission", "contents: read" in ios_release_validation and "contents: write" not in ios_release_validation)
check("release publisher is the only job with write permission", "contents: write" in ios_release and "contents: write" not in ios_release_validation)
check("release credential is scoped to individual publisher steps", "GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}" in ios_release and "    env:\n      GH_TOKEN:" not in ios_release)
check("release payload validator is required before staging", "verify-ios-release-payload.py dist" in ios_release_validation and "upload-artifact@v4" in ios_release_validation)
check("release artifacts are downloaded from this workflow run", "name: beszel-ios-arm64" in ios_release_validation and "name: ios-native-validation" in ios_release_validation and "run-id:" not in ios_release_validation)
check("release is staged as a verified draft before publishing", "--draft" in ios_release and "--verify-tag" in ios_release and "gh release edit \"${TAG}\" --draft=false --latest" in ios_release)
check("draft assets are downloaded and revalidated before publication", "gh release download \"${TAG}\"" in ios_release and "verify-ios-release-payload.py \"$VERIFY_DIR\"" in ios_release)
check("release workflow makes no physical-device validation claim", "Validated on: iPad mini 2" not in ios_release)
check("iOS release job does not use mutable Latest downloads", "releases/latest/download" not in ios_release and "latest/download" not in ios_release_validation)
check("release token is not echoed or enabled with shell tracing", "echo \"$GH_TOKEN\"" not in ios_release and "set -eux" not in ios_release)
installer_engine = (ROOT / "scripts" / "ios" / "install-beszel.sh").read_text()
check("public installer continues to resolve only published GitHub Latest", 'RELEASE_LATEST_PAGE="https://github.com/nghianguyen150612/beszel-ios/releases/latest"' in installer_engine and 'RELEASE_BASE="https://github.com/nghianguyen150612/beszel-ios/releases/latest/download"' in installer_engine)
check(
    "native build runs installer Mach-O validation on both generated binaries",
    "sh tests/ios-macho-validation-test.sh build/ios/beszel-agent-ios-arm64 build/ios/beszel-hub-ios-arm64" in ios_build,
)

for filename in ("release.yml", "docker-images.yml"):
    workflow = read(filename)
    events = section(workflow, "on:", 0)
    push = section(events, "push:", 2)
    tags = section(push, "tags:", 4)
    ordered_patterns = [line.strip() for line in tags.splitlines() if line.strip().startswith("- ")]
    check(f"{filename} has no PR or branch publication trigger", "pull_request:" not in events and "branches:" not in events)
    check(
        f"{filename} puts the iOS exclusion after the upstream tag pattern",
        ordered_patterns == ['- "v*"', '- "!v*-ios.*"'],
    )

release = read("release.yml")
check("GoReleaser publication is tag-triggered", "args: release --clean" in release and "pull_request:" not in section(release, "on:", 0))

docker = read("docker-images.yml")
check("Docker pushes are limited to tag refs", "push: ${{ github.ref_type == 'tag' }}" in docker)

helm = read("helm-charts.yml")
helm_events = section(helm, "on:", 0)
check("Helm event filters are limited to chart paths", "supplemental/helm/**" in helm_events)
check("Helm publication step is restricted to push events", "if: github.event_name == 'push'" in helm)

installer = read("ios-installer.yml")
check("installer workflow has read-only repository permission", "permissions:\n  contents: read" in installer and "packages: write" not in installer)

ios_tag = "v0.21.0-ios.1"
upstream_tag = "v0.21.0"
check(
    "iOS tag matches the native tag pattern and the explicit upstream exclusion",
    fnmatchcase(ios_tag, "v*-ios.*") and fnmatchcase(ios_tag, "v*"),
)
check(
    "upstream tag matches upstream workflows and not the iOS pattern",
    fnmatchcase(upstream_tag, "v*") and not fnmatchcase(upstream_tag, "v*-ios.*"),
)


def release_event_allowed(event_name, ref_type, ref, ref_name):
    return event_name == "push" and ref_type == "tag" and ref.startswith("refs/tags/v") and "-ios." in ref_name


for description, event_name, ref_type, ref, ref_name, expected in (
    ("ordinary branch push cannot publish", "push", "branch", "refs/heads/iOS", "iOS", False),
    ("pull request cannot publish", "pull_request", "branch", "refs/pull/26/merge", "26/merge", False),
    ("workflow dispatch on a release tag cannot publish", "workflow_dispatch", "tag", "refs/tags/v0.21.0-ios.1", "v0.21.0-ios.1", False),
    ("P022 branch push cannot publish", "push", "branch", "refs/heads/ios/p022-release-readiness", "ios/p022-release-readiness", False),
    ("ordinary stable tag cannot publish as iOS", "push", "tag", "refs/tags/v0.21.0", "v0.21.0", False),
    ("correctly scoped iOS release tag push can validate for publication", "push", "tag", "refs/tags/v0.21.0-ios.1", "v0.21.0-ios.1", True),
):
    check(description, release_event_allowed(event_name, ref_type, ref, ref_name) is expected)

print(f"\n{passed} passed, {failed} failed")
raise SystemExit(1 if failed else 0)
