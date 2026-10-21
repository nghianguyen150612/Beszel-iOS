# Beszel-iOS Build Notes

This documents the unusual requirements for producing working iOS binaries. The primary reference is the consolidated `.github/workflows/ios-build.yml`, which owns both the build and the tag-triggered release. See the YAML itself for exact steps.

## Target

- `GOOS=ios`, `GOARCH=arm64`
- `CGO_ENABLED=1` (required for the `ios/arm64` target)
- Minimum deployment target: iOS 12.0 (`-mios-version-min=12.0`)
- Runner: macOS (`macos-14`) with Xcode iPhoneOS SDK

## Toolchain

1. Use the exact Go version from `go.mod` and Bun 1.4.0. Run the disposable runtime/build-validator tests.
2. In `internal/site`, run `bun install --frozen-lockfile`, `bun run sync`, `bun test`, then `bun run build`. Verify the lock hash is unchanged; restore generated tracked PO text while retaining genuine production assets and compiled catalogs.
3. Run `.github/scripts/check-ios-source-selection.py` to inspect actual Go dependency selection for iOS, macOS and Linux. Run the macOS GPU parser tests and macOS builds with the original toolchain.
4. Copy the **complete** setup-go GOROOT into a job-local directory. Verify its layout/version, select it through GOROOT/PATH and GOTOOLCHAIN=local, and apply `.github/scripts/patch-go-ios-arm64-runtime.py` to the copy. Use a dedicated iOS GOCACHE. Verify the original runtime source hash remains unchanged; no sudo is needed.
5. Use `.github/scripts/ios-clang.sh` as CC for both native builds. It resolves the device SDK and Apple clang with `xcrun --sdk iphoneos`, fails on missing components or explicit incompatible target flags, and passes `-arch arm64 -isysroot <SDK> -mios-version-min=12.0`. Clear inherited host CGO/GOFLAGS.
6. Build Agent and Hub into `build/ios/` with `GOOS=ios GOARCH=arm64 GOARM64=v8.0 CGO_ENABLED=1`, `-trimpath` and `-ldflags="-s -w"`. Repeat with an independent clean cache and compare both outputs.
7. Build separate unstripped validation binaries with the same patched toolchain; run `.github/scripts/verify-ios-runtime-binary.py` to inspect the complete procyield symbol. Keep the production stripping flags.
8. Inspect each output with native `file`, `otool -hv` and `otool -l`. Generate bare-filename SHA256SUMS with `shasum -a 256`, verify both entries, then run `.github/scripts/verify-ios-artifacts.py build/ios --frontend internal/site/dist`. It validates ARM64 MH_EXECUTE, device iOS platform/minimum 12.0, entry point, Go entry package/CGO settings, exact payload and embedded frontend bytes.
9. Upload the review payload containing exactly `beszel-agent-ios-arm64`, `beszel-hub-ios-arm64`, and `SHA256SUMS`. Validation JSON is a separate review artifact. Branch/manual runs skip `release-ios`; this procedure does not invoke signing, release or deployment commands.

See [A7 runtime review](ios-a7-runtime-compatibility.md),
[native validation](ios-native-build-validation.md) and
[structured evidence](ios-build-validation.json). Two identical builds in one
verified environment establish that comparison; reproducibility across different
SDKs, toolchain versions or source commits is not assumed.

## Why `ldid` is needed at install time

Jailbroken iOS still requires pseudo-signing for custom native binaries. After copying to the device:

```sh
chown root:wheel /usr/local/bin/beszel-agent /usr/local/bin/beszel-hub
chmod 755 /usr/local/bin/beszel-agent /usr/local/bin/beszel-hub
ldid -S /usr/local/bin/beszel-agent /usr/local/bin/beszel-hub
```

Install under `/usr/local/bin`; `$HOME` and `/tmp` have previously produced execution/sandbox failures.

## Why the Apple A7 runtime patch exists

The historical Apple A7 / iOS 12 environment faults on the modern Go ARM64
`runtime.procyieldAsm` CNTVCT_EL0 path. The current P019 candidate uses Go
1.27.2; the script accepts only that version's reviewed complete executable
source shape and replaces it with a zero-guarded YIELD loop. Unsupported
versions, unknown layouts and repeat invocation fail without writing. The patch
is applied before every native iOS compilation to an isolated copy.
Source/binary inspection cannot establish SIGILL-free 0.21 execution: physical
A7 acceptance remains P018 / Gate F.

## Hub frontend build requirement

The Hub embeds the web UI. Before compiling the Hub binary:

```sh
cd internal/site
bun install --frozen-lockfile
bun run sync
bun test
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
2. The tagged commit must be an ancestor of `origin/iOS`, so a tag
   accidentally created on unrelated `main` history is rejected without
   publishing anything. Historical rebuilds (tag behind current `iOS` HEAD)
   remain allowed.
3. The job downloads the `beszel-ios-arm64` artifact from its own run and
   re-verifies it: all three files exist and are non-zero, `SHA256SUMS`
   contains exactly the two binary entries, and `sha256sum -c` passes.
4. It publishes with `gh release create <tag> --title "Beszel-iOS <tag>"
   --latest` plus the three files — non-draft, non-prerelease, Latest —
   then asserts the release state and that `/releases/latest` resolves to
   the new tag.

Path filters in the workflow trigger apply to `iOS` branch pushes only;
GitHub does not evaluate path filters for tag pushes, so release tags always
build. Upstream tag automation is scoped away from iOS tags: `release.yml`
(GoReleaser) and `docker-images.yml` both exclude `v*-ios.*`, so an iOS
release never gains upstream assets or Docker builds.

Each release therefore exposes the future installer contract:

- `.../releases/latest/download/beszel-agent-ios-arm64`
- `.../releases/latest/download/beszel-hub-ios-arm64`
- `.../releases/latest/download/SHA256SUMS`

The lifecycle engine (`scripts/ios/install-beszel.sh`, executed by the `install.sh`
bootstrap, v1.0.0) follows the `/releases/latest` redirect
once per run to resolve the current tag (validated against
`v<upstream>-ios.<rev>`), then pins every download for that run to
`.../releases/download/<tag>/...` so `SHA256SUMS` and both binaries always
come from the same immutable release. Successful installs and updates record
the release tag plus the original asset SHA-256 in
`/var/lib/beszel-ios/install-state`; the `ldid`-signed binaries on device are
never hash-compared against `SHA256SUMS`.
