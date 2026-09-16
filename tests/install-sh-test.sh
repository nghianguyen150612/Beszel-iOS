#!/bin/sh
# Lightweight tests for ../install.sh.
#
# Runs on a normal dev machine (Linux/macOS) as a non-root user. Nothing here
# touches /usr/local/bin, /var/lib or /Library/LaunchDaemons: path-dependent
# functions honour the BESZEL_* overrides, and only pure logic plus
# sandbox-redirected plist generation is exercised.
#
# Usage:  sh tests/install-sh-test.sh

set -eu

TEST_DIR=$(dirname "$0")
SCRIPT="${TEST_DIR}/../install.sh"

# Source install.sh for its functions without running the installer.
export BESZEL_INSTALL_LIB_ONLY=1
# shellcheck disable=SC1090,SC1091
. "$SCRIPT"

_pass=0
_fail=0

pass() { printf 'PASS: %s\n' "$1"; _pass=$((_pass + 1)); }
fail() { printf 'FAIL: %s\n' "$1"; _fail=$((_fail + 1)); }

# expect_ok <desc> <command...> — passes when the command succeeds.
expect_ok() {
	_desc="$1"
	shift
	if "$@" > /dev/null 2>&1; then
		pass "$_desc"
	else
		fail "$_desc"
	fi
}

# expect_fail <desc> <command...> — passes when the command fails.
expect_fail() {
	_desc="$1"
	shift
	if "$@" > /dev/null 2>&1; then
		fail "$_desc"
	else
		pass "$_desc"
	fi
}

assert_eq() {
	if [ "$2" = "$3" ]; then
		pass "$1"
	else
		fail "$1 (expected [$2], got [$3])"
	fi
}

SANDBOX=$(mktemp -d "${TMPDIR:-/tmp}/beszel-ios-test.XXXXXX")
trap 'rm -rf "$SANDBOX"' EXIT INT TERM HUP

printf '== port validation ==\n'
expect_ok "port 1 accepted" valid_port 1
expect_ok "port 80 accepted" valid_port 80
expect_ok "port 45876 accepted" valid_port 45876
expect_ok "port 8090 accepted" valid_port 8090
expect_ok "port 65535 accepted" valid_port 65535
expect_fail "empty port rejected" valid_port ""
expect_fail "port 0 rejected" valid_port 0
expect_fail "port 65536 rejected" valid_port 65536
expect_fail "port 99999 rejected" valid_port 99999
expect_fail "non-numeric port rejected" valid_port abc
expect_fail "partially numeric port rejected" valid_port 80a
expect_fail "negative port rejected" valid_port -1
expect_fail "port with space rejected" valid_port "8 0"
expect_fail "decimal port rejected" valid_port 1.5

printf '== ssh key validation ==\n'
expect_ok "ed25519 key with comment accepted" valid_ssh_key "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqM66/yBCvP5nLv8mQuczlB9lXh9B7 user@host"
expect_ok "rsa key accepted" valid_ssh_key "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7 comment"
expect_ok "ecdsa key accepted" valid_ssh_key "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBN comment"
expect_ok "key with xml-special comment accepted" valid_ssh_key "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqM66/yBCvP5nLv8mQuczlB9lXh9B7 comment & <tag> \"quoted\""
expect_fail "empty key rejected" valid_ssh_key ""
expect_fail "single field rejected" valid_ssh_key "ssh-ed25519"
expect_fail "unknown key type rejected" valid_ssh_key "ssh-foo AAAAC3NzaC1lZDI1NTE5AAAAIOMqqM66 comment"
expect_fail "bad base64 rejected" valid_ssh_key "ssh-ed25519 AAAAC3NzaC!!! comment"

printf '== xml escaping ==\n'
assert_eq "xml_escape basic" "a&amp;b&lt;c&gt;d&quot;e&apos;f" "$(xml_escape "a&b<c>d\"e'f")"
assert_eq "xml_escape key comment" "x &amp; &lt;y&gt;" "$(xml_escape "x & <y>")"
assert_eq "xml_escape plain unchanged" "abc123" "$(xml_escape "abc123")"

printf '== checksum parsing ==\n'
printf 'agent-payload' > "${SANDBOX}/beszel-agent-ios-arm64"
printf 'hub-payload-longer' > "${SANDBOX}/beszel-hub-ios-arm64"
_agent_hash=$(sha256sum "${SANDBOX}/beszel-agent-ios-arm64" | cut -d' ' -f1)
_hub_hash=$(sha256sum "${SANDBOX}/beszel-hub-ios-arm64" | cut -d' ' -f1)
printf '%s  %s\n%s  %s\n' "$_agent_hash" "beszel-agent-ios-arm64" "$_hub_hash" "beszel-hub-ios-arm64" > "${SANDBOX}/SHA256SUMS"
assert_eq "sums_hash_for agent" "$_agent_hash" "$(sums_hash_for "${SANDBOX}/SHA256SUMS" "beszel-agent-ios-arm64")"
assert_eq "sums_hash_for hub" "$_hub_hash" "$(sums_hash_for "${SANDBOX}/SHA256SUMS" "beszel-hub-ios-arm64")"
expect_ok "validate_sums_file accepts good file" validate_sums_file "${SANDBOX}/SHA256SUMS"
printf '%s  %s\n' "$_agent_hash" "beszel-agent-ios-arm64" > "${SANDBOX}/missing.SHA256SUMS"
expect_fail "missing hub entry rejected" validate_sums_file "${SANDBOX}/missing.SHA256SUMS"
printf '%s  %s\n%s  %s\n%s  %s\n' "$_agent_hash" "beszel-agent-ios-arm64" "$_agent_hash" "beszel-agent-ios-arm64" "$_hub_hash" "beszel-hub-ios-arm64" > "${SANDBOX}/dup.SHA256SUMS"
expect_fail "duplicate entry rejected" validate_sums_file "${SANDBOX}/dup.SHA256SUMS"
printf 'notahash  beszel-agent-ios-arm64\n%s  beszel-hub-ios-arm64\n' "$_hub_hash" > "${SANDBOX}/bad.SHA256SUMS"
expect_fail "malformed hash rejected" validate_sums_file "${SANDBOX}/bad.SHA256SUMS"
printf '%s  %s\n%s  %s\n%s  %s\n' "$_agent_hash" "beszel-agent-ios-arm64" "$_hub_hash" "beszel-hub-ios-arm64" "$_hub_hash" "extra-file" > "${SANDBOX}/extra.SHA256SUMS"
expect_fail "unexpected third entry rejected" validate_sums_file "${SANDBOX}/extra.SHA256SUMS"

printf '== hash computation and file verification ==\n'
_oracle=$(python3 -c 'import hashlib; print(hashlib.sha256(open("'"${SANDBOX}"'/beszel-agent-ios-arm64","rb").read()).hexdigest())')
assert_eq "compute_sha256 matches hashlib oracle" "$_oracle" "$(compute_sha256 "${SANDBOX}/beszel-agent-ios-arm64")"
expect_ok "verify_file accepts matching digest" verify_file "${SANDBOX}/beszel-agent-ios-arm64" "$_agent_hash"
expect_fail "verify_file rejects wrong digest" verify_file "${SANDBOX}/beszel-agent-ios-arm64" "$_hub_hash"
printf 'tampered' >> "${SANDBOX}/beszel-agent-ios-arm64"
expect_fail "verify_file rejects tampered file" verify_file "${SANDBOX}/beszel-agent-ios-arm64" "$_agent_hash"
: > "${SANDBOX}/empty.bin"
expect_fail "verify_file rejects empty file" verify_file "${SANDBOX}/empty.bin" "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

printf '== existing-install detection (sandboxed paths) ==\n'
BIN_DIR="${SANDBOX}/usr-local-bin"
LAUNCHD_DIR="${SANDBOX}/launchd"
mkdir -p "$BIN_DIR" "$LAUNCHD_DIR"
expect_fail "agent not detected when absent" agent_installed
expect_fail "hub not detected when absent" hub_installed
touch "${BIN_DIR}/beszel-agent"
expect_ok "agent detected via binary" agent_installed
rm "${BIN_DIR}/beszel-agent"
touch "${LAUNCHD_DIR}/dev.beszel.agent.plist"
expect_ok "agent detected via plist" agent_installed
touch "${BIN_DIR}/beszel-hub"
expect_ok "hub detected via binary" hub_installed
rm "${BIN_DIR}/beszel-hub"
touch "${LAUNCHD_DIR}/dev.beszel.hub.plist"
expect_ok "hub detected via plist" hub_installed

printf '== plist generation and xml validity ==\n'
NASTY_KEY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqM66/yBCvP5nLv8mQuczlB9lXh9B7 ipad & <lab> \"test\""
write_agent_plist "$NASTY_KEY" "45876" "${SANDBOX}/dev.beszel.agent.plist"
write_hub_plist "8090" "${SANDBOX}/dev.beszel.hub.plist"
export SANDBOX
python3 - << 'PYEOF'
import os, xml.dom.minidom
sandbox = os.environ["SANDBOX"]
for name in ("dev.beszel.agent.plist", "dev.beszel.hub.plist"):
    xml.dom.minidom.parse(os.path.join(sandbox, name))
    print("xml parses: " + name)
PYEOF
if grep -q "ipad &amp; &lt;lab&gt; &quot;test&quot;" "${SANDBOX}/dev.beszel.agent.plist"; then
	pass "agent key xml-escaped in plist"
else
	fail "agent key xml-escaped in plist"
fi
if grep -q "ipad & <lab>" "${SANDBOX}/dev.beszel.agent.plist"; then
	fail "no raw xml-special key text in plist"
else
	pass "no raw xml-special key text in plist"
fi
_roundtrip=$(python3 -c 'import os,xml.dom.minidom; d=xml.dom.minidom.parse(os.path.join(os.environ["SANDBOX"],"dev.beszel.agent.plist")); print([n.firstChild.data for n in d.getElementsByTagName("string") if n.firstChild and n.firstChild.data.startswith("ssh-ed25519")][0])')
assert_eq "agent key round-trips through xml" "$NASTY_KEY" "$_roundtrip"
if grep -q "<string>serve</string>" "${SANDBOX}/dev.beszel.hub.plist" && grep -q "0.0.0.0:8090" "${SANDBOX}/dev.beszel.hub.plist" && grep -q "ThrottleInterval" "${SANDBOX}/dev.beszel.hub.plist"; then
	pass "hub plist contains serve args and throttle"
else
	fail "hub plist contains serve args and throttle"
fi
if grep -q "dev.beszel.agent" "${SANDBOX}/dev.beszel.agent.plist" && grep -q "dev.beszel.hub" "${SANDBOX}/dev.beszel.hub.plist"; then
	pass "plist labels correct"
else
	fail "plist labels correct"
fi

printf '== fetch and verify over file:// (stdout must carry only the path) ==\n'
mkdir -p "${SANDBOX}/release" "${SANDBOX}/work"
cp "${SANDBOX}/SHA256SUMS" "${SANDBOX}/release/SHA256SUMS"
cp "${SANDBOX}/SHA256SUMS" "${SANDBOX}/work/SHA256SUMS"
cp "${SANDBOX}/beszel-hub-ios-arm64" "${SANDBOX}/release/beszel-hub-ios-arm64"
# Redirect downloads to a local file:// tree (consumed by install.sh functions).
# shellcheck disable=SC2034
RELEASE_BASE="file://${SANDBOX}/release"
WORK_DIR="${SANDBOX}/work"
_got=$(fetch_and_verify_binary "beszel-hub-ios-arm64" 2> /dev/null)
assert_eq "fetch returns clean verified path" "${WORK_DIR}/beszel-hub-ios-arm64" "$_got"
if ( fetch_and_verify_binary "beszel-agent-ios-arm64" ) > /dev/null 2>&1; then
	fail "fetch of asset missing from sums is refused"
else
	pass "fetch of asset missing from sums is refused"
fi

printf '== environment gates (must refuse this non-iOS host) ==\n'
# Gates use die (exit), so they must run in a subshell to stay testable.
if ( check_device ) > /dev/null 2>&1; then
	fail "check_device refuses non-iOS host"
else
	pass "check_device refuses non-iOS host"
fi
if [ "$(id -u)" != "0" ]; then
	if ( check_root ) > /dev/null 2>&1; then
		fail "check_root refuses non-root caller"
	else
		pass "check_root refuses non-root caller"
	fi
else
	pass "check_root skipped (tests running as root)"
fi

