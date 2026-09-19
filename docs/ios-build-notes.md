# iOS Build Notes

[Tiếng Việt](ios-build-notes.vi.md)

This documents the unusual requirements for producing working iOS binaries. The primary reference is the consolidated `.github/workflows/ios-build.yml`, which owns both the build and the tag-triggered release. See the YAML itself for exact steps.

## Target

- `GOOS=ios`, `GOARCH=arm64`
- `CGO_ENABLED=1` (required for the `ios/arm64` target)
- Minimum deployment target: iOS 12.0 (`-mios-version-min=12.0`)
- Runner: macOS (`macos-14`) with Xcode iPhoneOS SDK

## Toolchain

1. Apply the Go runtime patch first: `sudo python3 .github/scripts/patch-go-ios-arm64-runtime.py` (workflow fails if this fails).
2. Build the Hub frontend first with pinned Bun 1.4.0 (`internal/site`: `bun install --frozen-lockfile`, `bun run build`). The checked-in `bun.lock` reflects the `package.json` overrides, so frozen installs are required and reproducible from a clean checkout.
3. Resolve the SDK and compiler: `xcrun --sdk iphoneos --show-sdk-path`, `xcrun --sdk iphoneos --find clang`. Never hardcode an SDK path.
4. Create one clang wrapper passing `-arch arm64 -isysroot <SDK> -mios-version-min=12.0`, and reuse it as `CC` for both builds.
5. Build with the wrapper into `build/ios/`:
   - Agent: `go build -trimpath -ldflags="-s -w" -o build/ios/beszel-agent-ios-arm64 ./internal/cmd/agent`
   - Hub: `go build -trimpath -ldflags="-s -w" -o build/ios/beszel-hub-ios-arm64 ./internal/cmd/hub`
6. Verify both binaries: `file`, `otool -hv`, `otool -l | grep LC_BUILD_VERSION|LC_VERSION_MIN_IPHONEOS`. The workflow fails on missing/empty files, non-Mach-O output, non-arm64 output, missing iOS load commands, or missing `12.0` deployment metadata.
7. Generate checksums with macOS-compatible `shasum -a 256` (no absolute paths):
   - `shasum -a 256 beszel-agent-ios-arm64 beszel-hub-ios-arm64 > SHA256SUMS`
8. Verify with `shasum -a 256 -c SHA256SUMS`.
9. Upload one `beszel-ios-arm64` artifact with exactly the three files above. On branch pushes and manual dispatches the workflow stops here; on `v*-ios.*` tag pushes the dependent `release-ios` job continues (see Releases below).

## Why `ldid` is needed at install time

Jailbroken iOS still requires pseudo-signing for custom native binaries. After copying to the device:

```sh
chown root:wheel /usr/local/bin/beszel-agent /usr/local/bin/beszel-hub
chmod 755 /usr/local/bin/beszel-agent /usr/local/bin/beszel-hub
ldid -S /usr/local/bin/beszel-agent /usr/local/bin/beszel-hub
```

Install under `/usr/local/bin`; `$HOME` and `/tmp` have previously produced execution/sandbox failures.

## Why the Apple A7 runtime patch exists

On Apple A7 / iOS 12, the modern Go ARM64 `runtime.procyieldAsm` implementation reads `CNTVCT_EL0`, which faults (`SIGILL`) in this legacy userspace. The patch script replaces that function body with a legacy `YIELD` spin loop and refuses to run if the expected `CNTVCT_EL0` sequence is absent (so it fails loudly on unknown Go versions rather than corrupting the toolchain). It must run before every iOS build for the validated target. Newer SoCs/iOS versions may not need it — that is untested.

## Hub frontend build requirement

The Hub embeds the web UI. Before compiling the Hub binary:

```sh
cd internal/site
bun install --frozen-lockfile
bun run build
```

Bun is pinned to 1.4.0 in `.github/workflows/ios-build.yml`
(`oven-sh/setup-bun@v2` with `bun-version: 1.4.0`). The macOS CI build
remains authoritative for actual iPhoneOS binaries.

Skipping this produces a Hub without the current frontend. The agent build does not need this step.

## Releases

Pushing a tag of the form `v<upstream-version>-ios.<revision>` (for example
`v0.19.0-ios.1`) triggers the same workflow and additionally runs the
`release-ios` job (`ubuntu-latest`, `contents: write`), which depends on the
macOS build job:

1. The tag must match `v<beszel.Version>-ios.<positive integer>`, where
   `beszel.Version` is read from `beszel.go` at the tagged commit — not
   hardcoded in the workflow. The `beszel.Version` constant itself keeps
   reporting the upstream base version (no `-ios.N` suffix in code).
   When upstream Beszel moves to a new version, the iOS revision resets
   (for example `v0.20.0-ios.1`).
2. The tagged commit must be an ancestor of `origin/ios`, so a tag
   accidentally created on unrelated `main` history is rejected without
   publishing anything. Historical rebuilds (tag behind current `ios` HEAD)
   remain allowed.
3. The job downloads the `beszel-ios-arm64` artifact from its own run and
   re-verifies it: all three files exist and are non-zero, `SHA256SUMS`
   contains exactly the two binary entries, and `sha256sum -c` passes.
4. It publishes with `gh release create <tag> --title "Beszel iOS <tag>"
   --latest` plus the three files — non-draft, non-prerelease, Latest —
   then asserts the release state and that `/releases/latest` resolves to
   the new tag.

Path filters in the workflow trigger apply to `ios` branch pushes only;
GitHub does not evaluate path filters for tag pushes, so release tags always
build. Upstream tag automation is scoped away from iOS tags: `release.yml`
(GoReleaser) and `docker-images.yml` both exclude `v*-ios.*`, so an iOS
release never gains upstream assets or Docker builds.

Each release therefore exposes the future installer contract:

- `.../releases/latest/download/beszel-agent-ios-arm64`
- `.../releases/latest/download/beszel-hub-ios-arm64`
- `.../releases/latest/download/SHA256SUMS`

The installer (`install.sh`, v1.0.0) follows the `/releases/latest` redirect
once per run to resolve the current tag (validated against
`v<upstream>-ios.<rev>`), then pins every download for that run to
`.../releases/download/<tag>/...` so `SHA256SUMS` and both binaries always
come from the same immutable release. Successful installs and updates record
the release tag plus the original asset SHA-256 in
`/var/lib/beszel-ios/install-state`; the `ldid`-signed binaries on device are
never hash-compared against `SHA256SUMS`.
