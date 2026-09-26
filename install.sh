#!/bin/sh
# Beszel-iOS pipe-safe bootstrap.
#
# Intentionally small: no install lifecycle logic lives here. This script
# downloads the canonical lifecycle engine, verifies it against the SHA-256
# pinned below, and only then executes the verified local copy as manager.sh
# with this run's arguments. Order is always download -> verify -> execute;
# the engine is never executed from a network pipe and never executed before
# its checksum matches.
#
# The pinned checksum makes engine selection deterministic for a fetched
# bootstrap and closes the mutable-branch race: if the iOS branch changes
# between fetching this bootstrap and fetching the engine, the digests no
# longer match and the run aborts before execution. CI fails whenever the
# engine changes without engine_sha being updated in the same commit.
#
# Scope (deliberately not a release-signing claim): for a given fetched
# bootstrap this gives deterministic engine selection, TOCTOU protection
# between bootstrap fetch and engine fetch, and a persistent manager that is
# byte-for-byte the verified engine that ran the transaction. The bootstrap
# itself is still fetched over HTTPS from the iOS branch, so this does not
# protect against an attacker who can replace the bootstrap. Re-running this
# bootstrap is the supported way to adopt a newer engine; a normal
# beszel-ios update refreshes Beszel application binaries only.
#
# Lifecycle implementation: scripts/ios/install-beszel.sh.
set -eu

umask 077
PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin
export PATH

engine_sha=d4b4e381a76c75ac79cb7c99558853b62c7c2163879a72c0333acc5c40092d0c
engine_url=https://raw.githubusercontent.com/nghianguyen150612/Beszel-iOS/iOS/scripts/ios/install-beszel.sh

fail() {
	printf 'install.sh: %s\n' "$1" >&2
	exit 1
}

command -v curl > /dev/null 2>&1 ||
	fail 'curl is required to download the Beszel-iOS lifecycle engine.'

# First available SHA-256 implementation: verification is never skipped, and
# no package manager is invoked from this bootstrap.
hash_tool=""
if command -v sha256sum > /dev/null 2>&1; then
	hash_tool=sha256sum
elif command -v shasum > /dev/null 2>&1; then
	hash_tool=shasum
elif command -v openssl > /dev/null 2>&1; then
	hash_tool=openssl
else
	fail 'a SHA-256 tool (sha256sum, shasum or openssl) is required to verify the lifecycle engine.'
fi

# Fail closed on a corrupted/malformed pin instead of comparing against it.
[ "${#engine_sha}" -eq 64 ] || fail 'pinned lifecycle engine digest is malformed.'
case "$engine_sha" in
	*[!0-9a-f]*) fail 'pinned lifecycle engine digest is malformed.' ;;
esac

# engine_sha_of <file> — lowercase hex digest from the selected tool.
engine_sha_of() {
	_so_out=""
	case "$hash_tool" in
		sha256sum)
			_so_out=$(sha256sum "$1") || return 1
			_so_out=${_so_out%% *}
			;;
		shasum)
			_so_out=$(shasum -a 256 "$1") || return 1
			_so_out=${_so_out%% *}
			;;
		openssl)
			_so_out=$(openssl dgst -sha256 "$1") || return 1
			_so_out=${_so_out##* }
			;;
		*) return 1 ;;
	esac
	printf '%s' "$_so_out" | tr 'A-F' 'a-f'
	return 0
}

# Private staging: umask 077 plus this run's own directory. Cleanup removes
# only the file this bootstrap created inside its own directory, then the
# directory itself — never a recursive delete, never a caller-controlled path.
if command -v mktemp > /dev/null 2>&1; then
	work=$(mktemp -d /tmp/beszel-installer.XXXXXX) ||
		fail 'cannot create a private temporary directory.'
else
	work="/tmp/beszel-installer.$$"
	[ ! -e "$work" ] && mkdir "$work" || fail 'cannot create a private temporary directory.'
fi

cleanup() {
	rm -f "$work/manager.sh"
	rmdir "$work" 2> /dev/null || true
}
trap cleanup 0
trap 'exit 130' INT
trap 'exit 143' TERM HUP

# Download first: HTTPS only, no protocol downgrade on redirect, no pipe.
if ! curl \
	--proto '=https' \
	--proto-redir '=https' \
	-fsSL \
	--retry 3 \
	--retry-delay 2 \
	--connect-timeout 20 \
	--max-time 120 \
	"$engine_url" \
	-o "$work/manager.sh"; then
	fail "could not download the lifecycle engine from ${engine_url}."
fi
[ -s "$work/manager.sh" ] || fail 'downloaded lifecycle engine is empty.'

# Verify second: a digest mismatch (the branch moved between bootstrap fetch
# and engine fetch) aborts here, before any engine code runs.
actual_sha=$(engine_sha_of "$work/manager.sh") || fail 'SHA-256 verification could not run.'
if [ "$actual_sha" != "$engine_sha" ]; then
	fail "lifecycle engine checksum mismatch (expected ${engine_sha}, got ${actual_sha}); refusing to execute it."
fi

# Execute third: the verified local file, named manager.sh so the engine
# recognises it as the eligible source for persistent manager installation.
/bin/sh "$work/manager.sh" "$@"