printf '== prompt5: release tag helpers ==\n'
assert_eq "tag from standard latest URL" "v0.19.0-ios.1" "$(tag_from_latest_url "https://github.com/nghianguyen150612/beszel-ios/releases/tag/v0.19.0-ios.1")"
assert_eq "tag tolerates trailing slash" "v0.19.0-ios.1" "$(tag_from_latest_url "https://github.com/nghianguyen150612/beszel-ios/releases/tag/v0.19.0-ios.1/")"
expect_fail "empty url rejected" tag_from_latest_url ""
expect_fail "metachar url rejected" tag_from_latest_url "https://example.com/releases/tag/v1;touch"
# shellcheck disable=SC2016
expect_fail "dollar url rejected" tag_from_latest_url 'https://example.com/tag/$(x)'
expect_fail "space url rejected" tag_from_latest_url "https://example.com/v1 evil"
expect_ok "release tag accepted" valid_release_tag "v0.19.0-ios.1"
expect_ok "multi-digit ios rev accepted" valid_release_tag "v1.2.3-ios.10"
expect_fail "bare semver rejected" valid_release_tag "v0.19.0"
expect_fail "ios rev zero rejected" valid_release_tag "v0.19.0-ios.0"
expect_fail "non-numeric rev rejected" valid_release_tag "v0.19.0-ios.x"
expect_fail "latest pseudo-tag rejected" valid_release_tag "latest"
expect_fail "empty tag rejected" valid_release_tag ""
expect_fail "traversal tag rejected" valid_release_tag "v0.19.0-ios.1/../../x"
expect_fail "spaced tag rejected" valid_release_tag "v0.19.0-ios.1 evil"
expect_fail "missing v prefix rejected" valid_release_tag "0.19.0-ios.1"
assert_eq "pinned base construction" "https://github.com/nghianguyen150612/beszel-ios/releases/download/v0.19.0-ios.1" "$(pinned_base_for_tag "v0.19.0-ios.1")"
expect_fail "pinned base rejects bad tag" pinned_base_for_tag "latest"
expect_fail "pinned base rejects empty tag" pinned_base_for_tag ""

printf '== prompt5: mock commands and sandbox ==\n'
export BESZEL_UPDATE_SETTLE_SECS=0
REAL_CURL_BIN=$(command -v curl)
MOCKBIN="${SANDBOX}/mockbin"
MOCKCTL="${SANDBOX}/mockctl"
mkdir -p "$MOCKBIN" "$MOCKCTL"
export BESZEL_TEST_REAL_CURL="$REAL_CURL_BIN"
export BESZEL_TEST_MOCKCTL="$MOCKCTL"
export BESZEL_TEST_HEALTH=ok
export BESZEL_TEST_LATEST_URL="https://github.com/nghianguyen150612/beszel-ios/releases/tag/v0.19.9-ios.7"
export BESZEL_TEST_RESOLVE_FAIL=0
export BESZEL_TEST_DOWNLOAD_FAIL=0
export BESZEL_TEST_LDID_FAIL=0
export BESZEL_TEST_UNLOAD_FAIL=0
export BESZEL_TEST_LOAD_FAIL_LABEL=""
export BESZEL_TEST_NO_PID=0
export BESZEL_TEST_LIST_FAIL=0
export BESZEL_TEST_PLIST_VALID=ok

cat > "${MOCKBIN}/curl" << 'MOCKEOF'
#!/bin/sh
_last=""
_health=0
_latest=0
_want_eff=0
for _a in "$@"; do
	case "$_a" in
		*api/health*) _health=1 ;;
		*releases/latest*) _latest=1 ;;
		*url_effective*) _want_eff=1 ;;
	esac
	_last="$_a"
done
cd "${BESZEL_TEST_MOCKCTL}" || exit 1
if [ "$_health" = "1" ]; then
	printf '%s\n' "$_last" >> health.log
	if [ -f health_fail_remaining ]; then
		_n=$(cat health_fail_remaining)
		case "$_n" in '' | *[!0-9]*) _n=0 ;; esac
		if [ "$_n" -gt 0 ]; then
			_n=$((_n - 1))
			printf '%s' "$_n" > health_fail_remaining
			exit 7
		fi
	fi
	if [ "${BESZEL_TEST_HEALTH:-ok}" = "ok" ]; then
		printf '{}'
		exit 0
	fi
	exit 7
fi
if [ "$_latest" = "1" ] && [ "$_want_eff" = "1" ]; then
	if [ "${BESZEL_TEST_RESOLVE_FAIL:-0}" = "1" ]; then exit 6; fi
	printf '%s' "${BESZEL_TEST_LATEST_URL:-https://github.com/nghianguyen150612/beszel-ios/releases/tag/v0.19.0-ios.1}"
	exit 0
fi
if [ "${BESZEL_TEST_DOWNLOAD_FAIL:-0}" = "1" ]; then
	case "$_last" in
		*beszel-agent-ios-arm64 | *beszel-hub-ios-arm64) exit 1 ;;
	esac
fi
exec "${BESZEL_TEST_REAL_CURL}" "$@"
MOCKEOF
cat > "${MOCKBIN}/launchctl" << 'MOCKEOF'
#!/bin/sh
cd "${BESZEL_TEST_MOCKCTL}" || exit 1
_mcmd="${1:-}"
_plist=""
for _a in "$@"; do _plist="$_a"; done
_mlabel=""
case "$_plist" in
	*dev.beszel.agent*) _mlabel="agent" ;;
	*dev.beszel.hub*) _mlabel="hub" ;;
esac
case "$_mcmd" in
	unload)
		printf 'unload %s\n' "$_plist" >> mock.log
		if [ "${BESZEL_TEST_UNLOAD_FAIL:-0}" = "1" ]; then exit 1; fi
		if [ "$_mlabel" = "agent" ]; then printf '0' > loaded_agent; printf '-' > pid_agent; fi
		if [ "$_mlabel" = "hub" ]; then printf '0' > loaded_hub; printf '-' > pid_hub; fi
		exit 0
		;;
	load)
		printf 'load %s\n' "$_plist" >> mock.log
		if [ -n "$_mlabel" ] && [ -f "load_fail_remaining_${_mlabel}" ]; then
			_n=$(cat "load_fail_remaining_${_mlabel}")
			case "$_n" in '' | *[!0-9]*) _n=0 ;; esac
			if [ "$_n" -gt 0 ]; then
				_n=$((_n - 1))
				printf '%s' "$_n" > "load_fail_remaining_${_mlabel}"
				exit 1
			fi
		fi
		if [ -f load_fail_remaining ]; then
			_n=$(cat load_fail_remaining)
			case "$_n" in '' | *[!0-9]*) _n=0 ;; esac
			if [ "$_n" -gt 0 ]; then
				_n=$((_n - 1))
				printf '%s' "$_n" > load_fail_remaining
				exit 1
			fi
		fi
		_fl="${BESZEL_TEST_LOAD_FAIL_LABEL:-}"
		if [ -n "$_mlabel" ] && { [ "$_fl" = "$_mlabel" ] || [ "$_fl" = "both" ]; }; then exit 1; fi
		if [ "$_mlabel" = "agent" ]; then
			printf '1' > loaded_agent
			if [ "${BESZEL_TEST_NO_PID:-0}" = "1" ]; then printf '-' > pid_agent; else printf '4321' > pid_agent; fi
		fi
		if [ "$_mlabel" = "hub" ]; then
			printf '1' > loaded_hub
			if [ "${BESZEL_TEST_NO_PID:-0}" = "1" ]; then printf '-' > pid_hub; else printf '4322' > pid_hub; fi
		fi
		exit 0
		;;
	list)
		if [ "${BESZEL_TEST_LIST_FAIL:-0}" = "1" ]; then exit 1; fi
		printf 'PID Status Label\n'
		if [ "$(cat loaded_agent 2> /dev/null)" = "1" ]; then printf '%s 0 dev.beszel.agent\n' "$(cat pid_agent 2> /dev/null)"; fi
		if [ "$(cat loaded_hub 2> /dev/null)" = "1" ]; then printf '%s 0 dev.beszel.hub\n' "$(cat pid_hub 2> /dev/null)"; fi
		exit 0
		;;
	*)
		exit 0
		;;
esac
MOCKEOF
cat > "${MOCKBIN}/ldid" << 'MOCKEOF'
#!/bin/sh
if [ "${BESZEL_TEST_LDID_FAIL:-0}" = "1" ]; then exit 1; fi
_bin=""
for _a in "$@"; do _bin="$_a"; done
if [ -n "$_bin" ] && [ -f "$_bin" ]; then
	printf '\n# mock-ldid-signature\n' >> "$_bin"
fi
exit 0
MOCKEOF
cat > "${MOCKBIN}/sleep" << 'MOCKEOF'
#!/bin/sh
exit 0
MOCKEOF
cat > "${MOCKBIN}/plutil" << 'MOCKEOF'
#!/bin/sh
if [ "${BESZEL_TEST_PLIST_VALID:-ok}" = "ok" ]; then exit 0; fi
exit 1
MOCKEOF
chmod +x "${MOCKBIN}/curl" "${MOCKBIN}/launchctl" "${MOCKBIN}/ldid" "${MOCKBIN}/sleep" "${MOCKBIN}/plutil"
PATH="${MOCKBIN}:$PATH"
export PATH

SB_BIN="${SANDBOX}/bin"
SB_LIB="${SANDBOX}/lib"
SB_LAUNCHD="${SANDBOX}/launchd2"
SB_LOG="${SANDBOX}/log"
SB_REL="${SANDBOX}/rel"
SB_WORK="${TMPDIR:-/tmp}/beszel-txn-work-$$"
trap 'rm -rf "$SANDBOX" "${SB_WORK:-}"' EXIT
trap 'rm -rf "$SANDBOX" "${SB_WORK:-}"; exit 130' INT TERM HUP

t_reset_paths() {
	rm -rf "$SB_BIN" "$SB_LIB" "$SB_LAUNCHD" "$SB_LOG" "$SB_WORK"
	mkdir -p "$SB_BIN" "$SB_LIB" "$SB_LAUNCHD" "$SB_LOG" "$SB_WORK"
	BIN_DIR="$SB_BIN"
	LIB_DIR="$SB_LIB"
	LAUNCHD_DIR="$SB_LAUNCHD"
	# shellcheck disable=SC2034
	LOG_DIR="$SB_LOG"
	WORK_DIR="$SB_WORK"
	PINNED_RELEASE_BASE=""
	LATEST_TAG=""
	# shellcheck disable=SC2034
	SUMS_FETCHED=0
}

mock_reset() {
	printf '0' > "${MOCKCTL}/loaded_agent"
	printf '0' > "${MOCKCTL}/loaded_hub"
	printf '-' > "${MOCKCTL}/pid_agent"
	printf '-' > "${MOCKCTL}/pid_hub"
	: > "${MOCKCTL}/mock.log"
	: > "${MOCKCTL}/health.log"
	rm -f "${MOCKCTL}/health_fail_remaining" "${MOCKCTL}/load_fail_remaining"
	rm -f "${MOCKCTL}"/load_fail_remaining_*
	BESZEL_TEST_HEALTH=ok
	BESZEL_TEST_RESOLVE_FAIL=0
	BESZEL_TEST_DOWNLOAD_FAIL=0
	BESZEL_TEST_LDID_FAIL=0
	BESZEL_TEST_UNLOAD_FAIL=0
	BESZEL_TEST_LOAD_FAIL_LABEL=""
	BESZEL_TEST_NO_PID=0
	BESZEL_TEST_LIST_FAIL=0
	BESZEL_TEST_PLIST_VALID=ok
}

t_make_release() {
	rm -rf "$SB_REL"
	mkdir -p "$SB_REL"
	printf '%s' "$2" > "${SB_REL}/beszel-agent-ios-arm64"
	printf '%s' "$3" > "${SB_REL}/beszel-hub-ios-arm64"
	( cd "${SB_REL}" && sha256sum beszel-agent-ios-arm64 beszel-hub-ios-arm64 > SHA256SUMS )
	PINNED_RELEASE_BASE="file://${SB_REL}"
	LATEST_TAG="$1"
	# shellcheck disable=SC2034
	SUMS_FETCHED=0
	rm -f "${SB_WORK}/SHA256SUMS" "${SB_WORK}/beszel-agent-ios-arm64" "${SB_WORK}/beszel-hub-ios-arm64"
}

t_prep_agent() {
	printf '%s' "$1" > "${BIN_DIR}/${AGENT_BIN}"
	chmod 755 "${BIN_DIR}/${AGENT_BIN}"
	write_agent_plist "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqM66/yBCvP5nLv8mQuczlB9lXh9B7 test@host" "${2:-45876}" "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
}

t_prep_hub() {
	printf '%s' "$1" > "${BIN_DIR}/${HUB_BIN}"
	chmod 755 "${BIN_DIR}/${HUB_BIN}"
	write_hub_plist "${2:-8090}" "${LAUNCHD_DIR}/${HUB_LABEL}.plist"
}

t_mode() {
	if stat -c %a "$1" > /dev/null 2>&1; then
		stat -c %a "$1"
	else
		stat -f %Lp "$1"
	fi
}

