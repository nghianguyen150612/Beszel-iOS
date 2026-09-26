#!/bin/sh
# Tests for the pipe-safe root bootstrap (install.sh).
#
# The bootstrap is the documented entry point:
#
#   curl -fsSL https://raw.githubusercontent.com/nghianguyen150612/Beszel-iOS/iOS/install.sh | sudo sh
#
# These tests pin down its contract: it is a tiny POSIX script that downloads
# the lifecycle engine (scripts/ios/install-beszel.sh) over HTTPS, verifies
# the SHA-256 pinned inside the bootstrap, and only then executes the
# verified local copy as manager.sh with the caller's arguments. The
# mandatory check is the static pin: engine_sha must equal the digest of the
# engine committed in the same repository state (CI re-checks it with
# sha256sum -c).
#
# Behavioral cases use a test-only fixture copy of the bootstrap whose PATH
# line points at a curl stub; the production file keeps its fixed PATH and
# its HTTPS-only curl policy, and is never modified by these tests.
#
# Usage:  sh tests/bootstrap-test.sh

set -u

# shellcheck disable=SC1007
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
# shellcheck disable=SC1007
REPO_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"
BOOTSTRAP="$REPO_ROOT/install.sh"
ENGINE="$REPO_ROOT/scripts/ios/install-beszel.sh"
CANONICAL_ENGINE_URL='https://raw.githubusercontent.com/nghianguyen150612/Beszel-iOS/iOS/scripts/ios/install-beszel.sh'

_pass=0
_fail=0

pass() { printf 'PASS: %s\n' "$1"; _pass=$((_pass + 1)); }
fail() { printf 'FAIL: %s\n' "$1" >&2; _fail=$((_fail + 1)); }

assert_eq() {
	if [ "$2" = "$3" ]; then
		pass "$1"
	else
		fail "$1 (expected [$2], got [$3])"
	fi
}

SANDBOX=$(mktemp -d "${TMPDIR:-/tmp}/beszel-bootstrap-test.XXXXXX") || exit 1
# shellcheck disable=SC2329
trap 'rm -rf "$SANDBOX"' EXIT INT TERM HUP

if [ ! -f "$BOOTSTRAP" ]; then
	fail "bootstrap exists"
	printf '\n%d passed, %d failed\n' "$_pass" "$_fail"
	exit 1
fi
if [ -f "$ENGINE" ]; then
	pass "lifecycle engine exists at scripts/ios/install-beszel.sh"
else
	fail "lifecycle engine exists at scripts/ios/install-beszel.sh"
	printf '\n%d passed, %d failed\n' "$_pass" "$_fail"
	exit 1
fi

# digest_of <file> — lowercase hex digest using the first available tool.
digest_of() {
	if command -v sha256sum > /dev/null 2>&1; then
		sha256sum "$1" | cut -d' ' -f1
	elif command -v shasum > /dev/null 2>&1; then
		shasum -a 256 "$1" | cut -d' ' -f1
	elif command -v openssl > /dev/null 2>&1; then
		openssl dgst -sha256 "$1" | sed 's/.*= //'
	else
		return 1
	fi
}

printf '== bootstrap syntax and shape ==\n'
if sh -n "$BOOTSTRAP" > /dev/null 2>&1; then
	pass "bootstrap passes sh -n (POSIX syntax)"
else
	fail "bootstrap passes sh -n (POSIX syntax)"
fi
assert_eq "bootstrap shebang is /bin/sh" "#!/bin/sh" "$(head -n 1 "$BOOTSTRAP")"
if grep -q '^set -eu$' "$BOOTSTRAP"; then
	pass "bootstrap runs with set -eu"
else
	fail "bootstrap runs with set -eu"
fi
_boot_lines=$(wc -l < "$BOOTSTRAP" | tr -d '[:space:]')
if [ "$_boot_lines" -le 130 ]; then
	pass "bootstrap stays small (${_boot_lines} lines)"
else
	fail "bootstrap stays small (${_boot_lines} lines; contains lifecycle code?)"
fi
if sed 's/#.*//' "$BOOTSTRAP" | grep -nE '(^|[^A-Za-z0-9_])(local|declare|function)[[:space:]]' > /dev/null; then
	fail "bootstrap code uses no local/declare/function keywords"
