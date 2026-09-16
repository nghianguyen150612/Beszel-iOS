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
if grep -q 'INSTALLER_VERSION="0.1.0"' "$SCRIPT"; then
	pass "installer version constant present"
else
	fail "installer version constant present"
fi

printf '\n%d passed, %d failed\n' "$_pass" "$_fail"
[ "$_fail" = "0" ]
