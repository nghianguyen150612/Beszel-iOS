#!/bin/sh
# check-upstream-drift.sh — report whether the ios port lags upstream Beszel.
#
# Compares the Beszel base version used by this branch (beszel.go Version)
# against the newest stable vX.Y.Z tag on the upstream remote.
# Read-only: never modifies source, merges, bumps versions, or publishes.
#
#   exit 0 — in sync
#   exit 1 — drift detected (ios base older than upstream stable)
#   exit 2 — error (missing files, no network, unexpected state)
#
# Env overrides:
#   UPSTREAM_URL — upstream git URL (default: https://github.com/henrygd/beszel.git)
#   REPO_ROOT    — repository root (default: script's grandparent checkout root)

set -u

UPSTREAM_URL="${UPSTREAM_URL:-https://github.com/henrygd/beszel.git}"
# shellcheck disable=SC1007
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
# shellcheck disable=SC1007
REPO_ROOT="${REPO_ROOT:-$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)}"

BESZEL_GO="$REPO_ROOT/beszel.go"
[ -f "$BESZEL_GO" ] || { echo "ERROR: beszel.go not found at $BESZEL_GO" >&2; exit 2; }

# Parse: Version = "0.19.0" (first such assignment wins).
IOS_BASE="$(sed -n 's/^[[:space:]]*Version[[:space:]]*=[[:space:]]*"\(.*\)".*/\1/p' "$BESZEL_GO" | head -n 1)"
[ -n "$IOS_BASE" ] || { echo "ERROR: could not parse Version from $BESZEL_GO" >&2; exit 2; }

if ! printf '%s\n' "$IOS_BASE" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
	echo "ERROR: ios base '$IOS_BASE' is not a stable X.Y.Z version" >&2
	exit 2
fi

TAGS_OUT="$(mktemp "${TMPDIR:-/tmp}/beszel-upstream-tags.XXXXXX")" || {
	echo "ERROR: could not create temporary tag file" >&2
	exit 2
}
trap 'rm -f "$TAGS_OUT"' EXIT INT TERM
if ! git ls-remote --tags "$UPSTREAM_URL" > "$TAGS_OUT" 2>/dev/null; then
	echo "ERROR: cannot reach upstream at $UPSTREAM_URL (network or URL issue)" >&2
	exit 2
fi

# Stable tags only: refs/tags/vX.Y.Z, with or without an annotated-tag ^{}
# peel line. This also excludes iOS tags and prerelease/suffixed tags.
UPSTREAM_STABLE="$(awk '
		{
			ref = $2
			sub(/^refs\/tags\/v/, "", ref)
			sub(/\^\{\}$/, "", ref)
			if (ref !~ /^[0-9]+\.[0-9]+\.[0-9]+$/) {
				next
			}
			split(ref, parts, ".")
			if (!found || parts[1] + 0 > major ||
				(parts[1] + 0 == major && parts[2] + 0 > minor) ||
				(parts[1] + 0 == major && parts[2] + 0 == minor && parts[3] + 0 > patch)) {
				major = parts[1] + 0
				minor = parts[2] + 0
				patch = parts[3] + 0
				best = ref
				found = 1
			}
		}
		END {
			if (found) {
				print best
			}
		}
	' "$TAGS_OUT")"
[ -n "$UPSTREAM_STABLE" ] || { echo "ERROR: no stable vX.Y.Z tags found on upstream" >&2; exit 2; }

echo "ios Beszel base:        $IOS_BASE"
echo "upstream stable latest: $UPSTREAM_STABLE"

if [ "$IOS_BASE" = "$UPSTREAM_STABLE" ]; then
	echo "status: IN SYNC"
	exit 0
fi

version_is_behind() {
	awk -v ios="$1" -v upstream="$2" '
		BEGIN {
			split(ios, i, ".")
			split(upstream, u, ".")
			if (i[1] + 0 < u[1] + 0 ||
				(i[1] + 0 == u[1] + 0 && i[2] + 0 < u[2] + 0) ||
				(i[1] + 0 == u[1] + 0 && i[2] + 0 == u[2] + 0 && i[3] + 0 < u[3] + 0)) {
				exit 0
			}
			exit 1
		}
	'
}

# Compare versions numerically without relying on GNU sort -V, so the same
# check works on the Ubuntu CI runner and on a macOS maintenance workstation.
if version_is_behind "$IOS_BASE" "$UPSTREAM_STABLE"; then
	echo "status: DRIFT — ios base $IOS_BASE is behind upstream $UPSTREAM_STABLE"
	echo "next: follow docs/upstream-sync.md (audit, integrate, validate, then release)"
	exit 1
else
	echo "status: AHEAD/UNEXPECTED — ios base $IOS_BASE is newer than upstream stable $UPSTREAM_STABLE"
	echo "next: verify beszel.go was not bumped past a real upstream release"
	exit 1
fi