else
	pass "bootstrap code uses no local/declare/function keywords"
fi
if sed 's/#.*//' "$BOOTSTRAP" | grep -nwE 'eval|source' > /dev/null; then
	fail "bootstrap code uses no eval or source"
else
	pass "bootstrap code uses no eval or source"
fi
if sed 's/#.*//' "$BOOTSTRAP" | grep -n '`' > /dev/null; then
	fail "bootstrap code has no backtick substitution"
else
	pass "bootstrap code has no backtick substitution"
fi
if grep -nE 'apt-get|[[:space:]]apt[[:space:]]|brew|apk|yum|dpkg' "$BOOTSTRAP" > /dev/null; then
	fail "bootstrap never invokes a package manager"
else
	pass "bootstrap never invokes a package manager"
fi

printf '== bootstrap engine pin ==\n'
_engine_url=$(sed -n 's/^engine_url=//p' "$BOOTSTRAP")
assert_eq "engine URL is canonical" "$CANONICAL_ENGINE_URL" "$_engine_url"
if printf '%s\n' "$_engine_url" | grep -q 'Beszel-iOS/iOS/'; then
	pass "engine URL uses the canonical iOS branch casing"
else
	fail "engine URL uses the canonical iOS branch casing"
fi
_engine_sha=$(sed -n 's/^engine_sha=//p' "$BOOTSTRAP")
if printf '%s\n' "$_engine_sha" | grep -Eq '^[0-9a-f]{64}$'; then
	pass "engine_sha is exactly 64 lowercase hex characters"
else
	fail "engine_sha is exactly 64 lowercase hex characters (got [${_engine_sha}])"
fi
_engine_sha_lines=$(grep -c '^engine_sha=' "$BOOTSTRAP")
assert_eq "bootstrap pins exactly one engine_sha" "1" "$_engine_sha_lines"
_engine_url_lines=$(grep -c '^engine_url=' "$BOOTSTRAP")
assert_eq "bootstrap pins exactly one engine_url" "1" "$_engine_url_lines"
if [ "$_engine_sha" = "$(digest_of "$ENGINE")" ]; then
	pass "engine_sha matches the committed lifecycle engine"
else
	fail "engine_sha matches the committed lifecycle engine (pin [${_engine_sha}], actual [$(digest_of "$ENGINE")])"
fi

printf '== bootstrap transport policy ==\n'
if grep -Fq -- "--proto '=https'" "$BOOTSTRAP"; then
	pass "curl restricts the request protocol to HTTPS"
else
	fail "curl restricts the request protocol to HTTPS"
fi
if grep -Fq -- "--proto-redir '=https'" "$BOOTSTRAP"; then
	pass "curl restricts redirects to HTTPS (no downgrade)"
else
	fail "curl restricts redirects to HTTPS (no downgrade)"
fi
if grep -q 'http://' "$BOOTSTRAP"; then
	fail "bootstrap contains no plain-HTTP URL"
else
	pass "bootstrap contains no plain-HTTP URL"
fi
if grep -F '/bin/sh "$work/manager.sh" "$@"' "$BOOTSTRAP" | grep -q '|'; then
	fail "engine execution line is not a pipeline target"
else
	pass "engine execution line is not a pipeline target"
fi
if sed 's/#.*//' "$BOOTSTRAP" | grep -E 'curl[^|]*\|[^|]*(sh|bash|zsh)' > /dev/null; then
	fail "engine is never executed from a network pipe"
else
	pass "engine is never executed from a network pipe"
fi
if grep -Fq -- '--connect-timeout' "$BOOTSTRAP" && grep -Fq -- '--max-time' "$BOOTSTRAP"; then
	pass "curl uses explicit connection and overall timeouts"
else
	fail "curl uses explicit connection and overall timeouts"
fi

