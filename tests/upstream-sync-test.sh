#!/bin/sh
# Deterministic tests for the read-only upstream drift check.
#
# The fixture remote is local, so these tests do not require network access and
# do not depend on the current upstream tag set.

set -u

# shellcheck disable=SC1007
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$SCRIPT_DIR/.."
DRIFT_SCRIPT="$REPO_ROOT/.github/scripts/check-upstream-drift.sh"
TEST_DIR="$(mktemp -d "${TMPDIR:-/tmp}/beszel-upstream-sync.XXXXXX")" || exit 1

# shellcheck disable=SC2329
cleanup() {
	rm -rf "$TEST_DIR"
}
trap cleanup EXIT HUP INT TERM

_passed=0
_failed=0

pass() {
	printf 'PASS: %s\n' "$1"
	_passed=$((_passed + 1))
}

fail() {
	printf 'FAIL: %s\n' "$1" >&2
	_failed=$((_failed + 1))
}

assert_eq() {
	_label="$1"
	_expected="$2"
	_actual="$3"
	if [ "$_expected" = "$_actual" ]; then
		pass "$_label"
	else
		fail "$_label — expected '$_expected', got '$_actual'"
	fi
}

if [ ! -f "$DRIFT_SCRIPT" ]; then
	fail "drift script exists"
	exit 1
fi

REMOTE="$TEST_DIR/upstream.git"
SEED="$TEST_DIR/seed"
FIXTURE="$TEST_DIR/fixture"
mkdir -p "$FIXTURE"
printf 'package beszel\n\nVersion = "0.19.0"\n' > "$FIXTURE/beszel.go"

if ! git init -q --bare "$REMOTE" ||
	! git init -q "$SEED" ||
	! git -C "$SEED" config user.name "upstream-sync-test" ||
	! git -C "$SEED" config user.email "upstream-sync-test@example.invalid"; then
	fail "create local git fixture"
	exit 1
fi

printf 'package beszel\n\nVersion = "0.19.0"\n' > "$SEED/beszel.go"
if ! git -C "$SEED" add beszel.go ||
	! git -C "$SEED" commit -q -m "fixture"; then
	fail "commit local git fixture"
	exit 1
fi

# Exercise both lightweight and annotated stable tags, plus tags the checker
# must ignore (iOS and prerelease suffixes).
if ! git -C "$SEED" tag v0.18.0 ||
	! git -C "$SEED" tag -a v0.19.0 -m "stable fixture" ||
	! git -C "$SEED" tag v0.19.0-ios.1 ||
	! git -C "$SEED" tag v0.19.0-rc.1 ||
	! git -C "$SEED" push -q "$REMOTE" --tags; then
	fail "publish local git fixture tags"
	exit 1
fi

run_check() {
	_expected_rc="$1"
	_expected_status="$2"
	_label="$3"
	_output="$TEST_DIR/${_label}.out"
	if (
		REPO_ROOT="$FIXTURE" UPSTREAM_URL="$REMOTE" sh "$DRIFT_SCRIPT" > "$_output" 2>&1
	); then
		_actual_rc=0
	else
		_actual_rc=$?
	fi
	assert_eq "$_label exit status" "$_expected_rc" "$_actual_rc"
	if grep -qF -- "$_expected_status" "$_output"; then
		pass "$_label reports expected status"
	else
		fail "$_label reports expected status — expected '$_expected_status'"
	fi
}

run_check 0 "status: IN SYNC" "in-sync"
if grep -qF -- "ios Beszel base:        0.19.0" "$TEST_DIR/in-sync.out" &&
	grep -qF -- "upstream stable latest: 0.19.0" "$TEST_DIR/in-sync.out"; then
	pass "in-sync output names both versions"
else
	fail "in-sync output names both versions"
fi

printf 'package beszel\n\nVersion = "0.18.0"\n' > "$FIXTURE/beszel.go"
run_check 1 "status: DRIFT" "behind"
if grep -qF -- "ios base 0.18.0 is behind upstream 0.19.0" "$TEST_DIR/behind.out"; then
	pass "behind output names both versions"
else
	fail "behind output names both versions"
fi

printf 'package beszel\n\nVersion = "0.20.0"\n' > "$FIXTURE/beszel.go"
run_check 1 "status: AHEAD/UNEXPECTED" "ahead"

printf 'package beszel\n\nVersion = "0.19"\n' > "$FIXTURE/beszel.go"
run_check 2 "not a stable X.Y.Z version" "malformed"

printf '\n%d passed, %d failed\n' "$_passed" "$_failed"
if [ "$_failed" -gt 0 ]; then
	exit 1
fi
exit 0