printf '== prompt5: latest resolution and pinned base ==\n'
t_reset_paths
mock_reset
assert_eq "resolve_latest_tag via redirect" "v0.19.9-ios.7" "$(resolve_latest_tag)"
BESZEL_TEST_RESOLVE_FAIL=1
expect_fail "resolve fails when redirect fails" resolve_latest_tag
BESZEL_TEST_RESOLVE_FAIL=0
LATEST_TAG=""
PINNED_RELEASE_BASE=""
ensure_pinned_release > /dev/null 2>&1
assert_eq "ensure pins resolved tag" "v0.19.9-ios.7" "$LATEST_TAG"
assert_eq "ensure pins immutable base" "https://github.com/nghianguyen150612/beszel-ios/releases/download/v0.19.9-ios.7" "$PINNED_RELEASE_BASE"
assert_eq "current base prefers pin" "$PINNED_RELEASE_BASE" "$(current_release_base)"
BESZEL_TEST_RESOLVE_FAIL=1
expect_ok "ensure keeps pin without re-resolving" ensure_pinned_release
BESZEL_TEST_RESOLVE_FAIL=0
PINNED_RELEASE_BASE=""
assert_eq "current base falls back to latest" "$RELEASE_BASE" "$(current_release_base)"

printf '== prompt5: install state ==\n'
t_reset_paths
mock_reset
expect_fail "agent release missing without state" state_agent_release
expect_fail "hub release missing without state" state_hub_release
assert_eq "legacy label when state missing" "unknown (installed before release tracking)" "$(current_release_label agent)"
t_make_release "v0.19.0-ios.1" "agent-bytes-1" "hub-bytes-1"
fetch_sums > /dev/null 2>&1
_T_SHA_A=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$AGENT_ASSET")
_T_SHA_H=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$HUB_ASSET")
expect_ok "state write agent" state_write_component "agent" "v0.19.0-ios.1" "$_T_SHA_A"
assert_eq "state reads agent release" "v0.19.0-ios.1" "$(state_agent_release)"
assert_eq "state reads agent sha" "$_T_SHA_A" "$(state_agent_sha)"
expect_fail "hub still missing after agent-only write" state_hub_release
expect_ok "state write hub preserves agent" state_write_component "hub" "v0.19.0-ios.1" "$_T_SHA_H"
assert_eq "agent preserved after hub write" "v0.19.0-ios.1" "$(state_agent_release)"
assert_eq "hub recorded" "v0.19.0-ios.1" "$(state_hub_release)"
assert_eq "state file mode is 644" "644" "$(t_mode "$(state_path)")"
assert_eq "state dir mode is 755" "755" "$(t_mode "${LIB_DIR}/${STATE_SUBDIR}")"
if grep -q "STATE_VERSION=1" "$(state_path)" && grep -q "AGENT_RELEASE=v0.19.0-ios.1" "$(state_path)" && grep -q "HUB_RELEASE=v0.19.0-ios.1" "$(state_path)"; then
	pass "state file has expected keys"
else
	fail "state file has expected keys"
fi
_T_STATE_TMP_LEFT=0
for _t_f in "${LIB_DIR}/${STATE_SUBDIR}/${STATE_FILE_NAME}".new.*; do
	if [ -e "$_t_f" ]; then
		_T_STATE_TMP_LEFT=1
	fi
done
if [ "$_T_STATE_TMP_LEFT" = "1" ]; then
	fail "no state temp leftovers"
else
	pass "no state temp leftovers"
fi
printf 'BOGUS_KEY=yes\n' >> "$(state_path)"
if grep -q "BOGUS_KEY" "$(state_path)"; then
	_T_AFTER_JUNK=$(state_hub_release 2> /dev/null || true)
	assert_eq "unknown keys ignored, hub still readable" "v0.19.0-ios.1" "$_T_AFTER_JUNK"
else
	fail "junk key present for ignore test"
fi
# shellcheck disable=SC2016
printf 'STATE_VERSION=1\nAGENT_RELEASE=$(touch %s/pwned-by-state)\nAGENT_ASSET_SHA256=%s\nHUB_RELEASE=\nHUB_ASSET_SHA256=\nBOGUS2=$(touch %s/pwned2-by-state)\n' "$SANDBOX" "$_T_SHA_A" "$SANDBOX" > "$(state_path)"
expect_fail "malicious release value rejected" state_agent_release
expect_fail "nothing readable as current from evil state" release_is_current "agent" "v0.19.0-ios.1"
if [ -e "${SANDBOX}/pwned-by-state" ] || [ -e "${SANDBOX}/pwned2-by-state" ]; then
	fail "malicious state line never executes"
else
	pass "malicious state line never executes"
fi
rm -f "$(state_path)"
expect_ok "state rewrite works after evil file" state_write_component "agent" "v0.19.0-ios.1" "$_T_SHA_A"
assert_eq "state readable after rewrite" "v0.19.0-ios.1" "$(state_agent_release)"
expect_fail "state write rejects bad tag" state_write_component "agent" "latest" "$_T_SHA_A"
expect_fail "state write rejects bad sha" state_write_component "agent" "v0.19.0-ios.1" "deadbeef"
expect_fail "state write rejects bad component" state_write_component "hubbub" "v0.19.0-ios.1" "$_T_SHA_A"
if grep -q "ssh-ed25519\|AAAAC3" "$(state_path)"; then
	fail "no key material in state file"
else
	pass "no key material in state file"
fi
expect_ok "current release detected" release_is_current "agent" "v0.19.0-ios.1"
expect_fail "other release not current" release_is_current "agent" "v0.19.0-ios.2"
expect_fail "empty tag never current" release_is_current "agent" ""
expect_fail "missing component never current" release_is_current "hubbub" "v0.19.0-ios.1"

printf '== prompt5: component status ==\n'
t_reset_paths
mock_reset
assert_eq "status missing when absent" "missing" "$(component_status agent)"
assert_eq "hub status missing when absent" "missing" "$(component_status hub)"
printf 'x' > "${BIN_DIR}/${AGENT_BIN}"
assert_eq "binary without plist is incomplete" "incomplete" "$(component_status agent)"
rm "${BIN_DIR}/${AGENT_BIN}"
touch "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
assert_eq "plist without binary is incomplete" "incomplete" "$(component_status agent)"
printf 'x' > "${BIN_DIR}/${AGENT_BIN}"
assert_eq "binary plus plist is complete" "complete" "$(component_status agent)"

printf '== prompt5: hub port parsing ==\n'
t_reset_paths
mock_reset
t_prep_hub "hubbytes" "8090"
assert_eq "default hub port parsed" "8090" "$(hub_port_from_plist "${LAUNCHD_DIR}/${HUB_LABEL}.plist")"
t_prep_hub "hubbytes" "1234"
assert_eq "custom hub port parsed" "1234" "$(hub_port_from_plist "${LAUNCHD_DIR}/${HUB_LABEL}.plist")"
expect_fail "missing plist has no port" hub_port_from_plist "${LAUNCHD_DIR}/nope.plist"
printf '<plist><string>no bind here</string></plist>\n' > "${LAUNCHD_DIR}/${HUB_LABEL}.plist"
expect_fail "plist without bind is refused" hub_port_from_plist "${LAUNCHD_DIR}/${HUB_LABEL}.plist"
write_hub_plist "99999" "${LAUNCHD_DIR}/${HUB_LABEL}.plist"
expect_fail "out-of-range port refused" hub_port_from_plist "${LAUNCHD_DIR}/${HUB_LABEL}.plist"

printf '== prompt5: staging and backup ==\n'
t_reset_paths
mock_reset
t_make_release "v0.19.0-ios.2" "agent-new-bytes" "hub-new-bytes"
fetch_sums > /dev/null 2>&1
printf 'agent-old-bytes' > "${BIN_DIR}/${AGENT_BIN}"
_T_STAGED=$(stage_new_binary "$AGENT_ASSET" "$AGENT_BIN")
assert_eq "staged path is .new file" "${BIN_DIR}/${AGENT_BIN}.new" "$_T_STAGED"
if grep -q "mock-ldid-signature" "$_T_STAGED" && ! grep -q "mock-ldid-signature" "${BIN_DIR}/${AGENT_BIN}"; then
	pass "staged binary signed, installed untouched"
else
	fail "staged binary signed, installed untouched"
fi
expect_ok "backup created" backup_current_binary "$AGENT_BIN"
if cmp -s "${BIN_DIR}/${AGENT_BIN}" "${BIN_DIR}/${AGENT_BIN}.bak"; then
	pass "backup matches current binary bytes"
else
	fail "backup matches current binary bytes"
fi
if [ -e "${BIN_DIR}/${AGENT_BIN}.bak.new" ]; then
	fail "no backup temp leftovers"
else
	pass "no backup temp leftovers"
fi
printf 'agent-newer-bytes' > "${BIN_DIR}/${AGENT_BIN}"
expect_ok "backup replaced on next update" backup_current_binary "$AGENT_BIN"
if cmp -s "${BIN_DIR}/${AGENT_BIN}" "${BIN_DIR}/${AGENT_BIN}.bak"; then
	pass "replacement backup tracks current"
else
	fail "replacement backup tracks current"
fi
rm -f "${BIN_DIR}/${AGENT_BIN}" "${BIN_DIR}/${AGENT_BIN}.bak"
expect_fail "backup of missing binary fails" backup_current_binary "$AGENT_BIN"
printf 'x' > "${SANDBOX}/link-target"
ln -s "${SANDBOX}/link-target" "${BIN_DIR}/${AGENT_BIN}"
expect_fail "backup refuses symlinked binary" backup_current_binary "$AGENT_BIN"
rm -f "${BIN_DIR}/${AGENT_BIN}"
BESZEL_TEST_LDID_FAIL=1
expect_fail "ldid failure aborts staging" stage_new_binary "$AGENT_ASSET" "$AGENT_BIN"
BESZEL_TEST_LDID_FAIL=0
if [ -e "${BIN_DIR}/${AGENT_BIN}.new" ]; then
	fail "failed staging leaves no .new file"
else
	pass "failed staging leaves no .new file"
fi

