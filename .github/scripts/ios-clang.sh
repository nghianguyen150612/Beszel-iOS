#!/bin/sh
# Use the device SDK for every compile/link invocation, with explicit failures.
set -eu
sdk_path=$(xcrun --sdk iphoneos --show-sdk-path)
clang_path=$(xcrun --sdk iphoneos --find clang)
case "$sdk_path" in
  */iPhoneOS*.sdk) ;;
  *) echo "ERROR: expected iPhoneOS device SDK: $sdk_path" >&2; exit 1 ;;
esac
[ -d "$sdk_path" ] && [ -x "$clang_path" ] || {
  echo 'ERROR: missing iPhoneOS SDK or Apple clang' >&2; exit 1;
}
# Build steps clear inherited CGO/GOFLAGS. Refuse incompatible explicit arguments
# too; Go's own -arch arm64 is permitted.
previous=
for argument do
  case "$previous:$argument" in
    -arch:arm64) ;;
    -arch:*) echo 'ERROR: incompatible architecture' >&2; exit 1 ;;
  esac
  case "$argument" in
    -target|--target*|-isysroot|--sysroot*|-mmacosx*|-mios-simulator*|-miphoneos-version-min*|-mios-version-min*)
      echo "ERROR: caller overrides iOS target: $argument" >&2; exit 1 ;;
  esac
  previous=$argument
done
unset MACOSX_DEPLOYMENT_TARGET IPHONEOS_DEPLOYMENT_TARGET
SDKROOT=$sdk_path
export SDKROOT
exec "$clang_path" -arch arm64 -isysroot "$sdk_path" -mios-version-min=12.0 "$@"