printf '== bootstrap ordering: download, verify, execute ==\n'
_dl_line=$(grep -n -- '-o "$work/manager.sh"' "$BOOTSTRAP" | head -n 1 | cut -d: -f1)
_verify_line=$(grep -n 'actual_sha=$(engine_sha_of' "$BOOTSTRAP" | head -n 1 | cut -d: -f1)
_cmp_line=$(grep -n 'actual_sha" != "$engine_sha"' "$BOOTSTRAP" | head -n 1 | cut -d: -f1)
_exec_line=$(grep -n '/bin/sh "$work/manager.sh" "$@"' "$BOOTSTRAP" | head -n 1 | cut -d: -f1)
if [ -n "$_dl_line" ] && [ -n "$_verify_line" ] && [ -n "$_cmp_line" ] && [ -n "$_exec_line" ] &&
	[ "$_dl_line" -lt "$_verify_line" ] && [ "$_verify_line" -lt "$_cmp_line" ] && [ "$_cmp_line" -lt "$_exec_line" ]; then
	pass "ordering is download (line $_dl_line) -> verify (line $_verify_line/_cmp_line) -> execute (line $_exec_line)"
else
	fail "ordering is download -> verify -> execute (dl=${_dl_line:-missing} verify=${_verify_line:-missing} cmp=${_cmp_line:-missing} exec=${_exec_line:-missing})"
fi
if grep -Fq '/bin/sh "$work/manager.sh" "$@"' "$BOOTSTRAP"; then
	pass "verified local engine file is executed by /bin/sh as manager.sh"
else
	fail "verified local engine file is executed by /bin/sh as manager.sh"
fi
if grep -Fq '"$@"' "$BOOTSTRAP" && [ -n "$_exec_line" ]; then
	pass "bootstrap forwards its arguments to the engine"
else
	fail "bootstrap forwards its arguments to the engine"
fi
if grep -Fq 'refusing to execute it' "$BOOTSTRAP"; then
	pass "checksum mismatch aborts before execution with a clear error"
else
	fail "checksum mismatch aborts before execution with a clear error"
fi

printf '== bootstrap staging and cleanup ==\n'
if grep -q '^umask 077$' "$BOOTSTRAP"; then
	pass "bootstrap stages with umask 077"
else
	fail "bootstrap stages with umask 077"
fi
if grep -Fq 'mktemp -d /tmp/beszel-installer.XXXXXX' "$BOOTSTRAP"; then
	pass "bootstrap creates a private temporary directory"
else
	fail "bootstrap creates a private temporary directory"
fi
if grep -q 'trap cleanup 0' "$BOOTSTRAP" &&
	grep -Fq 'rm -f "$work/manager.sh"' "$BOOTSTRAP" &&
	grep -Fq 'rmdir "$work"' "$BOOTSTRAP"; then
	pass "bootstrap removes only its own staged file and directory"
else
	fail "bootstrap removes only its own staged file and directory"
fi
if grep -q 'rm -rf' "$BOOTSTRAP"; then
	fail "bootstrap performs no recursive delete"
else
	pass "bootstrap performs no recursive delete"
fi
if grep -q "trap 'exit 130' INT" "$BOOTSTRAP" && grep -q "trap 'exit 143' TERM HUP" "$BOOTSTRAP"; then
	pass "bootstrap handles INT/TERM/HUP conservatively"
else
	fail "bootstrap handles INT/TERM/HUP conservatively"
fi

printf '== bootstrap contains no lifecycle implementation ==\n'
for _token in INSTALLER_VERSION BESZEL_IOS_MANAGER_SOURCE_V1 launchctl dev.beszel do_install_agent flow_agent install_persistent_manager print_menu resolve_latest_tag ldid; do
	if grep -q "$_token" "$BOOTSTRAP"; then
		fail "bootstrap contains no lifecycle token '${_token}'"
	else
		pass "bootstrap contains no lifecycle token '${_token}'"
	fi
done
if grep -q 'BESZEL_IOS_MANAGER_SOURCE_V1' "$BOOTSTRAP"; then
	fail "bootstrap cannot be mistaken for manager source"
else
	pass "bootstrap cannot be mistaken for manager source"
fi

_tmp_before=$(ls -d /tmp/beszel-installer.* 2> /dev/null || true)

