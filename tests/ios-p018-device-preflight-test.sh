#!/bin/sh
# Host-side guard tests for the P018 device preflight. This does not emulate an
# iPad or provide physical-device evidence.
set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
SCRIPT="${ROOT}/scripts/ios/p018-device-preflight.sh"
TMP_ROOT=${TMPDIR:-/tmp}
TEST_TMP=$(mktemp -d "${TMP_ROOT%/}/p018-preflight.XXXXXX")
GUARD_BIN="${TEST_TMP}/bin"
LAUNCHCTL_LOG="${TEST_TMP}/launchctl.log"
FORBIDDEN_LOG="${TEST_TMP}/forbidden.log"

cleanup() {
	rm -rf "$TEST_TMP"
}
trap cleanup 0 HUP INT TERM

fail() {
	printf 'FAIL: %s\n' "$*" >&2
	exit 1
}

[ -f "$SCRIPT" ] || fail 'P018 preflight script exists'
mkdir -m 700 "$GUARD_BIN"
: > "$LAUNCHCTL_LOG"
: > "$FORBIDDEN_LOG"

cat > "${GUARD_BIN}/launchctl" <<'LAUNCHCTL_EOF'
#!/bin/sh
printf '%s\n' "$*" >> "$P018_LAUNCHCTL_LOG"
if [ "$#" -eq 1 ] && [ "$1" = list ]; then
	printf '%s\n' '123 0 dev.beszel.agent' '456 0 dev.beszel.hub'
	exit 0
fi
exit 97
LAUNCHCTL_EOF
chmod 700 "${GUARD_BIN}/launchctl"

for _tool in rm rmdir mv cp touch mkdir install reboot curl wget fetch nc ftp rsync ssh scp sqlite3 psql mysql kill killall pkill beszel-ios; do
	cat > "${GUARD_BIN}/${_tool}" <<'GUARD_EOF'
#!/bin/sh
printf '%s %s\n' "${0##*/}" "$*" >> "$P018_FORBIDDEN_LOG"
exit 97
GUARD_EOF
	chmod 700 "${GUARD_BIN}/${_tool}"
done

if _output=$(P018_LAUNCHCTL_LOG="$LAUNCHCTL_LOG" \
	P018_FORBIDDEN_LOG="$FORBIDDEN_LOG" \
	PATH="${GUARD_BIN}:${PATH}" sh "$SCRIPT" 2>&1); then
	_preflight_rc=0
else
	_preflight_rc=$?
fi
[ "$_preflight_rc" -eq 0 ] || fail 'preflight exits successfully with read-only host utilities'
printf '%s\n' "$_output" | grep -Fq 'P018 PRE-FLIGHT ONLY' || fail 'preflight identifies its non-acceptance scope'
printf '%s\n' "$_output" | grep -Fq 'No file was written, service restarted, network contacted, or database touched.' || fail 'preflight states its read-only contract'
[ "$(cat "$LAUNCHCTL_LOG")" = 'list' ] || fail 'launchctl is queried only with the read-only list action'
[ ! -s "$FORBIDDEN_LOG" ] || fail 'preflight invokes no destructive, service-control, network, or database-writer command'

if grep -Eq 'launchctl[[:space:]]+(load|unload|start|stop|bootstrap|bootout|remove)([[:space:]]|$)' "$SCRIPT"; then
	fail 'preflight contains no LaunchDaemon mutation command'
fi
if grep -Eq 'beszel-ios[[:space:]]+(update|install|uninstall)([[:space:]]|$)' "$SCRIPT"; then
	fail 'preflight contains no installer mutation command'
fi
if grep -Eq '(^|[[:space:]])(curl|wget|fetch|ftp|rsync)([[:space:]]|$)' "$SCRIPT"; then
	fail 'preflight contains no network download command'
fi
if grep -Eq '^[[:space:]]*(rm|rmdir|mv|cp|touch|mkdir|install|reboot|curl|wget|fetch|nc|ftp|rsync|sqlite3|psql|mysql|kill|killall|pkill)([[:space:]]|$)' "$SCRIPT"; then
	fail 'preflight contains no direct filesystem, reboot, network, process-stop, or database-writer command'
fi

_value_tools=$(sed -n "s/^[[:space:]]*value '[^']*' //p" "$SCRIPT" | awk '{print $1}' | LC_ALL=C sort -u)
_expected_value_tools=$(printf '%s\n' id plutil sysctl uname | LC_ALL=C sort -u)
[ "$_value_tools" = "$_expected_value_tools" ] || fail 'preflight invokes only the approved read-only identity utilities through value()'

: > "$LAUNCHCTL_LOG"
if _usage=$(P018_LAUNCHCTL_LOG="$LAUNCHCTL_LOG" \
	P018_FORBIDDEN_LOG="$FORBIDDEN_LOG" \
	PATH="${GUARD_BIN}:${PATH}" sh "$SCRIPT" unexpected 2>&1); then
	_usage_rc=0
else
	_usage_rc=$?
fi
[ "$_usage_rc" -eq 2 ] || fail 'preflight rejects unexpected arguments'
printf '%s\n' "$_usage" | grep -Fq 'Usage: sh p018-device-preflight.sh (no arguments)' || fail 'preflight shows its no-argument usage'
[ ! -s "$LAUNCHCTL_LOG" ] || fail 'invalid invocation exits before querying services'
[ ! -s "$FORBIDDEN_LOG" ] || fail 'invalid invocation performs no forbidden operation'

printf '%s\n' 'P018 read-only device preflight safety checks passed.'