printf '== prompt5: agent update transactions ==\n'
t_reset_paths
mock_reset
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
_T_SHA_NEW_A=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$AGENT_ASSET")
t_prep_agent "agent-v1-content"
cp "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/agent-plist-ref"
printf 'agent-v1-content' > "${SANDBOX}/agent-v1-ref"
printf '1' > "${MOCKCTL}/loaded_agent"
printf '4100' > "${MOCKCTL}/pid_agent"
if ( transact_agent_update "v0.19.0-ios.2" > "${SANDBOX}/txn.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "legacy agent update succeeds" "0" "$_rc"
if grep -q "agent-v2-content" "${BIN_DIR}/${AGENT_BIN}" && grep -q "mock-ldid-signature" "${BIN_DIR}/${AGENT_BIN}"; then
	pass "new agent binary installed and signed"
else
	fail "new agent binary installed and signed"
fi
if cmp -s "${BIN_DIR}/${AGENT_BIN}.bak" "${SANDBOX}/agent-v1-ref"; then
	pass "agent backup holds exact previous bytes"
else
	fail "agent backup holds exact previous bytes"
fi
if cmp -s "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/agent-plist-ref"; then
	pass "agent plist bytes preserved exactly"
else
	fail "agent plist bytes preserved exactly"
fi
assert_eq "agent state committed to new tag" "v0.19.0-ios.2" "$(state_agent_release)"
assert_eq "agent state sha matches release sums" "$_T_SHA_NEW_A" "$(state_agent_sha)"
assert_eq "agent service loaded after update" "1" "$(cat "${MOCKCTL}/loaded_agent")"
if [ -e "${BIN_DIR}/${AGENT_BIN}.new" ] || [ -e "${BIN_DIR}/${AGENT_BIN}.rollback" ]; then
	fail "no staging leftovers after success"
else
	pass "no staging leftovers after success"
fi

printf '== prompt5: update flows, confirmation, no-op ==\n'
# Invoked indirectly via the update_*_flow subshells below.
# shellcheck disable=SC2329
confirm_update() { return 0; }
t_reset_paths
mock_reset
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
_T_SHA_NEW_A2=$(sha256sum "${SB_REL}/beszel-agent-ios-arm64" | cut -d' ' -f1)
expect_ok "pre-write current state" state_write_component "agent" "v0.19.0-ios.2" "$_T_SHA_NEW_A2"
t_prep_agent "agent-v2-content"
if ( update_agent_flow "v0.19.0-ios.2" > "${SANDBOX}/noop.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "already-current agent is a no-op success" "0" "$_rc"
if grep -q "already on v0.19.0-ios.2" "${SANDBOX}/noop.log"; then
	pass "already-current message shown"
else
	fail "already-current message shown"
fi
if [ -e "${SB_WORK}/SHA256SUMS" ] || [ -e "${BIN_DIR}/${AGENT_BIN}.bak" ]; then
	fail "already-current downloads nothing and touches nothing"
else
	pass "already-current downloads nothing and touches nothing"
fi
rm -f "$(state_path)"
if ( update_agent_flow "v0.19.0-ios.2" > "${SANDBOX}/legacy.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "legacy unknown agent updates via flow" "0" "$_rc"
assert_eq "legacy flow commits state" "v0.19.0-ios.2" "$(state_agent_release)"
# shellcheck disable=SC2329
confirm_update() { return 1; }
t_reset_paths
mock_reset
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
t_prep_agent "agent-v1-content"
if ( update_agent_flow "v0.19.0-ios.2" > "${SANDBOX}/cancel.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "cancelled update exits zero" "0" "$_rc"
if [ -e "${SB_WORK}/SHA256SUMS" ] || [ -e "${BIN_DIR}/${AGENT_BIN}.bak" ]; then
	fail "cancelled update changes nothing"
else
	pass "cancelled update changes nothing"
fi
if cmp -s "${BIN_DIR}/${AGENT_BIN}" "${SANDBOX}/agent-v1-ref" 2> /dev/null || grep -q "agent-v1-content" "${BIN_DIR}/${AGENT_BIN}"; then
	pass "cancelled update keeps old binary"
else
	fail "cancelled update keeps old binary"
fi
# Invoked indirectly through the hub update flow exercised below.
# shellcheck disable=SC2329
confirm_update() { return 0; }

printf '== prompt5: hub update, data preservation ==\n'
t_reset_paths
mock_reset
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
_T_SHA_NEW_H=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$HUB_ASSET")
t_prep_hub "hub-v1-content" "8123"
mkdir -p "${LIB_DIR}/beszel-hub"
printf 'precious-hub-database-bytes' > "${LIB_DIR}/beszel-hub/db.sqlite"
cp "${LAUNCHD_DIR}/${HUB_LABEL}.plist" "${SANDBOX}/hub-plist-ref"
printf 'hub-v1-content' > "${SANDBOX}/hub-v1-ref"
printf '1' > "${MOCKCTL}/loaded_hub"
printf '4200' > "${MOCKCTL}/pid_hub"
if ( update_hub_flow "v0.19.0-ios.2" > "${SANDBOX}/hub.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "hub update succeeds" "0" "$_rc"
if grep -q ":8123/api/health" "${MOCKCTL}/health.log"; then
	pass "health checked on configured custom port"
else
	fail "health checked on configured custom port"
fi
if grep -q "hub-v2-content" "${BIN_DIR}/${HUB_BIN}"; then
	pass "new hub binary installed"
else
	fail "new hub binary installed"
fi
if cmp -s "${BIN_DIR}/${HUB_BIN}.bak" "${SANDBOX}/hub-v1-ref"; then
	pass "hub backup holds exact previous bytes"
else
	fail "hub backup holds exact previous bytes"
fi
if cmp -s "${LAUNCHD_DIR}/${HUB_LABEL}.plist" "${SANDBOX}/hub-plist-ref"; then
	pass "hub plist bytes preserved exactly"
else
	fail "hub plist bytes preserved exactly"
fi
if grep -q "precious-hub-database-bytes" "${LIB_DIR}/beszel-hub/db.sqlite"; then
	pass "hub database bytes untouched"
else
	fail "hub database bytes untouched"
fi
assert_eq "hub state committed" "v0.19.0-ios.2" "$(state_hub_release)"
assert_eq "hub state sha matches sums" "$_T_SHA_NEW_H" "$(state_hub_sha)"

printf '== prompt5: failure injection, pre-stop safety ==\n'
t_reset_paths
mock_reset
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
t_prep_agent "agent-v1-content"
cp "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/agent-plist-ref2"
BESZEL_TEST_DOWNLOAD_FAIL=1
if ( transact_agent_update "v0.19.0-ios.2" > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "download failure exits non-zero" "1" "$_rc"
BESZEL_TEST_DOWNLOAD_FAIL=0
if grep -q "agent-v1-content" "${BIN_DIR}/${AGENT_BIN}" && [ ! -e "${BIN_DIR}/${AGENT_BIN}.bak" ] && [ ! -e "${BIN_DIR}/${AGENT_BIN}.new" ]; then
	pass "download failure leaves install byte-for-byte unchanged"
else
	fail "download failure leaves install byte-for-byte unchanged"
fi
if cmp -s "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/agent-plist-ref2"; then
	pass "download failure preserves plist"
else
	fail "download failure preserves plist"
fi
printf 'hub-v1-content' >> "${SB_REL}/beszel-hub-ios-arm64"
t_prep_hub "hub-v1-content" "8090"
cp "${LAUNCHD_DIR}/${HUB_LABEL}.plist" "${SANDBOX}/hub-plist-ref2"
if ( transact_hub_update "v0.19.0-ios.2" > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "checksum mismatch exits non-zero" "1" "$_rc"
if grep -q "hub-v1-content" "${BIN_DIR}/${HUB_BIN}" && [ ! -e "${BIN_DIR}/${HUB_BIN}.bak" ] && [ ! -e "${BIN_DIR}/${HUB_BIN}.new" ]; then
	pass "checksum mismatch leaves install byte-for-byte unchanged"
else
	fail "checksum mismatch leaves install byte-for-byte unchanged"
fi
if [ "$(cat "${MOCKCTL}/loaded_hub")" = "0" ] && ! grep -q "^load " "${MOCKCTL}/mock.log"; then
	pass "checksum mismatch never touches the service"
else
	fail "checksum mismatch never touches the service"
fi
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
BESZEL_TEST_LDID_FAIL=1
if ( transact_agent_update "v0.19.0-ios.2" > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "ldid failure exits non-zero" "1" "$_rc"
BESZEL_TEST_LDID_FAIL=0
if grep -q "agent-v1-content" "${BIN_DIR}/${AGENT_BIN}" && [ ! -e "${BIN_DIR}/${AGENT_BIN}.bak" ]; then
	pass "ldid failure keeps old service untouched, no backup"
else
	fail "ldid failure keeps old service untouched, no backup"
fi
printf '<plist><string>no bind here</string></plist>\n' > "${LAUNCHD_DIR}/${HUB_LABEL}.plist"
if ( transact_hub_update "v0.19.0-ios.2" > "${SANDBOX}/portfail.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "unparseable hub port aborts" "1" "$_rc"
if grep -q "aborted without changing the service" "${SANDBOX}/portfail.log"; then
	pass "port failure message is explicit"
else
	fail "port failure message is explicit"
fi
if [ ! -e "${BIN_DIR}/${HUB_BIN}.bak" ] && [ ! -e "${BIN_DIR}/${HUB_BIN}.new" ]; then
	pass "port failure makes zero changes"
else
	fail "port failure makes zero changes"
fi

printf '== prompt5: rollback paths ==\n'
t_reset_paths
mock_reset
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
t_prep_agent "agent-v1-content"
printf '1' > "${MOCKCTL}/loaded_agent"
printf '4100' > "${MOCKCTL}/pid_agent"
printf '1' > "${MOCKCTL}/load_fail_remaining"
if ( transact_agent_update "v0.19.0-ios.2" > "${SANDBOX}/rb1.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "load failure exits non-zero" "1" "$_rc"
if grep -q "Rolling back" "${SANDBOX}/rb1.log" && grep -q "Rollback succeeded" "${SANDBOX}/rb1.log"; then
	pass "rollback announced and confirmed"
else
	fail "rollback announced and confirmed"
fi
if cmp -s "${BIN_DIR}/${AGENT_BIN}" "${SANDBOX}/agent-v1-ref"; then
	pass "rollback restores exact previous binary"
else
	fail "rollback restores exact previous binary"
fi
assert_eq "service loaded again after rollback" "1" "$(cat "${MOCKCTL}/loaded_agent")"
expect_fail "failed update commits no state" state_agent_release
BESZEL_TEST_NO_PID=1
if ( transact_agent_update "v0.19.0-ios.2" > "${SANDBOX}/rb2.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "dead service exits non-zero" "1" "$_rc"
if grep -q "CRITICAL: automatic rollback failed" "${SANDBOX}/rb2.log"; then
	pass "failed rollback reports CRITICAL"
else
	fail "failed rollback reports CRITICAL"
fi
if grep -q "${BIN_DIR}/${AGENT_BIN}" "${SANDBOX}/rb2.log" && grep -q "${BIN_DIR}/${AGENT_BIN}.bak" "${SANDBOX}/rb2.log" && grep -q "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/rb2.log"; then
	pass "CRITICAL message names binary, backup and plist"
else
	fail "CRITICAL message names binary, backup and plist"
fi
if cmp -s "${BIN_DIR}/${AGENT_BIN}" "${SANDBOX}/agent-v1-ref" && [ -s "${BIN_DIR}/${AGENT_BIN}.bak" ]; then
	pass "failed rollback keeps old binary and retains backup"
else
	fail "failed rollback keeps old binary and retains backup"
fi
BESZEL_TEST_NO_PID=0

printf '== prompt5: hub rollback keeps old state ==\n'
t_reset_paths
mock_reset
t_make_release "v0.19.0-ios.1" "agent-v1-content" "hub-v1-content"
fetch_sums > /dev/null 2>&1
_T_OLD_SHA_H=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$HUB_ASSET")
expect_ok "record old hub release" state_write_component "hub" "v0.19.0-ios.1" "$_T_OLD_SHA_H"
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
t_prep_hub "hub-v1-content" "8090"
mkdir -p "${LIB_DIR}/beszel-hub"
printf 'precious-hub-database-bytes' > "${LIB_DIR}/beszel-hub/db.sqlite"
printf '1' > "${MOCKCTL}/loaded_hub"
printf '30' > "${MOCKCTL}/health_fail_remaining"
if ( transact_hub_update "v0.19.0-ios.2" > "${SANDBOX}/rb3.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "unhealthy hub exits non-zero" "1" "$_rc"
if grep -q "Rolling back" "${SANDBOX}/rb3.log" && grep -q "Rollback succeeded" "${SANDBOX}/rb3.log"; then
	pass "hub rollback announced and confirmed"
else
	fail "hub rollback announced and confirmed"
fi
if cmp -s "${BIN_DIR}/${HUB_BIN}" "${SANDBOX}/hub-v1-ref"; then
	pass "hub rollback restores exact previous binary"
else
	fail "hub rollback restores exact previous binary"
fi
assert_eq "old hub state retained after failure" "v0.19.0-ios.1" "$(state_hub_release)"
if grep -q "precious-hub-database-bytes" "${LIB_DIR}/beszel-hub/db.sqlite"; then
	pass "failed hub update never touches database"
else
	fail "failed hub update never touches database"
fi
printf '30' > "${MOCKCTL}/health_fail_remaining"
BESZEL_TEST_HEALTH=fail
if ( transact_hub_update "v0.19.0-ios.2" > "${SANDBOX}/rb4.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "unrecoverable hub exits non-zero" "1" "$_rc"
BESZEL_TEST_HEALTH=ok
if grep -q "CRITICAL: automatic rollback failed" "${SANDBOX}/rb4.log"; then
	pass "hub rollback failure reports CRITICAL"
else
	fail "hub rollback failure reports CRITICAL"
fi
if cmp -s "${BIN_DIR}/${HUB_BIN}" "${SANDBOX}/hub-v1-ref" && [ -s "${BIN_DIR}/${HUB_BIN}.bak" ]; then
	pass "hub CRITICAL keeps binary and backup"
else
	fail "hub CRITICAL keeps binary and backup"
fi
assert_eq "hub state still old after CRITICAL" "v0.19.0-ios.1" "$(state_hub_release)"
printf '99' > "${MOCKCTL}/load_fail_remaining"
if ( transact_hub_update "v0.19.0-ios.2" > "${SANDBOX}/rb5.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "rollback load failure exits non-zero" "1" "$_rc"
rm -f "${MOCKCTL}/load_fail_remaining"
if grep -q "CRITICAL: automatic rollback failed" "${SANDBOX}/rb5.log"; then
	pass "rollback load failure reports CRITICAL"
else
	fail "rollback load failure reports CRITICAL"
fi
BESZEL_TEST_UNLOAD_FAIL=1
rm -f "${MOCKCTL}/load_fail_remaining"
t_prep_agent "agent-v1-content"
printf '1' > "${MOCKCTL}/loaded_agent"
printf '4100' > "${MOCKCTL}/pid_agent"
if ( transact_agent_update "v0.19.0-ios.2" > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
BESZEL_TEST_UNLOAD_FAIL=0
assert_eq "unload hiccup still updates" "0" "$_rc"
if grep -q "agent-v2-content" "${BIN_DIR}/${AGENT_BIN}"; then
	pass "update proceeds past unload warning"
else
	fail "update proceeds past unload warning"
fi

printf '== prompt5: update both ==\n'
t_reset_paths
mock_reset
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
t_prep_agent "agent-v1-content"
t_prep_hub "hub-v1-content" "8090"
printf '1' > "${MOCKCTL}/loaded_agent"
printf '1' > "${MOCKCTL}/loaded_hub"
if ( update_both_flow "v0.19.0-ios.2" > "${SANDBOX}/both.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "both update succeeds" "0" "$_rc"
_T_HUB_LOAD_LINE=$(grep -n "^load .*dev.beszel.hub.plist" "${MOCKCTL}/mock.log" | head -n 1 | cut -d: -f1)
_T_AGENT_LOAD_LINE=$(grep -n "^load .*dev.beszel.agent.plist" "${MOCKCTL}/mock.log" | head -n 1 | cut -d: -f1)
if [ -n "$_T_HUB_LOAD_LINE" ] && [ -n "$_T_AGENT_LOAD_LINE" ] && [ "$_T_HUB_LOAD_LINE" -lt "$_T_AGENT_LOAD_LINE" ]; then
	pass "both updates hub before agent"
else
	fail "both updates hub before agent"
fi
assert_eq "both commits hub state" "v0.19.0-ios.2" "$(state_hub_release)"
assert_eq "both commits agent state" "v0.19.0-ios.2" "$(state_agent_release)"
t_reset_paths
mock_reset
t_make_release "v0.19.0-ios.1" "agent-v1-content" "hub-v1-content"
fetch_sums > /dev/null 2>&1
_T_OA=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$AGENT_ASSET")
_T_OH=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$HUB_ASSET")
state_write_component "agent" "v0.19.0-ios.1" "$_T_OA" > /dev/null
state_write_component "hub" "v0.19.0-ios.1" "$_T_OH" > /dev/null
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
t_prep_agent "agent-v1-content"
t_prep_hub "hub-v1-content" "8090"
printf '1' > "${MOCKCTL}/loaded_agent"
printf '1' > "${MOCKCTL}/loaded_hub"
printf '30' > "${MOCKCTL}/health_fail_remaining"
if ( update_both_flow "v0.19.0-ios.2" > "${SANDBOX}/bothfail.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "both aborts when hub fails" "1" "$_rc"
if grep -q "agent-v1-content" "${BIN_DIR}/${AGENT_BIN}" && [ ! -e "${BIN_DIR}/${AGENT_BIN}.bak" ]; then
	pass "hub failure leaves agent completely untouched"
else
	fail "hub failure leaves agent completely untouched"
fi
assert_eq "agent state untouched after hub failure" "v0.19.0-ios.1" "$(state_agent_release)"
assert_eq "hub state untouched after hub failure" "v0.19.0-ios.1" "$(state_hub_release)"
rm -f "${SB_WORK}/SHA256SUMS" "${SB_WORK}/beszel-agent-ios-arm64" "${SB_WORK}/beszel-hub-ios-arm64"
if ( update_both_flow "v0.19.0-ios.1" > "${SANDBOX}/bothnoop.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "both all-current is a no-op" "0" "$_rc"
if [ -e "${SB_WORK}/SHA256SUMS" ]; then
	fail "both all-current downloads nothing"
else
	pass "both all-current downloads nothing"
fi
printf '1' > "${MOCKCTL}/load_fail_remaining_agent"
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
if ( transact_hub_update "v0.19.0-ios.2" > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "hub part succeeds" "0" "$_rc"
if ( transact_agent_update "v0.19.0-ios.2" > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "agent part fails and rolls back" "1" "$_rc"
rm -f "${MOCKCTL}/load_fail_remaining_agent"
if grep -q "hub-v2-content" "${BIN_DIR}/${HUB_BIN}" && grep -q "agent-v1-content" "${BIN_DIR}/${AGENT_BIN}"; then
	pass "successful hub kept while agent rolled back"
else
	fail "successful hub kept while agent rolled back"
fi
assert_eq "hub state new, agent state old" "v0.19.0-ios.2" "$(state_hub_release)"
assert_eq "agent state still old" "v0.19.0-ios.1" "$(state_agent_release)"

printf '== prompt5: update safety static checks ==\n'
t_body() {
	sed -n "/^$1()/,/^}/p" "$SCRIPT"
}
if grep -q "4) Update" "$SCRIPT" && grep -q "5) Repair / Reconfigure" "$SCRIPT" && grep -q "6) Exit" "$SCRIPT"; then
	pass "menu offers Update, Repair / Reconfigure and Exit"
else
	fail "menu offers Update, Repair / Reconfigure and Exit"
fi
if grep -q "Remove Agent\|Remove Hub\|) Uninstall\|uninstall_menu\|do_uninstall\|purge data" "$SCRIPT"; then
	fail "no uninstall exposed yet"
else
	pass "no uninstall exposed yet"
fi
if grep -q "transactional updates" "$SCRIPT"; then
	pass "header describes transactional updates"
else
	fail "header describes transactional updates"
fi
if t_body transact_agent_update | grep -q "prompt_agent_key\|AGENT_KEY\|ask_tty\|read_tty"; then
	fail "agent update never prompts for key"
else
	pass "agent update never prompts for key"
fi
if t_body transact_agent_update | grep -q "write_agent_plist\|install_plist"; then
	fail "agent update never rewrites plist"
else
	pass "agent update never rewrites plist"
fi
if t_body transact_hub_update | grep -q "write_hub_plist\|install_plist\|DEFAULT_HUB_PORT"; then
	fail "hub update never rewrites plist"
else
	pass "hub update never rewrites plist"
fi
# shellcheck disable=SC2016
if grep -F -q 'compute_sha256 "${BIN_DIR}' "$SCRIPT" || grep -F -q 'verify_file "${BIN_DIR}' "$SCRIPT"; then
	fail "no installed-binary hash version detection"
else
	pass "no installed-binary hash version detection"
fi
if grep -F -q "agent health" "$SCRIPT"; then
	fail "agent health subcommand not used for verification"
else
	pass "agent health subcommand not used for verification"
fi
_T_HUB_TX_LINE=$(t_body update_both_flow | grep -n "transact_hub_update" | head -n 1 | cut -d: -f1)
_T_AGENT_TX_LINE=$(t_body update_both_flow | grep -n "transact_agent_update" | head -n 1 | cut -d: -f1)
if [ -n "$_T_HUB_TX_LINE" ] && [ -n "$_T_AGENT_TX_LINE" ] && [ "$_T_HUB_TX_LINE" -lt "$_T_AGENT_TX_LINE" ]; then
	pass "both flow orders hub before agent"
else
	fail "both flow orders hub before agent"
fi
_T_CUR_LINE=$(t_body update_agent_flow | grep -n "release_is_current" | head -n 1 | cut -d: -f1)
_T_CONF_LINE=$(t_body update_agent_flow | grep -n "confirm_update" | head -n 1 | cut -d: -f1)
if [ -n "$_T_CUR_LINE" ] && [ -n "$_T_CONF_LINE" ] && [ "$_T_CUR_LINE" -lt "$_T_CONF_LINE" ]; then
	pass "currency checked before confirmation"
else
	fail "currency checked before confirmation"
fi
if t_body do_install_agent | grep -q "state_write_component" && t_body do_install_hub | grep -q "state_write_component"; then
	pass "fresh installs record state"
else
	fail "fresh installs record state"
fi
_T_WAIT_LINE=$(t_body do_install_hub | grep -n "wait_for_hub" | head -n 1 | cut -d: -f1)
_T_STATE_LINE=$(t_body do_install_hub | grep -n "state_write_component" | head -n 1 | cut -d: -f1)
if [ -n "$_T_WAIT_LINE" ] && [ -n "$_T_STATE_LINE" ] && [ "$_T_WAIT_LINE" -lt "$_T_STATE_LINE" ]; then
	pass "hub state recorded only after health success"
else
	fail "hub state recorded only after health success"
fi
if grep -q "update_signal_trap" "$SCRIPT" && grep -q "_UPDATE_NEED_ROLLBACK" "$SCRIPT"; then
	pass "interrupt rollback traps present"
else
	fail "interrupt rollback traps present"
fi
if grep -q "install-state" "$SCRIPT" && grep -q '\.bak' "$SCRIPT"; then
	pass "state and backup paths present"
else
	fail "state and backup paths present"
fi

printf '== prompt6: test doubles ==\n'
# shellcheck disable=SC2329
ask_tty() {
	_stub_var="$2"
	_stub_ans=""
	if IFS= read -r _stub_ans < "${MOCKCTL}/ask_queue" 2> /dev/null; then
		:
	else
		_stub_ans=""
	fi
	tail -n +2 "${MOCKCTL}/ask_queue" > "${MOCKCTL}/ask_queue.tmp" 2> /dev/null || true
	mv -f "${MOCKCTL}/ask_queue.tmp" "${MOCKCTL}/ask_queue" 2> /dev/null || true
	printf '%s' "$_stub_ans" > "${MOCKCTL}/ask_one"
	# Intentional indirection: read assigns to the variable NAMED by the stub arg.
	# shellcheck disable=SC2229
	IFS= read -r "$_stub_var" < "${MOCKCTL}/ask_one" || true
	# Fail closed on prompt starvation: never spin forever waiting for
	# answers the test did not queue (production reads a human on /dev/tty).
	_stub_empty_n=0
	if [ -z "$_stub_ans" ]; then
		_stub_empty_n=$(cat "${MOCKCTL}/ask_empty_n" 2> /dev/null || true)
		case "$_stub_empty_n" in '' | *[!0-9]*) _stub_empty_n=0 ;; esac
		_stub_empty_n=$((_stub_empty_n + 1))
		printf '%s' "$_stub_empty_n" > "${MOCKCTL}/ask_empty_n"
	else
		printf '0' > "${MOCKCTL}/ask_empty_n"
	fi
	if [ "$_stub_empty_n" -ge 3 ]; then
		return 1
	fi
	return 0
}
ask_queue_set() {
	: > "${MOCKCTL}/ask_queue"
	for _qa in "$@"; do
		printf '%s\n' "$_qa" >> "${MOCKCTL}/ask_queue"
	done
}
# The confirm stubs below are invoked indirectly through the repair and
# reconfigure flows exercised in subshells.
# shellcheck disable=SC2329
confirm_update() { return 0; }
# shellcheck disable=SC2329
confirm_destructive() { return 0; }
p6_reset() {
	mock_reset
	: > "${MOCKCTL}/ask_queue"
	printf '0' > "${MOCKCTL}/ask_empty_n"
	BESZEL_TEST_PLIST_VALID=ok
	BESZEL_TEST_LIST_FAIL=0
}
P6_K1="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqM66/yBCvP5nLv8mQuczlB9lXh9B7 p6-old-host"
P6_K2="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqM66/yBCvP5nLv8mQuczlB9lXh9B7 p6-new-host"
P6_KNASTY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqM66/yBCvP5nLv8mQuczlB9lXh9B7 amp & <lab> \"q\" 's'"
P6_KDIAG="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqM66/yBCvP5nLv8mQuczlB9lXh9B7 diagmarker-UNIQUE-7Z9Q"
p6_prep_agent_full() {
	printf '%s' "$1" > "${BIN_DIR}/${AGENT_BIN}"
	chmod 755 "${BIN_DIR}/${AGENT_BIN}"
	write_agent_plist "$3" "$2" "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
}

printf '== prompt6: component state ==\n'
t_reset_paths
p6_reset
assert_eq "agent ABSENT when empty" "ABSENT" "$(component_state agent)"
assert_eq "hub ABSENT when empty" "ABSENT" "$(component_state hub)"
t_prep_agent "agent-bytes"
assert_eq "agent COMPLETE with binary+plist" "COMPLETE" "$(component_state agent)"
rm "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
assert_eq "agent BINARY_ONLY without plist" "BINARY_ONLY" "$(component_state agent)"
rm "${BIN_DIR}/${AGENT_BIN}"
touch "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
assert_eq "agent PLIST_ONLY without binary" "PLIST_ONLY" "$(component_state agent)"
printf 'x' > "${SANDBOX}/p6-target"
ln -s "${SANDBOX}/p6-target" "${BIN_DIR}/${AGENT_BIN}"
assert_eq "symlinked binary is not trusted" "PLIST_ONLY" "$(component_state agent)"
rm -f "${BIN_DIR}/${AGENT_BIN}"
printf 'agent-bytes' > "${BIN_DIR}/${AGENT_BIN}"
rm -f "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
ln -s "${SANDBOX}/p6-target" "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
assert_eq "symlinked plist is not trusted" "BINARY_ONLY" "$(component_state agent)"
rm -f "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
t_prep_hub "hub-bytes" "8090"
assert_eq "hub COMPLETE with binary+plist" "COMPLETE" "$(component_state hub)"
rm "${LAUNCHD_DIR}/${HUB_LABEL}.plist"
assert_eq "hub BINARY_ONLY without plist" "BINARY_ONLY" "$(component_state hub)"
expect_fail "unknown component rejected" component_state "hubbub"

printf '== prompt6: service inspection ==\n'
t_reset_paths
p6_reset
printf '1' > "${MOCKCTL}/loaded_agent"
printf '4321' > "${MOCKCTL}/pid_agent"
expect_ok "loaded service detected" svc_loaded "$AGENT_LABEL"
assert_eq "live PID reported" "4321" "$(svc_pid "$AGENT_LABEL")"
printf '-' > "${MOCKCTL}/pid_agent"
expect_ok "stopped job still counts as loaded" svc_loaded "$AGENT_LABEL"
assert_eq "stopped job has no PID" "none" "$(svc_pid "$AGENT_LABEL")"
printf '0' > "${MOCKCTL}/loaded_agent"
expect_fail "unlisted service not loaded" svc_loaded "$AGENT_LABEL"
assert_eq "unlisted service PID none" "none" "$(svc_pid "$AGENT_LABEL")"
BESZEL_TEST_LIST_FAIL=1
expect_fail "unreadable launchctl not loaded" svc_loaded "$AGENT_LABEL"
assert_eq "unreadable launchctl PID none" "none" "$(svc_pid "$AGENT_LABEL")"
BESZEL_TEST_LIST_FAIL=0

printf '== prompt6: diagnostics ==\n'
t_reset_paths
p6_reset
t_make_release "v0.19.0-ios.1" "agent-v1" "hub-v1"
fetch_sums > /dev/null 2>&1
_T_DA_SHA=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$AGENT_ASSET")
_T_DH_SHA=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$HUB_ASSET")
p6_prep_agent_full "agent-v1-content" "45876" "$P6_KDIAG"
t_prep_hub "hub-v1-content" "8123"
state_write_component "agent" "v0.19.0-ios.1" "$_T_DA_SHA" > /dev/null
state_write_component "hub" "v0.19.0-ios.1" "$_T_DH_SHA" > /dev/null
mkdir -p "${LIB_DIR}/beszel-agent" "${LIB_DIR}/beszel-hub"
printf 'agent-data-marker' > "${LIB_DIR}/beszel-agent/marker"
printf 'hub-db-marker' > "${LIB_DIR}/beszel-hub/db.sqlite"
printf '1' > "${MOCKCTL}/loaded_agent"
printf '4321' > "${MOCKCTL}/pid_agent"
printf '1' > "${MOCKCTL}/loaded_hub"
printf '4322' > "${MOCKCTL}/pid_hub"
cp "${BIN_DIR}/${AGENT_BIN}" "${SANDBOX}/diag-agent-ref"
cp "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/diag-apl-ref"
cp "${BIN_DIR}/${HUB_BIN}" "${SANDBOX}/diag-hub-ref"
_D_OUT=$(diagnose_agent)
_D_RC=$?
assert_eq "agent diagnostics exit zero" "0" "$_D_RC"
for _d_need in "regular file" "XML valid" "PID 4321" "Agent port: 45876" "Agent key: configured" "v0.19.0-ios.1" "Data directory: present"; do
	case "$_D_OUT" in
		*"$_d_need"*) pass "agent diag shows: $_d_need" ;;
		*) fail "agent diag shows: $_d_need" ;;
	esac
done
case "$_D_OUT" in
	*diagmarker-UNIQUE-7Z9Q*) fail "agent key value never printed" ;;
	*) pass "agent key value never printed" ;;
esac
_H_OUT=$(diagnose_hub)
for _d_need in "regular file" "XML valid" "PID 4322" "Hub port: 8123" "reachable" "v0.19.0-ios.1" "Data directory: present"; do
	case "$_H_OUT" in
		*"$_d_need"*) pass "hub diag shows: $_d_need" ;;
		*) fail "hub diag shows: $_d_need" ;;
	esac
done
if cmp -s "${BIN_DIR}/${AGENT_BIN}" "${SANDBOX}/diag-agent-ref" && cmp -s "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/diag-apl-ref" && cmp -s "${BIN_DIR}/${HUB_BIN}" "${SANDBOX}/diag-hub-ref"; then
	pass "diagnostics change no files"
else
	fail "diagnostics change no files"
fi
if [ "$(cat "${MOCKCTL}/loaded_agent")" = "1" ] && [ "$(cat "${MOCKCTL}/loaded_hub")" = "1" ] && [ ! -e "${BIN_DIR}/${AGENT_BIN}.bak" ]; then
	pass "diagnostics change no service state"
else
	fail "diagnostics change no service state"
fi
if grep -q "hub-db-marker" "${LIB_DIR}/beszel-hub/db.sqlite"; then
	pass "diagnostics leave hub data alone"
else
	fail "diagnostics leave hub data alone"
fi
rm -f "$(state_path)"
_D_LEGACY=$(diagnose_agent)
case "$_D_LEGACY" in
	*"legacy installation"*) pass "legacy release displayed" ;;
	*) fail "legacy release displayed" ;;
esac
BESZEL_TEST_LIST_FAIL=1
_D_UNK=$(diagnose_hub)
case "$_D_UNK" in
	*"load state unknown"*) pass "unreadable launchctl reported" ;;
	*) fail "unreadable launchctl reported" ;;
esac
BESZEL_TEST_LIST_FAIL=0

printf '== prompt6: plist parsing ==\n'
t_reset_paths
p6_reset
write_agent_plist "$P6_KNASTY" "45999" "${LAUNCHD_DIR}/p6-agent.plist"
assert_eq "LISTEN extraction" "45999" "$(agent_port_from_plist "${LAUNCHD_DIR}/p6-agent.plist")"
assert_eq "KEY entity round-trip" "$P6_KNASTY" "$(agent_key_from_plist "${LAUNCHD_DIR}/p6-agent.plist")"
t_prep_hub "hub-bytes" "8213"
assert_eq "hub port extraction" "8213" "$(hub_port_from_plist "${LAUNCHD_DIR}/${HUB_LABEL}.plist")"
printf '<string>0.0.0.0:8213</string>\n' >> "${LAUNCHD_DIR}/${HUB_LABEL}.plist"
assert_eq "duplicated identical hub port accepted" "8213" "$(hub_port_from_plist "${LAUNCHD_DIR}/${HUB_LABEL}.plist")"
printf '<string>0.0.0.0:9999</string>\n' >> "${LAUNCHD_DIR}/${HUB_LABEL}.plist"
expect_fail "ambiguous hub ports rejected" hub_port_from_plist "${LAUNCHD_DIR}/${HUB_LABEL}.plist"
cat "${LAUNCHD_DIR}/p6-agent.plist" "${LAUNCHD_DIR}/p6-agent.plist" > "${LAUNCHD_DIR}/p6-dup.plist"
expect_fail "duplicated KEY rejected" agent_key_from_plist "${LAUNCHD_DIR}/p6-dup.plist"
expect_fail "duplicated LISTEN rejected" agent_port_from_plist "${LAUNCHD_DIR}/p6-dup.plist"
grep -v "<key>KEY</key>" "${LAUNCHD_DIR}/p6-agent.plist" > "${LAUNCHD_DIR}/p6-nokey.plist"
expect_fail "missing KEY rejected" agent_key_from_plist "${LAUNCHD_DIR}/p6-nokey.plist"
head -c 200 "${LAUNCHD_DIR}/p6-agent.plist" > "${LAUNCHD_DIR}/p6-trunc.plist"
expect_fail "truncated plist KEY rejected" agent_key_from_plist "${LAUNCHD_DIR}/p6-trunc.plist"
expect_fail "truncated plist port rejected" agent_port_from_plist "${LAUNCHD_DIR}/p6-trunc.plist"
write_agent_plist "not-a-key" "45876" "${LAUNCHD_DIR}/p6-badkey.plist"
expect_fail "non-key KEY rejected" agent_key_from_plist "${LAUNCHD_DIR}/p6-badkey.plist"
write_agent_plist "$P6_K1" "99999" "${LAUNCHD_DIR}/p6-badport.plist"
expect_fail "out-of-range LISTEN rejected" agent_port_from_plist "${LAUNCHD_DIR}/p6-badport.plist"
sed 's|<string>:45999</string>|<string>45999</string>|' "${LAUNCHD_DIR}/p6-agent.plist" > "${LAUNCHD_DIR}/p6-nocolon.plist"
expect_fail "colon-less LISTEN rejected" agent_port_from_plist "${LAUNCHD_DIR}/p6-nocolon.plist"
expect_fail "missing file rejected" agent_key_from_plist "${LAUNCHD_DIR}/nope.plist"
# shellcheck disable=SC2016
write_agent_plist 'ssh-ed25519 $(touch ${SANDBOX}/pwned-parse)/comment' "45876" "${LAUNCHD_DIR}/p6-evil.plist"
expect_fail "command-looking KEY rejected" agent_key_from_plist "${LAUNCHD_DIR}/p6-evil.plist"
if [ -e "${SANDBOX}/pwned-parse" ]; then
	fail "plist parsing never executes content"
else
	pass "plist parsing never executes content"
fi
mv "${MOCKBIN}/plutil" "${MOCKBIN}/plutil.hidden"
expect_ok "real xml checker accepts good plist" plist_valid "${LAUNCHD_DIR}/p6-agent.plist"
expect_fail "real xml checker rejects garbage" plist_valid "${LAUNCHD_DIR}/p6-trunc.plist"
mv "${MOCKBIN}/plutil.hidden" "${MOCKBIN}/plutil"
BESZEL_TEST_PLIST_VALID=fail
expect_fail "checker failure fails validation" plist_valid "${LAUNCHD_DIR}/p6-agent.plist"
BESZEL_TEST_PLIST_VALID=ok

printf '== prompt6: agent reconfigure ==\n'
t_reset_paths
p6_reset
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
_T_RA_SHA=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$AGENT_ASSET")
p6_prep_agent_full "agent-v1-content" "45876" "$P6_K1"
state_write_component "agent" "v0.19.0-ios.1" "$_T_RA_SHA" > /dev/null
mkdir -p "${LIB_DIR}/beszel-agent"
printf 'agent-data-marker' > "${LIB_DIR}/beszel-agent/marker"
cp "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/ra-plist-orig"
printf '1' > "${MOCKCTL}/loaded_agent"
printf '4100' > "${MOCKCTL}/pid_agent"
ask_queue_set "45900"
if ( reconfigure_agent_flow > "${SANDBOX}/ra1.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "keep-key change-port succeeds" "0" "$_rc"
assert_eq "port changed" "45900" "$(agent_port_from_plist "${LAUNCHD_DIR}/${AGENT_LABEL}.plist")"
assert_eq "key kept" "$P6_K1" "$(agent_key_from_plist "${LAUNCHD_DIR}/${AGENT_LABEL}.plist")"
if cmp -s "${BIN_DIR}/${AGENT_BIN}" "${SANDBOX}/diag-agent-ref" 2> /dev/null || grep -q "agent-v1-content" "${BIN_DIR}/${AGENT_BIN}"; then
	pass "reconfigure leaves binary untouched"
else
	fail "reconfigure leaves binary untouched"
fi
if cmp -s "${LAUNCHD_DIR}/${AGENT_LABEL}.plist.bak" "${SANDBOX}/ra-plist-orig"; then
	pass "plist backup holds previous config"
else
	fail "plist backup holds previous config"
fi
if grep -q "agent-data-marker" "${LIB_DIR}/beszel-agent/marker"; then
	pass "reconfigure leaves data alone"
else
	fail "reconfigure leaves data alone"
fi
assert_eq "reconfigure never changes state" "v0.19.0-ios.1" "$(state_agent_release)"
# shellcheck disable=SC2329
confirm_update() { return 1; }
# shellcheck disable=SC2329
confirm_destructive() { return 1; }
ask_queue_set "$P6_K2"
cp "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/ra-plist-45900"
if ( reconfigure_agent_flow > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "change-key keep-port succeeds" "0" "$_rc"
assert_eq "port kept" "45900" "$(agent_port_from_plist "${LAUNCHD_DIR}/${AGENT_LABEL}.plist")"
assert_eq "key changed" "$P6_K2" "$(agent_key_from_plist "${LAUNCHD_DIR}/${AGENT_LABEL}.plist")"
# shellcheck disable=SC2329
confirm_update() { return 1; }
# shellcheck disable=SC2329
confirm_destructive() { return 0; }
ask_queue_set "45901" "$P6_K1"
if ( reconfigure_agent_flow > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "change-both succeeds" "0" "$_rc"
assert_eq "both: port changed" "45901" "$(agent_port_from_plist "${LAUNCHD_DIR}/${AGENT_LABEL}.plist")"
assert_eq "both: key changed" "$P6_K1" "$(agent_key_from_plist "${LAUNCHD_DIR}/${AGENT_LABEL}.plist")"
# shellcheck disable=SC2329
confirm_update() { return 0; }
# shellcheck disable=SC2329
confirm_destructive() { return 1; }
cp "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/ra-plist-before-noop"
rm -f "${LAUNCHD_DIR}/${AGENT_LABEL}.plist.bak"
if ( reconfigure_agent_flow > "${SANDBOX}/ra-noop.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "no-op exits zero" "0" "$_rc"
if grep -q "No Agent configuration changes requested" "${SANDBOX}/ra-noop.log"; then
	pass "no-op message shown"
else
	fail "no-op message shown"
fi
if cmp -s "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/ra-plist-before-noop" && [ ! -e "${LAUNCHD_DIR}/${AGENT_LABEL}.plist.bak" ]; then
	pass "no-op changes nothing"
else
	fail "no-op changes nothing"
fi
# shellcheck disable=SC2329
confirm_destructive() { return 0; }
printf '1' > "${MOCKCTL}/load_fail_remaining"
cp "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/ra-plist-pre-fail"
ask_queue_set "45999"
if ( reconfigure_agent_flow > "${SANDBOX}/ra-rb.log" 2>&1 ); then _rc=0; else _rc=$?; fi
rm -f "${MOCKCTL}/load_fail_remaining"
assert_eq "load failure exits non-zero" "1" "$_rc"
if grep -q "Configuration rollback succeeded" "${SANDBOX}/ra-rb.log"; then
	pass "plist rollback announced"
else
	fail "plist rollback announced"
fi
if cmp -s "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/ra-plist-pre-fail"; then
	pass "plist rollback restores exact config"
else
	fail "plist rollback restores exact config"
fi
assert_eq "service running after plist rollback" "1" "$(cat "${MOCKCTL}/loaded_agent")"
assert_eq "state kept through failed reconfigure" "v0.19.0-ios.1" "$(state_agent_release)"
BESZEL_TEST_NO_PID=1
ask_queue_set "45998"
if ( reconfigure_agent_flow > "${SANDBOX}/ra-rb2.log" 2>&1 ); then _rc=0; else _rc=$?; fi
BESZEL_TEST_NO_PID=0
assert_eq "dead service exits non-zero" "1" "$_rc"
if grep -q "CRITICAL: configuration rollback failed" "${SANDBOX}/ra-rb2.log"; then
	pass "failed plist rollback reports CRITICAL"
else
	fail "failed plist rollback reports CRITICAL"
fi
if cmp -s "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/ra-plist-pre-fail" && [ -s "${LAUNCHD_DIR}/${AGENT_LABEL}.plist.bak" ]; then
	pass "CRITICAL keeps restored plist and backup"
else
	fail "CRITICAL keeps restored plist and backup"
fi
printf '99' > "${MOCKCTL}/load_fail_remaining"
ask_queue_set "45997"
if ( reconfigure_agent_flow > "${SANDBOX}/ra-rb3.log" 2>&1 ); then _rc=0; else _rc=$?; fi
rm -f "${MOCKCTL}/load_fail_remaining"
assert_eq "rollback load failure exits non-zero" "1" "$_rc"
if grep -q "CRITICAL: configuration rollback failed" "${SANDBOX}/ra-rb3.log"; then
	pass "rollback load failure is CRITICAL"
else
	fail "rollback load failure is CRITICAL"
fi
rm -f "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
if ( reconfigure_agent_flow > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "incomplete install refused" "1" "$_rc"

printf '== prompt6: hub reconfigure ==\n'
t_reset_paths
p6_reset
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
_T_RH_SHA=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$HUB_ASSET")
t_prep_hub "hub-v1-content" "8090"
state_write_component "hub" "v0.19.0-ios.1" "$_T_RH_SHA" > /dev/null
mkdir -p "${LIB_DIR}/beszel-hub"
printf 'hub-db-marker' > "${LIB_DIR}/beszel-hub/db.sqlite"
cp "${LAUNCHD_DIR}/${HUB_LABEL}.plist" "${SANDBOX}/rh-plist-orig"
cp "${BIN_DIR}/${HUB_BIN}" "${SANDBOX}/rh-bin-orig"
printf '1' > "${MOCKCTL}/loaded_hub"
printf '4200' > "${MOCKCTL}/pid_hub"
ask_queue_set "8213"
if ( reconfigure_hub_flow > "${SANDBOX}/rh1.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "hub port change succeeds" "0" "$_rc"
assert_eq "new port active" "8213" "$(hub_port_from_plist "${LAUNCHD_DIR}/${HUB_LABEL}.plist")"
if grep -q ":8213/api/health" "${MOCKCTL}/health.log"; then
	pass "new port health-checked"
else
	fail "new port health-checked"
fi
if cmp -s "${BIN_DIR}/${HUB_BIN}" "${SANDBOX}/rh-bin-orig"; then
	pass "hub reconfigure leaves binary untouched"
else
	fail "hub reconfigure leaves binary untouched"
fi
if grep -q "hub-db-marker" "${LIB_DIR}/beszel-hub/db.sqlite"; then
	pass "hub reconfigure leaves database alone"
else
	fail "hub reconfigure leaves database alone"
fi
assert_eq "hub reconfigure never changes state" "v0.19.0-ios.1" "$(state_hub_release)"
if cmp -s "${LAUNCHD_DIR}/${HUB_LABEL}.plist.bak" "${SANDBOX}/rh-plist-orig"; then
	pass "hub plist backup holds previous config"
else
	fail "hub plist backup holds previous config"
fi
cp "${LAUNCHD_DIR}/${HUB_LABEL}.plist" "${SANDBOX}/rh-plist-8213"
ask_queue_set ""
if ( reconfigure_hub_flow > "${SANDBOX}/rh-noop.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "hub no-op exits zero" "0" "$_rc"
if grep -q "No Hub configuration changes requested" "${SANDBOX}/rh-noop.log"; then
	pass "hub no-op message shown"
else
	fail "hub no-op message shown"
fi
ask_queue_set "8222"
printf '30' > "${MOCKCTL}/health_fail_remaining"
if ( reconfigure_hub_flow > "${SANDBOX}/rh-rb.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "unhealthy new port exits non-zero" "1" "$_rc"
if grep -q "Configuration rollback succeeded" "${SANDBOX}/rh-rb.log"; then
	pass "hub plist rollback announced"
else
	fail "hub plist rollback announced"
fi
if cmp -s "${LAUNCHD_DIR}/${HUB_LABEL}.plist" "${SANDBOX}/rh-plist-8213"; then
	pass "hub rollback restores previous port"
else
	fail "hub rollback restores previous port"
fi
if grep -q "hub-db-marker" "${LIB_DIR}/beszel-hub/db.sqlite"; then
	pass "failed hub reconfigure leaves database alone"
else
	fail "failed hub reconfigure leaves database alone"
fi
assert_eq "hub state kept through failed reconfigure" "v0.19.0-ios.1" "$(state_hub_release)"
ask_queue_set "8223"
BESZEL_TEST_HEALTH=fail
if ( reconfigure_hub_flow > "${SANDBOX}/rh-rb2.log" 2>&1 ); then _rc=0; else _rc=$?; fi
BESZEL_TEST_HEALTH=ok
assert_eq "unrecoverable hub reconfig exits non-zero" "1" "$_rc"
if grep -q "CRITICAL: configuration rollback failed" "${SANDBOX}/rh-rb2.log"; then
	pass "hub rollback failure is CRITICAL"
else
	fail "hub rollback failure is CRITICAL"
fi
if cmp -s "${LAUNCHD_DIR}/${HUB_LABEL}.plist" "${SANDBOX}/rh-plist-8213"; then
	pass "CRITICAL keeps restored hub plist"
else
	fail "CRITICAL keeps restored hub plist"
fi

printf '== prompt6: agent repair ==\n'
# shellcheck disable=SC2329
confirm_update() { return 0; }
# shellcheck disable=SC2329
confirm_destructive() { return 0; }
t_reset_paths
p6_reset
t_prep_agent "agent-v1-content"
cp "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/rpa-plist-ref"
if ( repair_agent_flow > "${SANDBOX}/rpa1.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "restart-only repair succeeds" "0" "$_rc"
assert_eq "service loaded by restart" "1" "$(cat "${MOCKCTL}/loaded_agent")"
if cmp -s "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/rpa-plist-ref"; then
	pass "restart preserves plist bytes"
else
	fail "restart preserves plist bytes"
fi
if [ -e "${SB_WORK}/beszel-agent-ios-arm64" ] || [ -e "${SB_WORK}/SHA256SUMS" ]; then
	fail "restart downloads nothing"
else
	pass "restart downloads nothing"
fi
expect_fail "restart fabricates no release" state_agent_release
printf '1' > "${MOCKCTL}/loaded_agent"
printf '4321' > "${MOCKCTL}/pid_agent"
if ( repair_agent_flow > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "already-running repair is a no-op success" "0" "$_rc"
t_reset_paths
p6_reset
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
write_agent_plist "$P6_K1" "45876" "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
cp "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/rpa-plist-kept"
ask_queue_set "SENTINEL-UNTOUCHED"
if ( repair_agent_flow > "${SANDBOX}/rpa2.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "missing binary restored" "0" "$_rc"
if grep -q "agent-v2-content" "${BIN_DIR}/${AGENT_BIN}"; then
	pass "restored binary is Latest"
else
	fail "restored binary is Latest"
fi
if cmp -s "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/rpa-plist-kept"; then
	pass "restore preserves existing plist"
else
	fail "restore preserves existing plist"
fi
assert_eq "restore records Latest release" "v0.19.0-ios.2" "$(state_agent_release)"
assert_eq "service loaded after restore" "1" "$(cat "${MOCKCTL}/loaded_agent")"
if [ "$(cat "${MOCKCTL}/ask_queue")" = "SENTINEL-UNTOUCHED" ]; then
	pass "restore asks for no key"
else
	fail "restore asks for no key"
fi
t_reset_paths
p6_reset
t_prep_agent "agent-v1-content"
rm "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
assert_eq "repair working state is BINARY_ONLY" "BINARY_ONLY" "$(component_state agent)"
# shellcheck disable=SC2329
confirm_destructive() { return 1; }
if ( repair_agent_flow > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "cancelled recreation exits zero" "0" "$_rc"
if [ ! -e "${LAUNCHD_DIR}/${AGENT_LABEL}.plist.new" ] && grep -q "agent-v1-content" "${BIN_DIR}/${AGENT_BIN}"; then
	pass "cancelled recreation changes nothing"
else
	fail "cancelled recreation changes nothing"
fi
# shellcheck disable=SC2329
confirm_destructive() { return 0; }
ask_queue_set "$P6_K2" "45902"
cp "${BIN_DIR}/${AGENT_BIN}" "${SANDBOX}/rpa-bin-kept"
if ( repair_agent_flow > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "recreated config succeeds" "0" "$_rc"
assert_eq "recreated key stored" "$P6_K2" "$(agent_key_from_plist "${LAUNCHD_DIR}/${AGENT_LABEL}.plist")"
assert_eq "recreated port stored" "45902" "$(agent_port_from_plist "${LAUNCHD_DIR}/${AGENT_LABEL}.plist")"
if cmp -s "${BIN_DIR}/${AGENT_BIN}" "${SANDBOX}/rpa-bin-kept"; then
	pass "recreation keeps existing binary"
else
	fail "recreation keeps existing binary"
fi
expect_fail "recreation fabricates no release" state_agent_release
assert_eq "service loaded after recreation" "1" "$(cat "${MOCKCTL}/loaded_agent")"
t_reset_paths
p6_reset
if ( repair_agent_flow > "${SANDBOX}/rpa5.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "absent install rejected" "1" "$_rc"
if grep -q "not installed" "${SANDBOX}/rpa5.log"; then
	pass "absent message directs to install"
else
	fail "absent message directs to install"
fi
t_reset_paths
p6_reset
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
t_prep_agent "agent-v1-content"
cp "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/rpa-plist-broken"
printf 'agent-v1-content' > "${SANDBOX}/rpa-bin-broken"
printf '1' > "${MOCKCTL}/load_fail_remaining"
if ( repair_agent_flow > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
rm -f "${MOCKCTL}/load_fail_remaining"
assert_eq "broken binary replaced" "0" "$_rc"
if grep -q "agent-v2-content" "${BIN_DIR}/${AGENT_BIN}"; then
	pass "replacement binary is Latest"
else
	fail "replacement binary is Latest"
fi
if cmp -s "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/rpa-plist-broken"; then
	pass "replacement preserves config"
else
	fail "replacement preserves config"
fi
assert_eq "replacement records release" "v0.19.0-ios.2" "$(state_agent_release)"
t_reset_paths
p6_reset
t_make_release "v0.19.0-ios.1" "agent-v1-content" "hub-v1-content"
fetch_sums > /dev/null 2>&1
_T_RPA_OA=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$AGENT_ASSET")
state_write_component "agent" "v0.19.0-ios.1" "$_T_RPA_OA" > /dev/null
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
t_prep_agent "agent-v1-content"
cp "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/rpa-plist-r7"
printf 'agent-v1-content' > "${SANDBOX}/rpa-bin-r7"
printf '2' > "${MOCKCTL}/load_fail_remaining"
if ( repair_agent_flow > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
rm -f "${MOCKCTL}/load_fail_remaining"
assert_eq "failed replacement exits non-zero" "1" "$_rc"
if cmp -s "${BIN_DIR}/${AGENT_BIN}" "${SANDBOX}/rpa-bin-r7"; then
	pass "failed replacement rolls binary back"
else
	fail "failed replacement rolls binary back"
fi
if cmp -s "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/rpa-plist-r7"; then
	pass "failed replacement keeps config"
else
	fail "failed replacement keeps config"
fi
assert_eq "old state retained after failed repair" "v0.19.0-ios.1" "$(state_agent_release)"

printf '== prompt6: hub repair ==\n'
# shellcheck disable=SC2329
confirm_update() { return 0; }
# shellcheck disable=SC2329
confirm_destructive() { return 0; }
t_reset_paths
p6_reset
t_prep_hub "hub-v1-content" "8090"
cp "${LAUNCHD_DIR}/${HUB_LABEL}.plist" "${SANDBOX}/rph-plist-ref"
mkdir -p "${LIB_DIR}/beszel-hub"
printf 'hub-db-marker' > "${LIB_DIR}/beszel-hub/db.sqlite"
if ( repair_hub_flow > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "hub restart-only repair succeeds" "0" "$_rc"
assert_eq "hub loaded by restart" "1" "$(cat "${MOCKCTL}/loaded_hub")"
if cmp -s "${LAUNCHD_DIR}/${HUB_LABEL}.plist" "${SANDBOX}/rph-plist-ref"; then
	pass "hub restart preserves plist bytes"
else
	fail "hub restart preserves plist bytes"
fi
if [ -e "${SB_WORK}/beszel-hub-ios-arm64" ] || [ -e "${SB_WORK}/SHA256SUMS" ]; then
	fail "hub restart downloads nothing"
else
	pass "hub restart downloads nothing"
fi
expect_fail "hub restart fabricates no release" state_hub_release
if grep -q "hub-db-marker" "${LIB_DIR}/beszel-hub/db.sqlite"; then
	pass "hub restart leaves database alone"
else
	fail "hub restart leaves database alone"
fi
t_reset_paths
p6_reset
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
write_hub_plist "8090" "${LAUNCHD_DIR}/${HUB_LABEL}.plist"
cp "${LAUNCHD_DIR}/${HUB_LABEL}.plist" "${SANDBOX}/rph-plist-kept"
mkdir -p "${LIB_DIR}/beszel-hub"
printf 'hub-db-marker' > "${LIB_DIR}/beszel-hub/db.sqlite"
if ( repair_hub_flow > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "missing hub binary restored" "0" "$_rc"
if grep -q "hub-v2-content" "${BIN_DIR}/${HUB_BIN}"; then
	pass "restored hub is Latest"
else
	fail "restored hub is Latest"
fi
if cmp -s "${LAUNCHD_DIR}/${HUB_LABEL}.plist" "${SANDBOX}/rph-plist-kept"; then
	pass "hub restore preserves plist"
else
	fail "hub restore preserves plist"
fi
if grep -q "hub-db-marker" "${LIB_DIR}/beszel-hub/db.sqlite"; then
	pass "hub restore preserves database"
else
	fail "hub restore preserves database"
fi
assert_eq "hub restore records Latest" "v0.19.0-ios.2" "$(state_hub_release)"
if grep -q ":8090/api/health" "${MOCKCTL}/health.log"; then
	pass "restored hub health-checked on existing port"
else
	fail "restored hub health-checked on existing port"
fi
t_reset_paths
p6_reset
t_prep_hub "hub-v1-content" "8090"
rm "${LAUNCHD_DIR}/${HUB_LABEL}.plist"
assert_eq "hub repair working state is BINARY_ONLY" "BINARY_ONLY" "$(component_state hub)"
# shellcheck disable=SC2329
confirm_destructive() { return 1; }
if ( repair_hub_flow > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "hub recreation cancel exits zero" "0" "$_rc"
if [ ! -e "${LAUNCHD_DIR}/${HUB_LABEL}.plist" ]; then
	pass "hub cancel creates nothing"
else
	fail "hub cancel creates nothing"
fi
# shellcheck disable=SC2329
confirm_destructive() { return 0; }
mkdir -p "${LIB_DIR}/beszel-hub"
printf 'hub-db-marker' > "${LIB_DIR}/beszel-hub/db.sqlite"
cp "${BIN_DIR}/${HUB_BIN}" "${SANDBOX}/rph-bin-kept"
ask_queue_set "8222"
if ( repair_hub_flow > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "hub recreation succeeds" "0" "$_rc"
assert_eq "recreated hub port stored" "8222" "$(hub_port_from_plist "${LAUNCHD_DIR}/${HUB_LABEL}.plist")"
if cmp -s "${BIN_DIR}/${HUB_BIN}" "${SANDBOX}/rph-bin-kept"; then
	pass "hub recreation keeps binary"
else
	fail "hub recreation keeps binary"
fi
if grep -q "hub-db-marker" "${LIB_DIR}/beszel-hub/db.sqlite"; then
	pass "hub recreation keeps database"
else
	fail "hub recreation keeps database"
fi
expect_fail "hub recreation fabricates no release" state_hub_release
t_reset_paths
p6_reset
if ( repair_hub_flow > "${SANDBOX}/rph5.log" 2>&1 ); then _rc=0; else _rc=$?; fi
assert_eq "absent hub rejected" "1" "$_rc"
if grep -q "not installed" "${SANDBOX}/rph5.log"; then
	pass "absent hub message directs to install"
else
	fail "absent hub message directs to install"
fi
t_reset_paths
p6_reset
t_make_release "v0.19.0-ios.1" "agent-v1-content" "hub-v1-content"
fetch_sums > /dev/null 2>&1
_T_RPH_OH=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$HUB_ASSET")
state_write_component "hub" "v0.19.0-ios.1" "$_T_RPH_OH" > /dev/null
t_make_release "v0.19.0-ios.2" "agent-v2-content" "hub-v2-content"
fetch_sums > /dev/null 2>&1
t_prep_hub "hub-v1-content" "8090"
mkdir -p "${LIB_DIR}/beszel-hub"
printf 'hub-db-marker' > "${LIB_DIR}/beszel-hub/db.sqlite"
cp "${LAUNCHD_DIR}/${HUB_LABEL}.plist" "${SANDBOX}/rph-plist-r6"
printf 'hub-v1-content' > "${SANDBOX}/rph-bin-r6"
printf '1' > "${MOCKCTL}/load_fail_remaining"
printf '30' > "${MOCKCTL}/health_fail_remaining"
if ( repair_hub_flow > /dev/null 2>&1 ); then _rc=0; else _rc=$?; fi
rm -f "${MOCKCTL}/load_fail_remaining"
assert_eq "unhealthy hub repair exits non-zero" "1" "$_rc"
if cmp -s "${BIN_DIR}/${HUB_BIN}" "${SANDBOX}/rph-bin-r6"; then
	pass "failed hub repair rolls binary back"
else
	fail "failed hub repair rolls binary back"
fi
if cmp -s "${LAUNCHD_DIR}/${HUB_LABEL}.plist" "${SANDBOX}/rph-plist-r6"; then
	pass "failed hub repair keeps plist"
else
	fail "failed hub repair keeps plist"
fi
if grep -q "hub-db-marker" "${LIB_DIR}/beszel-hub/db.sqlite"; then
	pass "failed hub repair keeps database"
else
	fail "failed hub repair keeps database"
fi
assert_eq "old hub state retained" "v0.19.0-ios.1" "$(state_hub_release)"

printf '== prompt6: plist transaction signal trap ==\n'
t_reset_paths
p6_reset
p6_prep_agent_full "agent-v1-content" "45876" "$P6_K1"
expect_ok "plist backup for trap test" backup_plist "agent"
cp "${LAUNCHD_DIR}/${AGENT_LABEL}.plist.bak" "${SANDBOX}/trap-plist-good"
printf 'definitely-not-a-plist' > "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
printf '0' > "${MOCKCTL}/loaded_agent"
printf '-' > "${MOCKCTL}/pid_agent"
if BESZEL_INSTALL_LIB_ONLY=1 BESZEL_UPDATE_SETTLE_SECS=0 BESZEL_BIN_DIR="$SB_BIN" BESZEL_LIB_DIR="$SB_LIB" BESZEL_LAUNCHD_DIR="$SB_LAUNCHD" BESZEL_LOG_DIR="$SB_LOG" BESZEL_TEST_MOCKCTL="$MOCKCTL" PATH="${MOCKBIN}:$PATH" sh -c '. ./install.sh; _UPDATE_ACTIVE="agent-plist"; _UPDATE_NEED_ROLLBACK=1; update_signal_trap; printf "UNREACHABLE\n"' > "${SANDBOX}/trap.log" 2>&1; then _rc=0; else _rc=$?; fi
assert_eq "signal trap exits 130" "130" "$_rc"
if cmp -s "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" "${SANDBOX}/trap-plist-good"; then
	pass "signal trap restores known-good plist"
else
	fail "signal trap restores known-good plist"
fi
assert_eq "service reloaded by signal rollback" "1" "$(cat "${MOCKCTL}/loaded_agent")"

printf '== prompt6: repair/reconfigure safety statics ==\n'
if t_body reconfigure_agent_flow | grep -q "state_write_component\|backup_current_binary\|stage_new_binary\|fetch_and_verify_binary"; then
	fail "agent reconfigure touches no binary/state"
else
	pass "agent reconfigure touches no binary/state"
fi
if t_body reconfigure_hub_flow | grep -q "state_write_component\|backup_current_binary\|stage_new_binary\|fetch_and_verify_binary"; then
	fail "hub reconfigure touches no binary/state"
else
	pass "hub reconfigure touches no binary/state"
fi
if t_body restore_missing_binary | grep -q "prompt_agent_key\|prompt_agent_port\|prompt_hub_port"; then
	fail "binary restore asks for no config"
else
	pass "binary restore asks for no config"
fi
_T_RM_RF_OFFENDERS=$(grep -n "rm -rf" "$SCRIPT" | grep -v "WORK_DIR" | grep -v "SANDBOX" || true)
if [ -z "$_T_RM_RF_OFFENDERS" ]; then
	pass "rm -rf limited to work/sandbox paths"
else
	fail "rm -rf limited to work/sandbox paths"
fi
if grep -q "chown -R\|chmod -R" "$SCRIPT"; then
	fail "no recursive chown/chmod"
else
	pass "no recursive chown/chmod"
fi
if grep -q "Uninstall is NOT implemented" "$SCRIPT"; then
	pass "uninstall still marked unimplemented"
else
	fail "uninstall still marked unimplemented"
fi
if grep -q -i "repair.*not implemented\|reconfigure.*not implemented" "$SCRIPT"; then
	fail "no stale repair/reconfigure TODO text"
else
	pass "no stale repair/reconfigure TODO text"
fi
# shellcheck disable=SC2016
if t_body backup_plist | grep -F -q '_bp_bak="${_bp_plist}.bak"' && t_body rollback_plist | grep -F -q '_rp_bak="${_rp_final}.bak"'; then
	pass "fixed plist backup paths present"
else
	fail "fixed plist backup paths present"
fi

printf '== posix / safety static checks ==\n'
_first=$(head -n 1 "$SCRIPT")
assert_eq "shebang is /bin/sh" "#!/bin/sh" "$_first"
if grep -nE '(^|[^A-Za-z0-9_])(local|declare|function)[[:space:]]' "$SCRIPT" | grep -v '^.*#' > /dev/null; then
	fail "no local/declare/function keywords"
else
	pass "no local/declare/function keywords"
fi
if grep -nE '(^|[[:space:];])\[\[[[:space:]]|<\(|&>|[^[:space:]]select[[:space:]]' "$SCRIPT" > /dev/null; then
	fail "no bash-only operators"
else
	pass "no bash-only operators"
fi
if sed 's/#.*//' "$SCRIPT" | grep -nwE 'eval|source' > /dev/null; then
	fail "no eval/source"
else
	pass "no eval/source"
fi
if grep -n '`' "$SCRIPT" > /dev/null; then
	fail "no backtick substitution"
else
	pass "no backtick substitution"
fi
if grep -q 'INSTALLER_VERSION="0.3.0"' "$SCRIPT"; then
	pass "installer version constant present"
else
	fail "installer version constant present"
fi

printf '\n%d passed, %d failed\n' "$_pass" "$_fail"
[ "$_fail" = "0" ]