printf '== bootstrap behavior: mismatch is rejected before execution ==\n'
# Test-only fixture: a copy of the real bootstrap whose PATH line points at a
# curl stub (the production bootstrap resets PATH before calling curl). The
# stub "downloads" a fixture engine instead of reaching the network.
SHIM="$SANDBOX/shim"
mkdir -p "$SHIM"
cat > "$SHIM/curl" << 'SHIMEOF'
#!/bin/sh
_dest=""
_prev=""
for _arg in "$@"; do
	if [ "$_prev" = "-o" ]; then _dest="$_arg"; fi
	_prev="$_arg"
done
printf '%s\n' "$*" >> "$BESZEL_BOOTSTRAP_CURL_LOG"
[ -n "$_dest" ] || exit 2
/bin/cp "$BESZEL_BOOTSTRAP_FIXTURE_ENGINE" "$_dest"
exit 0
SHIMEOF
chmod +x "$SHIM/curl"

FIXTURE_ENGINE="$SANDBOX/fixture-engine.sh"
cat > "$FIXTURE_ENGINE" << 'ENGEOF'
#!/bin/sh
printf 'ENGINE_EXECUTED\n' >> "$BESZEL_BOOTSTRAP_MARKER"
printf 'zero=%s\n' "$0" >> "$BESZEL_BOOTSTRAP_MARKER"
printf 'argc=%s\n' "$#" >> "$BESZEL_BOOTSTRAP_MARKER"
for _arg in "$@"; do
	printf 'arg=%s\n' "$_arg" >> "$BESZEL_BOOTSTRAP_MARKER"
done
exit 0
ENGEOF

# fixture_bootstrap <engine_sha> <output-path> [path-value]
fixture_bootstrap() {
	sed -e "s|^PATH=.*|PATH=${3:-$SHIM:/usr/bin:/bin}|" \
		-e "s|^engine_sha=.*|engine_sha=$1|" \
		"$BOOTSTRAP" > "$2"
}

export BESZEL_BOOTSTRAP_FIXTURE_ENGINE="$FIXTURE_ENGINE"

# Case 1: the pinned digest belongs to the real engine while the transport
# delivers a different file -> verification fails and the engine is NOT run.
MISMATCH_MARKER="$SANDBOX/marker-mismatch"
MISMATCH_LOG="$SANDBOX/curl-mismatch.log"
export BESZEL_BOOTSTRAP_MARKER="$MISMATCH_MARKER"
export BESZEL_BOOTSTRAP_CURL_LOG="$MISMATCH_LOG"
fixture_bootstrap "$_engine_sha" "$SANDBOX/bootstrap-mismatch.sh"
: > "$MISMATCH_LOG"
rm -f "$MISMATCH_MARKER"
if sh "$SANDBOX/bootstrap-mismatch.sh" alpha "beta gamma" > "$SANDBOX/mismatch.out" 2>&1; then
	_mismatch_rc=0
else
	_mismatch_rc=$?
fi
if [ "$_mismatch_rc" != "0" ] && grep -q 'checksum mismatch' "$SANDBOX/mismatch.out"; then
	pass "mismatched engine digest aborts with a non-zero status"
else
	fail "mismatched engine digest aborts with a non-zero status (rc=${_mismatch_rc})"
fi
if [ ! -e "$MISMATCH_MARKER" ]; then
	pass "mismatched engine is never executed (execution marker absent)"
else
	fail "mismatched engine is never executed (execution marker absent)"
fi
if [ -f "$MISMATCH_LOG" ] && [ "$(wc -l < "$MISMATCH_LOG" | tr -d '[:space:]')" = "1" ]; then
	pass "mismatch case downloads exactly once"
else
	fail "mismatch case downloads exactly once"
fi
_dl_args=$(cat "$MISMATCH_LOG" 2> /dev/null || true)
case "$_dl_args" in
	*"--proto =https"*"--proto-redir =https"*) pass "stub observed the HTTPS-only curl policy" ;;
	*) fail "stub observed the HTTPS-only curl policy (got [${_dl_args}])" ;;
esac
case "$_dl_args" in
	*"$CANONICAL_ENGINE_URL"*) pass "download requests the canonical engine URL" ;;
	*) fail "download requests the canonical engine URL (got [${_dl_args}])" ;;
