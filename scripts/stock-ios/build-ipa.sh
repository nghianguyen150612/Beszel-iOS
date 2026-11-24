#!/usr/bin/env bash
# P1 development archive: statically linked Go c-archive in a SwiftUI app.
# The ad-hoc signature is for local validation only; SideStore must re-sign it.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

for tool in go xcrun xcodebuild codesign otool lipo plutil zip shasum; do
  command -v "$tool" >/dev/null || { echo "Missing required macOS tool: $tool" >&2; exit 1; }
done

SDK="$(xcrun --sdk iphoneos --show-sdk-path)"
IPHONEOS_CLANG_BIN="$(xcrun --sdk iphoneos --find clang)"
export IPHONEOS_CLANG_BIN
export IPHONEOS_SDK_ROOT="$SDK"

TEMP_DIR="$(mktemp -d "${RUNNER_TEMP:-/tmp}/beszel-stock-ios.XXXXXXXX")"
trap 'rm -rf -- "$TEMP_DIR"' EXIT
OUT="$ROOT/build/stock-ios"
APP="$TEMP_DIR/Payload/BeszelStockAgent.app"
LIB="$TEMP_DIR/libbeszelbridge.a"
mkdir -p "$OUT" "$APP"

cat > "$TEMP_DIR/clang-ios.sh" <<'CLANG_WRAPPER'
#!/bin/sh
set -eu
exec "$IPHONEOS_CLANG_BIN" -arch arm64 -isysroot "$IPHONEOS_SDK_ROOT" -mios-version-min=16.0 "$@"
CLANG_WRAPPER
chmod +x "$TEMP_DIR/clang-ios.sh"

echo "Building Go c-archive with an iPhoneOS target (no A7 runtime patch)."
CGO_ENABLED=1 GOOS=ios GOARCH=arm64 CC="$TEMP_DIR/clang-ios.sh" \
  go build -trimpath -buildmode=c-archive -o "$LIB" ./ios-app/GoBridge
test -s "$LIB"
test -s "${LIB%.a}.h"

echo "Compiling SwiftUI host and linking the real Go runtime."
xcrun --sdk iphoneos swiftc \
  -parse-as-library \
  -target arm64-apple-ios16.0 \
  -sdk "$SDK" \
  -import-objc-header ios-app/Bridge/BeszelGoBridge.h \
  ios-app/App/BeszelStockAgentApp.swift \
  ios-app/App/ContentView.swift \
  -Xlinker -force_load -Xlinker "$LIB" \
  -lresolv \
  -o "$APP/BeszelStockAgent"

cp ios-app/Resources/Info.plist "$APP/Info.plist"
plutil -lint "$APP/Info.plist"
file "$APP/BeszelStockAgent"
lipo -info "$APP/BeszelStockAgent" | grep -q 'arm64'
otool -l "$APP/BeszelStockAgent" | grep -q 'LC_BUILD_VERSION'

# This deliberately does not embed a provisioning profile or Apple credentials.
codesign --force --sign - --timestamp=none "$APP"
codesign --verify --verbose=2 "$APP"

IPA="$OUT/BeszelStockAgent-development-adhoc.ipa"
( cd "$TEMP_DIR" && zip -q -r "$IPA" Payload )
unzip -t "$IPA" >/dev/null
( cd "$OUT" && shasum -a 256 "$(basename "$IPA")" > SHA256SUMS )
shasum -a 256 -c "$OUT/SHA256SUMS"
echo "Development archive produced: $IPA"
echo "P1 ONLY: installability, Agent↔Hub transport and on-device execution NOT VALIDATED."
