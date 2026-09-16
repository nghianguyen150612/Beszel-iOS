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
chmod +x "${MOCKBIN}/curl" "${MOCKBIN}/launchctl" "${MOCKBIN}/ldid" "${MOCKBIN}/sleep"
PATH="${MOCKBIN}:$PATH"
export PATH

SB_BIN="${SANDBOX}/bin"
SB_LIB="${SANDBOX}/lib"
SB_LAUNCHD="${SANDBOX}/launchd2"
SB_LOG="${SANDBOX}/log"
SB_REL="${SANDBOX}/rel"
SB_WORK="${TMPDIR:-/tmp}/beszel-txn-work-$$"
trap 'rm -rf "$SANDBOX" "${SB_WORK:-}"' EXIT INT TERM HUP

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
	BESZEL_TEST_HEALTH=ok
	BESZEL_TEST_RESOLVE_FAIL=0
	BESZEL_TEST_DOWNLOAD_FAIL=0
	BESZEL_TEST_LDID_FAIL=0
	BESZEL_TEST_UNLOAD_FAIL=0
	BESZEL_TEST_LOAD_FAIL_LABEL=""
	BESZEL_TEST_NO_PID=0
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
if grep -q "4) Update" "$SCRIPT" && grep -q "5) Exit" "$SCRIPT"; then
	pass "menu offers Update and Exit"
else
	fail "menu offers Update and Exit"
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
if grep -q 'INSTALLER_VERSION="0.2.0"' "$SCRIPT"; then
	pass "installer version constant present"
else
	fail "installer version constant present"
fi

printf '\n%d passed, %d failed\n' "$_pass" "$_fail"
[ "$_fail" = "0" ]