esac
case "$_dl_args" in
	*-o\ /tmp/beszel-installer.*) pass "engine is downloaded into private staging" ;;
	*) fail "engine is downloaded into private staging (got [${_dl_args}])" ;;
esac

# Case 2: pin and delivered engine agree -> download, verify, then execute
# the local manager.sh with the caller's arguments intact.
MATCH_MARKER="$SANDBOX/marker-match"
MATCH_LOG="$SANDBOX/curl-match.log"
export BESZEL_BOOTSTRAP_MARKER="$MATCH_MARKER"
export BESZEL_BOOTSTRAP_CURL_LOG="$MATCH_LOG"
fixture_bootstrap "$(digest_of "$FIXTURE_ENGINE")" "$SANDBOX/bootstrap-match.sh"
: > "$MATCH_LOG"
rm -f "$MATCH_MARKER"
if sh "$SANDBOX/bootstrap-match.sh" alpha "beta gamma" > "$SANDBOX/match.out" 2>&1; then
	_match_rc=0
else
	_match_rc=$?
fi
assert_eq "matching engine digest executes successfully" "0" "$_match_rc"
if [ -f "$MATCH_MARKER" ] && grep -q 'ENGINE_EXECUTED' "$MATCH_MARKER"; then
	pass "verified engine is executed"
else
	fail "verified engine is executed"
fi
if [ -f "$MATCH_MARKER" ] && grep -Eq '^zero=/tmp/beszel-installer\.[^/]*/manager\.sh$' "$MATCH_MARKER"; then
	pass "engine executes from /tmp/beszel-installer.XXXXXX/manager.sh"
else
	fail "engine executes from /tmp/beszel-installer.XXXXXX/manager.sh (got [$(cat "$MATCH_MARKER" 2> /dev/null || true)])"
fi
if [ -f "$MATCH_MARKER" ] && grep -q '^argc=2$' "$MATCH_MARKER" &&
	grep -q '^arg=alpha$' "$MATCH_MARKER" && grep -q '^arg=beta gamma$' "$MATCH_MARKER"; then
	pass "bootstrap forwards every argument, including spaces, unchanged"
else
	fail "bootstrap forwards every argument, including spaces, unchanged"
fi

# Case 3: no SHA-256 tool available -> abort before download and before
# execution. Verification is never skipped.
NOHASH_MARKER="$SANDBOX/marker-nohash"
NOHASH_LOG="$SANDBOX/curl-nohash.log"
export BESZEL_BOOTSTRAP_MARKER="$NOHASH_MARKER"
export BESZEL_BOOTSTRAP_CURL_LOG="$NOHASH_LOG"
fixture_bootstrap "$_engine_sha" "$SANDBOX/bootstrap-nohash.sh" "$SHIM"
: > "$NOHASH_LOG"
rm -f "$NOHASH_MARKER"
if sh "$SANDBOX/bootstrap-nohash.sh" > "$SANDBOX/nohash.out" 2>&1; then
	_nohash_rc=0
else
	_nohash_rc=$?
fi
if [ "$_nohash_rc" != "0" ] && grep -q 'SHA-256' "$SANDBOX/nohash.out"; then
	pass "missing SHA-256 tool aborts with a clear error"
else
	fail "missing SHA-256 tool aborts with a clear error (rc=${_nohash_rc})"
fi
if [ ! -e "$NOHASH_MARKER" ] && [ ! -s "$NOHASH_LOG" ]; then
	pass "missing SHA-256 tool aborts before download and before execution"
else
	fail "missing SHA-256 tool aborts before download and before execution"
fi

printf '== bootstrap staging cleanup ==\n'
_tmp_leftovers=$(ls -d /tmp/beszel-installer.* 2> /dev/null || true)
if [ "$_tmp_leftovers" = "$_tmp_before" ]; then
	pass "fixture runs leave no /tmp/beszel-installer.* staging directories behind"
else
	fail "fixture runs leave no /tmp/beszel-installer.* staging directories behind (before [${_tmp_before}], after [${_tmp_leftovers}])"
fi

printf '\n%d passed, %d failed\n' "$_pass" "$_fail"
[ "$_fail" = "0" ]
