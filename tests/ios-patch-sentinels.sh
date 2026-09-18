#!/bin/sh
# ios-patch-sentinels.sh — fail loudly if future upstream merges drop iOS behavior.
#
# Each check corresponds to a critical iOS-specific patch. If an upstream
# merge removes or breaks it, this script exits non-zero with a clear
# message naming the affected area. No upstream source is modified.
#
# Run: sh tests/ios-patch-sentinels.sh
# Exit 0 = all sentinels intact. Non-zero = review required.

set -u

# shellcheck disable=SC1007
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
# shellcheck disable=SC1007
REPO_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"
_pass=0
_fail=0

pass() { printf 'PASS: %s\n' "$1"; _pass=$((_pass + 1)); }
fail() { printf 'FAIL: %s\n' "$1" >&2; _fail=$((_fail + 1)); }

# _require_file <path> <description> — file must exist.
_require_file() {
	if [ -f "$REPO_ROOT/$1" ]; then
		pass "$2"
	else
		fail "$2 — missing: $1"
	fi
}

# _require_in_file <path> <literal-string> <description> — literal string must appear.
_require_in_file() {
	if grep -qF -- "$2" "$REPO_ROOT/$1" 2>/dev/null; then
		pass "$2"
	else
		fail "$2 — literal not found in $1"
	fi
}

# _require_grep <path> <regex> <description> — regex must match.
_require_grep() {
	if grep -qE -- "$2" "$REPO_ROOT/$1" 2>/dev/null; then
		pass "$3"
	else
		fail "$3 — regex '$2' not found in $1"
	fi
}

# _require_build_tag <path> <tag> <description>
_require_build_tag() {
	if grep -qE -- "$2" "$REPO_ROOT/$1" 2>/dev/null; then
		pass "$3"
	else
		fail "$3 — build tag '$2' not found in $1"
	fi
}

printf '== iOS patch regression sentinels ==\n'

# ---- A. Apple A7 Go runtime patch ---------------------------------------------------

_require_file ".github/scripts/patch-go-ios-arm64-runtime.py" \
	"A.1 runtime patch script exists"

_require_in_file ".github/scripts/patch-go-ios-arm64-runtime.py" \
	"procyieldAsm" \
	"A.2 runtime patch targets procyieldAsm"

_require_in_file ".github/scripts/patch-go-ios-arm64-runtime.py" \
	"CNTVCT_EL0" \
	"A.3 runtime patch guards on CNTVCT_EL0"

_require_in_file ".github/scripts/patch-go-ios-arm64-runtime.py" \
	"refusing to patch unknown Go runtime" \
	"A.4 runtime patch refuses loudly on unknown shape"

_require_in_file ".github/scripts/patch-go-ios-arm64-runtime.py" \
	"expected_shape" \
	"A.5 runtime patch checks the known procyield source shape"

_require_in_file ".github/scripts/patch-go-ios-arm64-runtime.py" \
	"CNTFRQ_EL0" \
	"A.6 runtime patch checks the counter-frequency read"

# CI applies the patch before Go builds (verify workflow ordering).
_require_in_file ".github/workflows/ios-build.yml" \
	"patch-go-ios-arm64-runtime.py" \
	"A.7 CI applies runtime patch"

# Confirm the patch step runs before any go build in the workflow.
_patch_line=$(grep -n "patch-go-ios-arm64-runtime.py" "$REPO_ROOT/.github/workflows/ios-build.yml" | head -1 | cut -d: -f1)
_go_build_line=$(grep -n "go build" "$REPO_ROOT/.github/workflows/ios-build.yml" | head -1 | cut -d: -f1)
if [ -n "$_patch_line" ] && [ -n "$_go_build_line" ] && [ "$_patch_line" -lt "$_go_build_line" ]; then
	pass "A.8 CI applies runtime patch before go build"
else
	fail "A.8 CI patch step must precede go build (patch line $_patch_line, build line $_go_build_line)"
fi

# ---- B. Architecture / deployment target ---------------------------------------------------

_require_in_file ".github/workflows/ios-build.yml" \
	"GOOS=ios" \
	"B.1 workflow builds GOOS=ios"

_require_in_file ".github/workflows/ios-build.yml" \
	"GOARCH=arm64" \
	"B.2 workflow builds GOARCH=arm64"

_require_in_file ".github/workflows/ios-build.yml" \
	"**/*.go" \
	"B.3 workflow covers Go source changes in its push paths"

_require_in_file ".github/workflows/ios-build.yml" \
	"mios-version-min=12.0" \
	"B.4 workflow targets iOS 12.0 minimum"

_require_grep ".github/workflows/ios-build.yml" \
	"file .*Mach-O" \
	"B.5 workflow validates Mach-O output"

_require_in_file ".github/workflows/ios-build.yml" \
	"arm64" \
	"B.6 workflow validates arm64"

_require_in_file ".github/workflows/ios-build.yml" \
	"LC_BUILD_VERSION" \
	"B.7 workflow checks iOS version load commands"

