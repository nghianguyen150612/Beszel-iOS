#!/bin/sh
# Test the installer's POSIX Mach-O pre-install gate with structural fixtures.
# Optional arguments are genuine Agent and Hub iOS binaries from a CI artifact.
set -eu

TEST_DIR=$(dirname "$0")
SCRIPT="${TEST_DIR}/../scripts/ios/install-beszel.sh"
export BESZEL_INSTALL_LIB_ONLY=1
# shellcheck disable=SC1090,SC1091
. "$SCRIPT"

TEST_TMP=$(mktemp -d "${TMPDIR:-/tmp}/beszel-macho-test.XXXXXX")
trap 'rm -rf "$TEST_TMP"' EXIT INT TERM HUP

pass_count=0
fail_count=0
pass() { printf 'PASS: %s\n' "$1"; pass_count=$((pass_count + 1)); }
fail() { printf 'FAIL: %s\n' "$1"; fail_count=$((fail_count + 1)); }

make_fixture() {
	python3 - "$@" <<'PY'
import struct
import sys

path, platform, minos, cpu = sys.argv[1], int(sys.argv[2], 0), int(sys.argv[3], 0), int(sys.argv[4], 0)
subtype = int(sys.argv[5], 0) if len(sys.argv) > 5 else 0
header = struct.pack("<IIIIIIII", 0xfeedfacf, cpu, subtype, 2, 1, 24, 0, 0)
command = struct.pack("<IIIIII", 0x32, 24, platform, minos, minos, 0)
with open(path, "wb") as out:
    out.write(header + command)
PY
}

make_fixture "${TEST_TMP}/ios-arm64" 2 0x000c0000 0x0100000c
if validate_ios_arm64_macho "${TEST_TMP}/ios-arm64"; then pass "iOS ARM64 minimum iOS 12.0 accepted"; else fail "iOS ARM64 minimum iOS 12.0 accepted"; fi

make_fixture "${TEST_TMP}/ios-arm64-older" 2 0x000b0000 0x0100000c
if validate_ios_arm64_macho "${TEST_TMP}/ios-arm64-older"; then pass "iOS ARM64 older deployment target accepted"; else fail "iOS ARM64 older deployment target accepted"; fi

make_fixture "${TEST_TMP}/simulator" 7 0x000c0000 0x0100000c
if validate_ios_arm64_macho "${TEST_TMP}/simulator"; then fail "iOS simulator binary rejected"; else pass "iOS simulator binary rejected"; fi

make_fixture "${TEST_TMP}/macos" 1 0x000c0000 0x0100000c
if validate_ios_arm64_macho "${TEST_TMP}/macos"; then fail "macOS binary rejected"; else pass "macOS binary rejected"; fi

make_fixture "${TEST_TMP}/newer-ios" 2 0x000d0000 0x0100000c
if validate_ios_arm64_macho "${TEST_TMP}/newer-ios"; then fail "unsupported minimum iOS target rejected"; else pass "unsupported minimum iOS target rejected"; fi

make_fixture "${TEST_TMP}/wrong-cpu" 2 0x000c0000 0x01000007
if validate_ios_arm64_macho "${TEST_TMP}/wrong-cpu"; then fail "wrong Mach-O architecture rejected"; else pass "wrong Mach-O architecture rejected"; fi

make_fixture "${TEST_TMP}/arm64e" 2 0x000c0000 0x0100000c 2
if validate_ios_arm64_macho "${TEST_TMP}/arm64e"; then fail "unsupported ARM64 subtype rejected"; else pass "unsupported ARM64 subtype rejected"; fi

printf 'truncated' > "${TEST_TMP}/truncated"
if validate_ios_arm64_macho "${TEST_TMP}/truncated"; then fail "truncated Mach-O rejected"; else pass "truncated Mach-O rejected"; fi

printf 'not a Mach-O executable' > "${TEST_TMP}/invalid"
if validate_ios_arm64_macho "${TEST_TMP}/invalid"; then fail "invalid Mach-O rejected"; else pass "invalid Mach-O rejected"; fi

if [ "$#" -ge 2 ]; then
	if validate_ios_arm64_macho "$1"; then pass "genuine P015 Agent iOS ARM64 artifact accepted"; else fail "genuine P015 Agent iOS ARM64 artifact accepted"; fi
	if validate_ios_arm64_macho "$2"; then pass "genuine P015 Hub iOS ARM64 artifact accepted"; else fail "genuine P015 Hub iOS ARM64 artifact accepted"; fi
fi

printf '\n%d passed, %d failed\n' "$pass_count" "$fail_count"
[ "$fail_count" -eq 0 ]
