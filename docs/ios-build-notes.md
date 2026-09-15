# iOS Build Notes

This documents the unusual requirements for producing working iOS binaries. The primary reference is the consolidated `.github/workflows/ios-build.yml`; the two probe workflows (`ios-agent-probe.yml`, `ios-hub-probe.yml`) are legacy references kept as fallback. See the YAML itself for exact steps.

## Target

- `GOOS=ios`, `GOARCH=arm64`
- `CGO_ENABLED=1` (required for the `ios/arm64` target)
- Minimum deployment target: iOS 12.0 (`-mios-version-min=12.0`)
- Runner: macOS (`macos-14`) with Xcode iPhoneOS SDK

## Toolchain

1. Apply the Go runtime patch first: `sudo python3 .github/scripts/patch-go-ios-arm64-runtime.py` (workflow fails if this fails).
2. Build the Hub frontend first (`internal/site`: `bun install`, `bun run build`). Do not switch to frozen-lockfile installs without verifying against the current repo.
3. Resolve the SDK and compiler: `xcrun --sdk iphoneos --show-sdk-path`, `xcrun --sdk iphoneos --find clang`. Never hardcode an SDK path.
4. Create one clang wrapper passing `-arch arm64 -isysroot <SDK> -mios-version-min=12.0`, and reuse it as `CC` for both builds.
5. Build with the wrapper into `build/ios/`:
   - Agent: `go build -trimpath -ldflags="-s -w" -o build/ios/beszel-agent-ios-arm64 ./internal/cmd/agent`
   - Hub: `go build -trimpath -ldflags="-s -w" -o build/ios/beszel-hub-ios-arm64 ./internal/cmd/hub`
6. Verify both binaries: `file`, `otool -hv`, `otool -l | grep LC_BUILD_VERSION|LC_VERSION_MIN_IPHONEOS`. The workflow fails on missing/empty files, non-Mach-O output, non-arm64 output, missing iOS load commands, or missing `12.0` deployment metadata.
7. Generate checksums with macOS-compatible `shasum -a 256` (no absolute paths):
   - `shasum -a 256 beszel-agent-ios-arm64 beszel-hub-ios-arm64 > SHA256SUMS`
8. Verify with `shasum -a 256 -c SHA256SUMS`.
9. Upload one `beszel-ios-arm64` artifact with exactly the three files above. No release publishing happens in this workflow.

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
bun install
bun run build
```

Skipping this produces a Hub without the current frontend. The agent build does not need this step.