_require_grep ".github/workflows/ios-build.yml" \
	"12\.0" \
	"B.8 workflow validates 12.0 deployment metadata"

# ---- C. Agent / Hub outputs ----------------------------------------------------------------

_require_in_file ".github/workflows/ios-build.yml" \
	"beszel-agent-ios-arm64" \
	"C.1 workflow produces Agent binary"

_require_in_file ".github/workflows/ios-build.yml" \
	"beszel-hub-ios-arm64" \
	"C.2 workflow produces Hub binary"

_require_in_file ".github/workflows/ios-build.yml" \
	"SHA256SUMS" \
	"C.3 workflow produces SHA256SUMS"

_require_in_file ".github/workflows/ios-build.yml" \
	"./internal/cmd/agent" \
	"C.4 Agent build command present"

_require_in_file ".github/workflows/ios-build.yml" \
	"./internal/cmd/hub" \
	"C.5 Hub build command present"

# ---- D. Battery telemetry source ----------------------------------------------------

_require_file "agent/battery/battery_ios.go" \
	"D.1 iOS battery source exists"

_require_build_tag "agent/battery/battery_ios.go" "ios" \
	"D.2 iOS battery has //go:build ios"

_require_in_file "agent/battery/battery_ios.go" \
	"AppleARMPMUCharger" \
	"D.3 iOS battery uses AppleARMPMUCharger"

_require_in_file "agent/battery/battery_ios.go" \
	"ioreg" \
	"D.4 iOS battery uses ioreg"

_require_file "agent/battery/battery_darwin.go" \
	"D.5 Darwin battery source exists"

_require_build_tag "agent/battery/battery_darwin.go" "darwin && !ios" \
	"D.6 Darwin battery gated darwin && !ios"

# ---- E. Installer release tag expectations --------------------------------------------

_require_in_file "install.sh" \
	"-ios\\." \
	"E.1 install.sh accepts iOS tag scheme"

_require_in_file "install.sh" \
	"valid_release_tag" \
	"E.2 install.sh validates release tags"

_require_grep ".github/workflows/ios-build.yml" \
	"v\*-ios\.\*" \
	"E.3 iOS build triggers on v*-ios.* tags"

_require_grep ".github/workflows/release.yml" \
	'"!v\*-ios\.\*"' \
	"E.4 release.yml excludes iOS tags"

_require_grep ".github/workflows/docker-images.yml" \
	'"!v\*-ios\.\*"' \
	"E.5 docker-images.yml excludes iOS tags"

# ---- F. Frontend embed/build dependency ------------------------------------------------

_require_file "internal/site/embed.go" \
	"F.1 frontend embed source exists"

_require_in_file "internal/site/embed.go" \
	"//go:embed all:dist" \
	"F.2 frontend embeds dist/"

# dist/ must be gitignored so fresh checkouts cannot have it pre-filled.
if [ -f "$REPO_ROOT/.gitignore" ] && grep -qxF "dist" "$REPO_ROOT/.gitignore"; then
	pass "F.3 dist/ is gitignored"
else
	fail "F.3 dist/ must be in .gitignore (cannot fake frontend build output)"
fi

# Workflow must build frontend BEFORE Go builds (no empty dist directory).
_site_build_line=$(grep -n "bun run build" "$REPO_ROOT/.github/workflows/ios-build.yml" | head -1 | cut -d: -f1)
_first_go_build_line=$(grep -n "go build" "$REPO_ROOT/.github/workflows/ios-build.yml" | head -1 | cut -d: -f1)
if [ -n "$_site_build_line" ] && [ -n "$_first_go_build_line" ] && [ "$_site_build_line" -lt "$_first_go_build_line" ]; then
	pass "F.4 CI builds frontend before Go build"
else
	fail "F.4 CI must build frontend before Go build (site line $_site_build_line, go build line $_first_go_build_line)"
fi

# ---- G. System metadata hook ---------------------------------------------------------

_require_grep "agent/system.go" \
	'runtime\.GOOS[^=]*!=.*"ios"' \
	"G.1 system.go guards Darwin CPU probe on iOS"

_require_in_file "agent/system.go" \
	"adjustPlatformSystemDetails" \
	"G.2 system.go calls adjustPlatformSystemDetails"

_require_file "agent/system_platform_ios.go" \
	"G.3 iOS platform file exists"

_require_build_tag "agent/system_platform_ios.go" "ios" \
	"G.4 iOS platform file has //go:build ios"

_require_file "agent/system_platform_other.go" \
	"G.5 non-iOS platform file exists"

_require_build_tag "agent/system_platform_other.go" "!ios" \
	"G.6 non-iOS platform file has //go:build !ios"

# ---- Summary --------------------------------------------------------------------------

printf '\n%d passed, %d failed\n' "$_pass" "$_fail"
if [ "$_fail" -gt 0 ]; then
	printf '\niOS patch regression checks FAILED. Review upstream merge impact.\n'
	exit 1
fi
printf 'All iOS patch regression checks passed.\n'
exit 0
