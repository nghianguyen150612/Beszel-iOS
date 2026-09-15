# iOS Build Notes

This documents the unusual requirements for producing working iOS binaries. It summarizes the two probe workflows; see the YAML itself for exact steps.

## Target

- `GOOS=ios`, `GOARCH=arm64`
- `CGO_ENABLED=1` (required for the `ios/arm64` target)
- Minimum deployment target: iOS 12.0 (`-mios-version-min=12.0`)
- Runner: macOS (`macos-14`) with Xcode iPhoneOS SDK

## Toolchain

1. Apply the Go runtime patch first: `sudo python3 .github/scripts/patch-go-ios-arm64-runtime.py`.
2. Resolve the SDK and compiler: `xcrun --sdk iphoneos --show-sdk-path`, `xcrun --sdk iphoneos --find clang`.
3. Create a clang wrapper passing `-arch arm64 -isysroot <SDK> -mios-version-min=12.0`, and export it as `CC`.
4. Build with the wrapper:
   - Agent: `go build -trimpath -ldflags="-s -w" -o build/beszel-agent-ios-arm64 ./internal/cmd/agent`
   - Hub: `go build -trimpath -ldflags="-s -w" -o build/beszel-hub-ios-arm64 ./internal/cmd/hub`
5. Sanity-check with `file`, `otool -hv`, and `otool -l | grep LC_BUILD_VERSION|LC_VERSION_MIN_IPHONEOS`.

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
