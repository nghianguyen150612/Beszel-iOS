#!/bin/sh
# Beszel iOS Installer — unofficial community port of Beszel for jailbroken iOS.
#
# Primary invocation (no clone, no Go toolchain, no manual downloads needed):
#
#   curl -fsSL https://raw.githubusercontent.com/nghianguyen150612/beszel-ios/ios/install.sh | sudo sh
#
# POSIX /bin/sh only: no bashisms, no zsh-isms. Interactive input is read
# from /dev/tty so the menu works when stdin is a curl pipe.
#
# Scope: fresh installs of Agent and/or Hub, safe transactional updates
# with signed-binary staging, binary backup and automatic rollback,
# non-destructive diagnostics, conservative repair, safe reconfiguration,
# safe application uninstall (data preserved by default) and optional
# explicit data purge. Nothing else is intentionally missing from the
# installer lifecycle.
#
# Data policy: /var/lib/beszel-agent and especially /var/lib/beszel-hub
# (Hub database, accounts, configuration) are USER DATA. Ordinary uninstall
# removes only application/service artifacts and always preserves data;
# deleting data requires a separate explicit typed confirmation.
#
# Version model: installed binaries are ldid-signed on device, so the hash of
# an installed binary differs from its unsigned release asset. Update
# detection therefore uses the installer-managed state file
# (/var/lib/beszel-ios/install-state), never a hash comparison of the
# installed binary against SHA256SUMS. SHA256SUMS only validates freshly
# downloaded release assets before installation.
#
# Key handling: the Agent plist holds the Hub public key. Diagnostics and
# summaries report only whether a key is configured; the value is never
# printed, logged, or stored in install-state.

set -eu
umask 022

INSTALLER_VERSION="0.4.0"
USER_AGENT="beszel-ios-installer/${INSTALLER_VERSION}"

RELEASE_LATEST_PAGE="https://github.com/nghianguyen150612/beszel-ios/releases/latest"
RELEASE_DOWNLOAD_ROOT="https://github.com/nghianguyen150612/beszel-ios/releases/download"
RELEASE_BASE="https://github.com/nghianguyen150612/beszel-ios/releases/latest/download"

AGENT_ASSET="beszel-agent-ios-arm64"
HUB_ASSET="beszel-hub-ios-arm64"
SUMS_ASSET="SHA256SUMS"

# Install locations. The BESZEL_* overrides exist so the bundled test suite
# can redirect writes into a sandbox; normal production runs never set them.
BIN_DIR="${BESZEL_BIN_DIR:-/usr/local/bin}"
LIB_DIR="${BESZEL_LIB_DIR:-/var/lib}"
LAUNCHD_DIR="${BESZEL_LAUNCHD_DIR:-/Library/LaunchDaemons}"
LOG_DIR="${BESZEL_LOG_DIR:-/var/log}"

AGENT_BIN="beszel-agent"
HUB_BIN="beszel-hub"
AGENT_LABEL="dev.beszel.agent"
HUB_LABEL="dev.beszel.hub"

# Installer-managed release state. STATE_DIR is derived from LIB_DIR at use
# time (via state_path) so sandbox overrides keep working. The state file
# records which RELEASE was installed, never the signed binary hash.
STATE_SUBDIR="beszel-ios"
STATE_FILE_NAME="install-state"
STATE_VERSION="1"

# Pinned immutable release for the current run. Resolved once from the Latest
# page; every asset for the transaction (SHA256SUMS + binaries) then comes
# from the same tag, so a release cannot change mid-transaction.
LATEST_TAG=""
PINNED_RELEASE_BASE=""
SUMS_FETCHED=0

# Update transaction flags for the INT/TERM/HUP rollback traps.
_UPDATE_ACTIVE=""
_UPDATE_NEED_ROLLBACK=0
_UPDATE_TRAP_BUSY=0

DEFAULT_AGENT_PORT="45876"
DEFAULT_HUB_PORT="8090"
LAUNCHD_PATH_VALUE="/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

WORK_DIR=""

# ---------------------------------------------------------------- logging ---

say_info() { printf '[INFO] %s\n' "$*"; }
say_ok() { printf '[OK] %s\n' "$*"; }
say_warn() { printf '[WARN] %s\n' "$*" >&2; }
say_err() { printf '[ERROR] %s\n' "$*" >&2; }

die() {
	say_err "$*"
	exit 1
}

# ------------------------------------------------------- interactive input ---

# read_tty <prompt> <varname> [default]
# Prints the prompt to /dev/tty and reads one line from /dev/tty into the
# named variable (which must already exist, initialised to ""). Works when
# stdin is a pipe (curl ... | sh). No eval: read assigns via the expanded name.
read_tty() {
	_rt_prompt="$1"
	_rt_var="$2"
	_rt_default="${3:-}"
	if [ ! -r /dev/tty ] || [ ! -w /dev/tty ]; then
		say_err "Interactive input needs a terminal but /dev/tty is unavailable."
		return 1
	fi
	if [ -n "$_rt_default" ]; then
		printf '%s [%s]: ' "$_rt_prompt" "$_rt_default" > /dev/tty
	else
		printf '%s: ' "$_rt_prompt" > /dev/tty
	fi
	# Intentional indirection: read assigns to the variable NAMED by $_rt_var.
	# shellcheck disable=SC2229
	IFS= read -r "$_rt_var" < /dev/tty || return 1
	return 0
}

# ask_tty <prompt> <varname> [default]
# Wrapper around read_tty that fails clearly when /dev/tty cannot be used.
# Callers apply the default on empty input themselves.
ask_tty() {
	_aq_prompt="$1"
	_aq_var="$2"
	_aq_default="${3:-}"
	read_tty "$_aq_prompt" "$_aq_var" "$_aq_default" || {
		say_err "Cannot read input (no terminal). Aborting instead of hanging."
		return 1
	}
	return 0
}

wait_for_enter() {
	_wf_note="$1"
	if [ ! -r /dev/tty ] || [ ! -w /dev/tty ]; then
		die "Cannot pause for input: /dev/tty is unavailable."
	fi
	printf '%s\n(Press Enter to continue) ' "$_wf_note" > /dev/tty
	_wf_junk=""
	IFS= read -r _wf_junk < /dev/tty || true
	return 0
}

# ------------------------------------------------------------ pure helpers ---

# valid_port <value> — numeric TCP port 1-65535.
valid_port() {
	case "${1:-}" in
		'' | *[!0-9]*) return 1 ;;
	esac
	[ "$1" -ge 1 ] && [ "$1" -le 65535 ]
}

# valid_ssh_key <value> — non-empty single line whose first two fields look
# like an SSH public key. The trailing comment is unrestricted on purpose.
valid_ssh_key() {
	_vk_key="${1:-}"
	[ -n "$_vk_key" ] || return 1
	case "$_vk_key" in
		*'
'*) return 1 ;;
	esac
	set -f
	# shellcheck disable=SC2086
	set -- $_vk_key
	set +f
	[ "$#" -ge 2 ] || return 1
	case "$1" in
		ssh-ed25519 | ssh-rsa | ssh-dss | \
			ecdsa-sha2-nistp256 | ecdsa-sha2-nistp384 | ecdsa-sha2-nistp521 | \
			sk-ssh-ed25519@openssh.com | sk-ecdsa-sha2-nistp256@openssh.com) ;;
		*) return 1 ;;
	esac
	case "$2" in
		'' | *[!A-Za-z0-9+/=]*) return 1 ;;
	esac
	return 0
}

# xml_escape <value> — escape & < > " ' for plist string elements.
xml_escape() {
	printf '%s' "$1" | sed -e 's/&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g' -e 's/"/\&quot;/g' -e "s/'/\&apos;/g"
}

# ------------------------------------------------- release tag resolution ---

# tag_from_latest_url <effective-url> — print the trailing tag segment of a
# GitHub /releases/latest redirect target. Rejects anything outside a strict
# allowlist so no shell metacharacters or path traversal can enter a URL.
tag_from_latest_url() {
	_tu_work="${1:-}"
	[ -n "$_tu_work" ] || return 1
	while :; do
		case "$_tu_work" in
			*/) _tu_work=${_tu_work%/} ;;
			*) break ;;
		esac
	done
	_tu_tag=${_tu_work##*/}
	[ -n "$_tu_tag" ] || return 1
	case "$_tu_tag" in
		*[!A-Za-z0-9._-]* | "") return 1 ;;
	esac
	printf '%s' "$_tu_tag"
	return 0
}

# valid_release_tag <tag> — Beszel iOS release scheme v<semver>-ios.<rev>.
valid_release_tag() {
	_vt_tag="${1:-}"
	[ -n "$_vt_tag" ] || return 1
	printf '%s' "$_vt_tag" | grep -q -E '^v[0-9]+\.[0-9]+\.[0-9]+-ios\.[1-9][0-9]*$' || return 1
	return 0
}

# pinned_base_for_tag <tag> — immutable asset base for one release tag.
pinned_base_for_tag() {
	_pb_tag="${1:-}"
	valid_release_tag "$_pb_tag" || return 1
	printf '%s/%s' "$RELEASE_DOWNLOAD_ROOT" "$_pb_tag"
	return 0
}

# resolve_latest_tag — follow the Latest page redirect and print its tag.
resolve_latest_tag() {
	_rl_eff=""
	_rl_eff=$(curl -fsSL -A "$USER_AGENT" --retry 3 --retry-delay 2 -o /dev/null -w '%{url_effective}' "$RELEASE_LATEST_PAGE") || return 1
	[ -n "$_rl_eff" ] || return 1
	_rl_tag=""
	_rl_tag=$(tag_from_latest_url "$_rl_eff") || return 1
	valid_release_tag "$_rl_tag" || return 1
	printf '%s' "$_rl_tag"
	return 0
}

# ensure_pinned_release — resolve Latest once per run and pin all downloads.
ensure_pinned_release() {
	if [ -n "$LATEST_TAG" ] && [ -n "$PINNED_RELEASE_BASE" ]; then
		return 0
	fi
	_ep_tag=""
	_ep_tag=$(resolve_latest_tag) || die "Cannot resolve the Latest Beszel iOS release tag; aborting without changing anything."
	LATEST_TAG="$_ep_tag"
	PINNED_RELEASE_BASE=$(pinned_base_for_tag "$LATEST_TAG") || die "Resolved release tag '${LATEST_TAG}' failed validation; aborting."
	say_info "Latest Beszel iOS release: ${LATEST_TAG}"
	return 0
}

# current_release_base — pinned immutable base when resolved, else the
# legacy Latest download base (used only by tests that redirect downloads).
current_release_base() {
	if [ -n "${PINNED_RELEASE_BASE:-}" ]; then
		printf '%s' "$PINNED_RELEASE_BASE"
	else
		printf '%s' "$RELEASE_BASE"
	fi
	return 0
}

# sums_hash_for <sums-file> <filename> — print the checksum when the file
# contains exactly one well-formed entry for that name.
sums_hash_for() {
	_sh_file="$1"
	_sh_name="$2"
	[ -f "$_sh_file" ] || return 1
	_sh_count=$(grep -E -c "^[0-9a-fA-F]{64}[[:space:]][ *]${_sh_name}\$" "$_sh_file" || true)
	[ "$_sh_count" = "1" ] || return 1
	_sh_line=$(grep -E "^[0-9a-fA-F]{64}[[:space:]][ *]${_sh_name}\$" "$_sh_file" | head -n 1)
	_sh_hash=${_sh_line%% *}
	printf '%s' "$_sh_hash" | tr 'A-F' 'a-f'
	return 0
}

# validate_sums_file <sums-file> — the release contract: exactly two entries,
# one per binary, both well-formed.
validate_sums_file() {
	_vs_file="$1"
	[ -f "$_vs_file" ] && [ -s "$_vs_file" ] || return 1
	_vs_total=$(grep -c -E '.' "$_vs_file" || true)
	[ "$_vs_total" = "2" ] || return 1
	sums_hash_for "$_vs_file" "$AGENT_ASSET" > /dev/null || return 1
	sums_hash_for "$_vs_file" "$HUB_ASSET" > /dev/null || return 1
	return 0
}

# compute_sha256 <file> — lowercase hex digest using the first available tool.
compute_sha256() {
	_cs_file="$1"
	_cs_out=""
	if command -v sha256sum > /dev/null 2>&1; then
		_cs_out=$(sha256sum "$_cs_file") || return 1
		_cs_out=${_cs_out%% *}
	elif command -v shasum > /dev/null 2>&1; then
		_cs_out=$(shasum -a 256 "$_cs_file") || return 1
		_cs_out=${_cs_out%% *}
	elif command -v openssl > /dev/null 2>&1; then
		_cs_out=$(openssl dgst -sha256 "$_cs_file") || return 1
		_cs_out=${_cs_out##* }
	else
		return 1
	fi
	printf '%s' "$_cs_out" | tr 'A-F' 'a-f'
	return 0
}

# verify_file <path> <expected-hex> — non-empty file whose digest matches.
verify_file() {
	_vf_path="$1"
	_vf_expected=$(printf '%s' "$2" | tr 'A-F' 'a-f')
	[ -f "$_vf_path" ] && [ -s "$_vf_path" ] || return 1
	_vf_actual=$(compute_sha256 "$_vf_path") || return 1
	[ "$_vf_actual" = "$_vf_expected" ]
}

agent_installed() {
	[ -e "${BIN_DIR}/${AGENT_BIN}" ] || [ -e "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" ]
}

hub_installed() {
	[ -e "${BIN_DIR}/${HUB_BIN}" ] || [ -e "${LAUNCHD_DIR}/${HUB_LABEL}.plist" ]
}

# ------------------------------------------------------- install state ---

# Security policy for /var/lib/beszel-ios/install-state:
# - parsed explicitly key by key with grep/head/parameter expansion only;
# - never dotted, sourced or passed through a shell, so hostile content such
#   as AGENT_RELEASE=$(touch /tmp/x) can never execute;
# - unknown keys are ignored (only the five known keys are ever read);
# - values are validated before use (release tags, 64-hex digests);
# - no secrets, keys, addresses or identifiers are stored there.
state_path() {
	printf '%s/%s/%s' "$LIB_DIR" "$STATE_SUBDIR" "$STATE_FILE_NAME"
	return 0
}

# state_raw_value <KEY> — print the raw value of the first "KEY=" line.
state_raw_value() {
	_sr_key="$1"
	_sr_file=$(state_path)
	[ -f "$_sr_file" ] || return 1
	_sr_line=$(grep -E "^${_sr_key}=" "$_sr_file" 2> /dev/null | head -n 1 || true)
	[ -n "$_sr_line" ] || return 1
	_sr_val=${_sr_line#*=}
	printf '%s' "$_sr_val"
	return 0
}

state_agent_release() {
	_sa_rel=""
	_sa_rel=$(state_raw_value "AGENT_RELEASE") || return 1
	[ -n "$_sa_rel" ] || return 1
	valid_release_tag "$_sa_rel" || return 1
	printf '%s' "$_sa_rel"
	return 0
}

state_hub_release() {
	_sh_rel=""
	_sh_rel=$(state_raw_value "HUB_RELEASE") || return 1
	[ -n "$_sh_rel" ] || return 1
	valid_release_tag "$_sh_rel" || return 1
	printf '%s' "$_sh_rel"
	return 0
}

state_agent_sha() {
	_sas_sha=""
	_sas_sha=$(state_raw_value "AGENT_ASSET_SHA256") || return 1
	printf '%s' "$_sas_sha" | grep -q -E '^[0-9a-fA-F]{64}$' || return 1
	printf '%s' "$_sas_sha" | tr 'A-F' 'a-f'
	return 0
}

state_hub_sha() {
	_shs_sha=""
	_shs_sha=$(state_raw_value "HUB_ASSET_SHA256") || return 1
	printf '%s' "$_shs_sha" | grep -q -E '^[0-9a-fA-F]{64}$' || return 1
	printf '%s' "$_shs_sha" | tr 'A-F' 'a-f'
	return 0
}

# state_write_all <agent-rel> <agent-sha> <hub-rel> <hub-sha> — atomically
# writes the whole state file. Each value may be empty (unknown); non-empty
# values must validate. Shared by record and clear paths.
state_write_all() {
	_sa_agent_rel="$1"
	_sa_agent_sha="$2"
	_sa_hub_rel="$3"
	_sa_hub_sha="$4"
	if [ -n "$_sa_agent_rel" ]; then
		valid_release_tag "$_sa_agent_rel" || return 1
	fi
	if [ -n "$_sa_agent_sha" ]; then
		printf '%s' "$_sa_agent_sha" | grep -q -E '^[0-9a-fA-F]{64}$' || return 1
		_sa_agent_sha=$(printf '%s' "$_sa_agent_sha" | tr 'A-F' 'a-f')
	fi
	if [ -n "$_sa_hub_rel" ]; then
		valid_release_tag "$_sa_hub_rel" || return 1
	fi
	if [ -n "$_sa_hub_sha" ]; then
		printf '%s' "$_sa_hub_sha" | grep -q -E '^[0-9a-fA-F]{64}$' || return 1
		_sa_hub_sha=$(printf '%s' "$_sa_hub_sha" | tr 'A-F' 'a-f')
	fi
	_sa_dir="${LIB_DIR}/${STATE_SUBDIR}"
	mkdir -p "$_sa_dir" || return 1
	fix_path_owner "$_sa_dir" || true
	chmod 755 "$_sa_dir" || return 1
	_sa_tmp="${_sa_dir}/${STATE_FILE_NAME}.new.$$"
	{
		printf 'STATE_VERSION=%s\n' "$STATE_VERSION"
		printf 'AGENT_RELEASE=%s\n' "$_sa_agent_rel"
		printf 'AGENT_ASSET_SHA256=%s\n' "$_sa_agent_sha"
		printf 'HUB_RELEASE=%s\n' "$_sa_hub_rel"
		printf 'HUB_ASSET_SHA256=%s\n' "$_sa_hub_sha"
	} > "$_sa_tmp" || {
		rm -f "$_sa_tmp"
		return 1
	}
	[ -s "$_sa_tmp" ] || {
		rm -f "$_sa_tmp"
		return 1
	}
	fix_path_owner "$_sa_tmp" || true
	chmod 644 "$_sa_tmp" || {
		rm -f "$_sa_tmp"
		return 1
	}
	mv -f "$_sa_tmp" "$(state_path)" || {
		rm -f "$_sa_tmp"
		return 1
	}
	fix_path_owner "$(state_path)" || true
	return 0
}

# state_write_component <agent|hub> <release-tag> <asset-sha256>
# Atomically records one component, preserving the other component's values.
state_write_component() {
	_sw_comp="$1"
	_sw_rel="$2"
	_sw_sha="$3"
	case "$_sw_comp" in
		agent | hub) ;;
		*) return 1 ;;
	esac
	valid_release_tag "$_sw_rel" || return 1
	printf '%s' "$_sw_sha" | grep -q -E '^[0-9a-fA-F]{64}$' || return 1
	_sw_agent_rel=""
	_sw_agent_sha=""
	_sw_hub_rel=""
	_sw_hub_sha=""
	_sw_agent_rel=$(state_agent_release 2> /dev/null || true)
	_sw_agent_sha=$(state_agent_sha 2> /dev/null || true)
	_sw_hub_rel=$(state_hub_release 2> /dev/null || true)
	_sw_hub_sha=$(state_hub_sha 2> /dev/null || true)
	case "$_sw_comp" in
		agent)
			_sw_agent_rel="$_sw_rel"
			_sw_agent_sha="$_sw_sha"
			;;
		hub)
			_sw_hub_rel="$_sw_rel"
			_sw_hub_sha="$_sw_sha"
			;;
	esac
	if state_write_all "$_sw_agent_rel" "$_sw_agent_sha" "$_sw_hub_rel" "$_sw_hub_sha"; then
		return 0
	fi
	return 1
}

# state_clear_component <agent|hub> — clears only that component's release
# fields, preserving the other component. When no component release
# information remains, the state file itself is removed (exact path) and the
# state directory is removed with rmdir, but only if empty.
state_clear_component() {
	_sc_comp="$1"
	case "$_sc_comp" in
		agent | hub) ;;
		*) return 1 ;;
	esac
	_sc_agent_rel=""
	_sc_agent_sha=""
	_sc_hub_rel=""
	_sc_hub_sha=""
	_sc_agent_rel=$(state_agent_release 2> /dev/null || true)
	_sc_agent_sha=$(state_agent_sha 2> /dev/null || true)
	_sc_hub_rel=$(state_hub_release 2> /dev/null || true)
	_sc_hub_sha=$(state_hub_sha 2> /dev/null || true)
	case "$_sc_comp" in
		agent)
			_sc_agent_rel=""
			_sc_agent_sha=""
			;;
		hub)
			_sc_hub_rel=""
			_sc_hub_sha=""
			;;
	esac
	if [ -z "$_sc_agent_rel" ] && [ -z "$_sc_agent_sha" ] && [ -z "$_sc_hub_rel" ] && [ -z "$_sc_hub_sha" ]; then
		rm -f "$(state_path)" || return 1
		rmdir "${LIB_DIR}/${STATE_SUBDIR}" 2> /dev/null || true
		return 0
	fi
	if state_write_all "$_sc_agent_rel" "$_sc_agent_sha" "$_sc_hub_rel" "$_sc_hub_sha"; then
		return 0
	fi
	return 1
}

# fix_path_owner <path> — best-effort root:wheel for production; tolerates
# sandboxes and Linux hosts where wheel/root chown cannot succeed.
fix_path_owner() {
	_fo_file="$1"
	chown root:wheel "$_fo_file" 2> /dev/null && return 0
	chown root "$_fo_file" 2> /dev/null && return 0
	if [ "$(id -u)" = "0" ]; then
		return 1
	fi
	return 0
}

# component_status <agent|hub> — prints: missing | incomplete | complete.
component_status() {
	_cs_comp="$1"
	_cs_bin=""
	_cs_plist=""
	case "$_cs_comp" in
		agent)
			_cs_bin="${BIN_DIR}/${AGENT_BIN}"
			_cs_plist="${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
			;;
		hub)
			_cs_bin="${BIN_DIR}/${HUB_BIN}"
			_cs_plist="${LAUNCHD_DIR}/${HUB_LABEL}.plist"
			;;
		*)
			return 1
			;;
	esac
	_cs_have_bin=0
	_cs_have_plist=0
	[ -e "$_cs_bin" ] && _cs_have_bin=1
	[ -e "$_cs_plist" ] && _cs_have_plist=1
	if [ "$_cs_have_bin" = "0" ] && [ "$_cs_have_plist" = "0" ]; then
		printf 'missing'
	elif [ "$_cs_have_bin" = "1" ] && [ "$_cs_have_plist" = "1" ]; then
		printf 'complete'
	else
		printf 'incomplete'
	fi
	return 0
}

# component_state <agent|hub> — prints one of:
#   ABSENT       neither a trustworthy binary nor a plist exists
#   COMPLETE     regular binary and plist both exist
#   BINARY_ONLY  regular binary exists, plist missing
#   PLIST_ONLY   plist exists, usable binary missing
# A symlinked or non-regular expected binary is never counted: repair must
# not blindly follow it. Diagnostics and repair decide on this state; the
# older component_status helper is kept for install/update compatibility.
component_state() {
	_cst_comp="$1"
	_cst_bin=""
	_cst_plist=""
	case "$_cst_comp" in
		agent)
			_cst_bin="${BIN_DIR}/${AGENT_BIN}"
			_cst_plist="${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
			;;
		hub)
			_cst_bin="${BIN_DIR}/${HUB_BIN}"
			_cst_plist="${LAUNCHD_DIR}/${HUB_LABEL}.plist"
			;;
		*)
			return 1
			;;
	esac
	_cst_have_bin=0
	_cst_have_plist=0
	if [ -f "$_cst_bin" ] && [ ! -L "$_cst_bin" ]; then
		_cst_have_bin=1
	fi
	if [ -f "$_cst_plist" ] && [ ! -L "$_cst_plist" ]; then
		_cst_have_plist=1
	fi
	if [ "$_cst_have_bin" = "1" ] && [ "$_cst_have_plist" = "1" ]; then
		printf 'COMPLETE'
	elif [ "$_cst_have_bin" = "1" ]; then
		printf 'BINARY_ONLY'
	elif [ "$_cst_have_plist" = "1" ]; then
		printf 'PLIST_ONLY'
	else
		printf 'ABSENT'
	fi
	return 0
}

# release_is_current <agent|hub> <tag> — true when tracked release == tag.
release_is_current() {
	_ri_comp="$1"
	_ri_tag="$2"
	[ -n "$_ri_tag" ] || return 1
	_ri_cur=""
	case "$_ri_comp" in
		agent) _ri_cur=$(state_agent_release 2> /dev/null || true) ;;
		hub) _ri_cur=$(state_hub_release 2> /dev/null || true) ;;
		*) return 1 ;;
	esac
	[ -n "$_ri_cur" ] && [ "$_ri_cur" = "$_ri_tag" ]
}

# ------------------------------------------------------------- environment ---

check_root() {
	if [ "$(id -u)" != "0" ]; then
		cat >&2 << 'EOF'
[ERROR] This installer must run as root (it writes /usr/local/bin,
[ERROR] /var/lib and /Library/LaunchDaemons). Run:

  curl -fsSL https://raw.githubusercontent.com/nghianguyen150612/beszel-ios/ios/install.sh | sudo sh
EOF
		exit 1
	fi
}

check_device() {
	_cd_os=$(uname -s)
	[ "$_cd_os" = "Darwin" ] || die "Refusing to install: uname -s is '${_cd_os}', not Darwin. This installer targets jailbroken iOS only."
	# On iOS, uname -m returns the model identifier (e.g. "iPad4,4"),
	# NOT the CPU architecture, so detect arm64 via hw.cputype instead.
	# CPU_TYPE_ARM64 = 16777228 (0x0100000C).
	_cd_cputype=$(sysctl -n hw.cputype 2> /dev/null || true)
	[ "$_cd_cputype" = "16777228" ] || die "Refusing to install: hw.cputype is '${_cd_cputype}', not arm64 (16777228). This installer targets iOS arm64 only."
	_cd_machine=$(sysctl -n hw.machine 2> /dev/null || true)
	[ -n "$_cd_machine" ] || _cd_machine="unknown"
	case "$_cd_machine" in
		iPhone* | iPad* | iPod*)
			say_ok "Device identifier: ${_cd_machine}"
			;;
		*)
			if command -v sw_vers > /dev/null 2>&1 && sw_vers 2> /dev/null | grep -q -i "macos"; then
				die "Refusing to install: this looks like macOS (${_cd_machine}), not iOS."
			fi
			case "$_cd_machine" in
				Mac* | VMware*) die "Refusing to install: this looks like a Mac (${_cd_machine}), not iOS." ;;
			esac
			say_warn "Device identifier '${_cd_machine}' is not a recognised iPhone/iPad/iPod value; continuing anyway."
			;;
	esac
	say_warn "Tested: iPad mini 2 / Apple A7 / iOS 12.5.7. Other devices are currently community-tested / unverified."
}

check_layout() {
	_cl_d=""
	for _cl_d in "$BIN_DIR" "$LIB_DIR" "$LAUNCHD_DIR" "$LOG_DIR"; do
		if [ ! -d "$_cl_d" ]; then
			mkdir -p "$_cl_d" || die "Cannot create ${_cl_d}. This installer supports the standard layout (/usr/local/bin, /var/lib, /Library/LaunchDaemons); rootless layouts are not supported yet."
		fi
		if [ ! -w "$_cl_d" ]; then
			die "Directory ${_cl_d} is not writable. Rootless jailbreak layouts are not supported by this installer yet."
		fi
	done
}

# check_launchctl — minimal tool gate for local-only operations (menu,
# diagnostics, uninstall, purge). Uninstall must stay possible when the
# network or ldid are broken, so those are checked separately per flow.
check_launchctl() {
	if ! command -v launchctl > /dev/null 2>&1; then
		die "Missing required tool: launchctl. This installer manages iOS LaunchDaemons; aborting."
	fi
}

check_deps() {
	# Full gate for install / update / repair paths that download and sign
	# binaries. Deliberately NOT called for plain uninstall/purge.
	_missing=""
	for _dep in curl launchctl; do
		if ! command -v "$_dep" > /dev/null 2>&1; then
			_missing="${_missing} ${_dep}"
		fi
	done
	if ! command -v sha256sum > /dev/null 2>&1 && ! command -v shasum > /dev/null 2>&1 && ! command -v openssl > /dev/null 2>&1; then
		_missing="${_missing} (a SHA-256 tool: sha256sum, shasum or openssl)"
	fi
	if [ -n "$_missing" ]; then
		die "Missing required tools:${_missing}. Checksum verification cannot be skipped; aborting."
	fi
	if ! command -v ldid > /dev/null 2>&1; then
		say_warn "ldid is required to sign iOS binaries but was not found."
		if command -v apt-get > /dev/null 2>&1; then
			_ld_answer=""
			ask_tty "Install ldid using apt? [Y/n]" _ld_answer "Y" || exit 1
			case "$_ld_answer" in
				'' | [Yy] | [Yy][Ee][Ss])
					say_info "Running: apt-get install -y ldid (no system upgrade)."
					apt-get install -y ldid || die "apt-get failed to install ldid; aborting."
					;;
				*)
					die "ldid is mandatory; aborting."
					;;
			esac
			command -v ldid > /dev/null 2>&1 || die "ldid still missing after install attempt; aborting."
		else
			die "ldid is mandatory and apt-get is unavailable. Install ldid from your jailbreak repositories, then re-run."
		fi
	fi
	say_ok "Dependencies present: curl, launchctl, ldid, SHA-256 tool."
}

# ---------------------------------------------------------- temp workspace ---

cleanup_work_dir() {
	if [ -n "${WORK_DIR:-}" ] && [ -d "$WORK_DIR" ]; then
		case "$WORK_DIR" in
			*beszel-ios.*) rm -rf "$WORK_DIR" ;;
		esac
	fi
}

setup_work_dir() {
	if command -v mktemp > /dev/null 2>&1; then
		WORK_DIR=$(mktemp -d "${TMPDIR:-/tmp}/beszel-ios.XXXXXX") || die "Cannot create a temporary directory; aborting."
	else
		WORK_DIR="${TMPDIR:-/tmp}/beszel-ios.$$"
		mkdir -p "$WORK_DIR" || die "Cannot create a temporary directory; aborting."
		chmod 700 "$WORK_DIR" || die "Cannot secure the temporary directory; aborting."
	fi
	trap cleanup_work_dir EXIT INT TERM HUP
}

# ---------------------------------------------------------------- download ---

download() {
	_dl_url="$1"
	_dl_dest="$2"
	curl -fsSL -A "$USER_AGENT" --retry 3 --retry-delay 2 -o "$_dl_dest" "$_dl_url" || die "Download failed: ${_dl_url}"
	[ -s "$_dl_dest" ] || die "Download produced an empty file: ${_dl_dest}"
}

fetch_sums() {
	if [ "${SUMS_FETCHED:-0}" = "1" ] && [ -s "${WORK_DIR}/${SUMS_ASSET}" ]; then
		return 0
	fi
	say_info "Downloading SHA256SUMS..."
	download "$(current_release_base)/${SUMS_ASSET}" "${WORK_DIR}/${SUMS_ASSET}"
	validate_sums_file "${WORK_DIR}/${SUMS_ASSET}" || die "SHA256SUMS failed validation (expected exactly one entry each for ${AGENT_ASSET} and ${HUB_ASSET}). Aborting before trusting any binary."
	SUMS_FETCHED=1
	say_ok "SHA256SUMS structure valid."
}

fetch_and_verify_binary() {
	# $1 = asset name (also the release URL leaf); prints the verified path
	# on stdout. Status messages go to stderr so command substitution
	# captures only the path.
	_fb_asset="$1"
	say_info "Downloading ${_fb_asset}..." >&2
	download "$(current_release_base)/${_fb_asset}" "${WORK_DIR}/${_fb_asset}"
	_fb_expected=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$_fb_asset") || die "No checksum entry for ${_fb_asset}; refusing to install it."
	if verify_file "${WORK_DIR}/${_fb_asset}" "$_fb_expected"; then
		say_ok "Checksum verified: ${_fb_asset}" >&2
	else
		die "Checksum MISMATCH for ${_fb_asset}; refusing to install it."
	fi
	printf '%s' "${WORK_DIR}/${_fb_asset}"
	return 0
}

# ----------------------------------------------------------------- install ---

# install_binary <verified-source> <final-name>
# Returns 0 on fresh install, 2 when the destination already exists
# (never overwritten here), 1 on real failure.
install_binary() {
	_ib_src="$1"
	_ib_name="$2"
	_ib_dest="${BIN_DIR}/${_ib_name}"
	if [ -e "$_ib_dest" ]; then
		return 2
	fi
	cp "$_ib_src" "${_ib_dest}.new" || return 1
	if ! chown root:wheel "${_ib_dest}.new"; then
		rm -f "${_ib_dest}.new"
		return 1
	fi
	if ! chmod 755 "${_ib_dest}.new"; then
		rm -f "${_ib_dest}.new"
		return 1
	fi
	if ! ldid -S "${_ib_dest}.new"; then
		rm -f "${_ib_dest}.new"
		return 1
	fi
	mv -f "${_ib_dest}.new" "$_ib_dest" || return 1
	return 0
}

ensure_data_dir() {
	_ed_dir="$1"
	if [ ! -d "$_ed_dir" ]; then
		mkdir -p "$_ed_dir" || die "Cannot create data directory ${_ed_dir}."
		fix_path_owner "$_ed_dir" || die "Cannot set ownership on ${_ed_dir}."
		chmod 755 "$_ed_dir" || die "Cannot set permissions on ${_ed_dir}."
	fi
}

# plist_valid <file> — 0 when the file parses as XML with the first
# available checker. Returns 0 (with no claim) when no checker exists.
plist_valid() {
	_pv_file="$1"
	[ -f "$_pv_file" ] && [ -s "$_pv_file" ] || return 1
	if command -v plutil > /dev/null 2>&1; then
		plutil --lint "$_pv_file" > /dev/null 2>&1 || return 1
		return 0
	fi
	if command -v xmllint > /dev/null 2>&1; then
		xmllint --noout "$_pv_file" > /dev/null 2>&1 || return 1
		return 0
	fi
	if command -v python3 > /dev/null 2>&1; then
		python3 -c 'import sys,xml.dom.minidom; xml.dom.minidom.parse(sys.argv[1])' "$_pv_file" > /dev/null 2>&1 || return 1
		return 0
	fi
	return 0
}

check_plist() {
	_cp_file="$1"
	_cp_tool="none"
	if command -v plutil > /dev/null 2>&1; then
		_cp_tool="plutil"
	elif command -v xmllint > /dev/null 2>&1; then
		_cp_tool="xmllint"
	elif command -v python3 > /dev/null 2>&1; then
		_cp_tool="xml parser"
	fi
	if [ "$_cp_tool" = "none" ]; then
		say_warn "No plist checker available; skipping validation for ${_cp_file}."
		return 0
	fi
	if plist_valid "$_cp_file"; then
		say_ok "Plist valid (${_cp_tool}): ${_cp_file}"
	else
		die "Generated plist failed validation: ${_cp_file}"
	fi
}

write_agent_plist() {
	# $1 = hub public key, $2 = agent port, $3 = output path.
	_wa_key_esc=$(xml_escape "$1")
	_wa_port_esc=$(xml_escape "$2")
	cat > "$3" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>${AGENT_LABEL}</string>
	<key>ProgramArguments</key>
	<array>
		<string>${BIN_DIR}/${AGENT_BIN}</string>
	</array>
	<key>EnvironmentVariables</key>
	<dict>
		<key>DATA_DIR</key>
		<string>${LIB_DIR}/beszel-agent</string>
		<key>LISTEN</key>
		<string>:${_wa_port_esc}</string>
		<key>KEY</key>
		<string>${_wa_key_esc}</string>
		<key>LOG_LEVEL</key>
		<string>info</string>
		<key>PATH</key>
		<string>${LAUNCHD_PATH_VALUE}</string>
	</dict>
	<key>WorkingDirectory</key>
	<string>${LIB_DIR}/beszel-agent</string>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>${LOG_DIR}/beszel-agent.log</string>
	<key>StandardErrorPath</key>
	<string>${LOG_DIR}/beszel-agent.err.log</string>
</dict>
</plist>
EOF
}

write_hub_plist() {
	# $1 = hub port, $2 = output path.
	_wh_port_esc=$(xml_escape "$1")
	cat > "$2" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>${HUB_LABEL}</string>
	<key>ProgramArguments</key>
	<array>
		<string>${BIN_DIR}/${HUB_BIN}</string>
		<string>serve</string>
		<string>--http</string>
		<string>0.0.0.0:${_wh_port_esc}</string>
		<string>--dir</string>
		<string>${LIB_DIR}/beszel-hub</string>
	</array>
	<key>EnvironmentVariables</key>
	<dict>
		<key>PATH</key>
		<string>${LAUNCHD_PATH_VALUE}</string>
	</dict>
	<key>WorkingDirectory</key>
	<string>${LIB_DIR}/beszel-hub</string>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>ThrottleInterval</key>
	<integer>10</integer>
	<key>StandardOutPath</key>
	<string>${LOG_DIR}/beszel-hub.log</string>
	<key>StandardErrorPath</key>
	<string>${LOG_DIR}/beszel-hub.err.log</string>
</dict>
</plist>
EOF
}

install_plist() {
	# $1 = source tmp plist, $2 = final plist path.
	_ip_src="$1"
	_ip_dest="$2"
	cp "$_ip_src" "$_ip_dest" || die "Cannot write ${_ip_dest}."
	chown root:wheel "$_ip_dest" || die "Cannot set ownership on ${_ip_dest}."
	chmod 644 "$_ip_dest" || die "Cannot set permissions on ${_ip_dest}."
	check_plist "$_ip_dest"
}

start_service() {
	# $1 = label, $2 = plist path.
	launchctl unload "$2" > /dev/null 2>&1 || true
	if ! svc_load "$2"; then
		die "launchctl failed to load $1."
	fi
	if svc_running "$1"; then
		say_ok "Service running: $1"
	else
		say_warn "Loaded $1 but could not confirm it in 'launchctl list'."
	fi
}

# ------------------------------------------------------- service control ---

# Small launchctl wrappers so update/rollback paths share one implementation
# compatible with the validated iOS 12 behaviour (unload/load, no
# bootstrap/bootout assumptions).
svc_unload() {
	_su_plist="$1"
	launchctl unload "$_su_plist"
}

svc_load() {
	_sl_plist="$1"
	launchctl load -w "$_sl_plist"
}

# svc_running <label> — true only when launchctl lists the label with a live
# numeric PID (a bare grep would also match a loaded-but-stopped job whose
# PID column is "-").
svc_running() {
	_sv_label="$1"
	_sv_out=""
	_sv_out=$(launchctl list 2> /dev/null || true)
	[ -n "$_sv_out" ] || return 1
	_sv_line=$(printf '%s\n' "$_sv_out" | grep -F "$_sv_label" | head -n 1 || true)
	[ -n "$_sv_line" ] || return 1
	set -f
	# shellcheck disable=SC2086
	set -- $_sv_line
	set +f
	[ "$#" -ge 1 ] || return 1
	case "$1" in
		'' | '-' | '0' | *[!0-9]*) return 1 ;;
	esac
	return 0
}

# svc_loaded <label> — true when the label appears in launchctl output at
# all (running or stopped). False when launchctl yields nothing usable.
svc_loaded() {
	_sl2_label="$1"
	_sl2_out=""
	_sl2_out=$(launchctl list 2> /dev/null || true)
	[ -n "$_sl2_out" ] || return 1
	printf '%s\n' "$_sl2_out" | grep -q -F "$_sl2_label" || return 1
	return 0
}

# svc_pid <label> — print the launchd PID for a label, or "none" when no
# live numeric PID is listed. Read-only; used by diagnostics.
svc_pid() {
	_sp_label="$1"
	_sp_out=""
	_sp_out=$(launchctl list 2> /dev/null || true)
	_sp_line=$(printf '%s\n' "$_sp_out" | grep -F "$_sp_label" | head -n 1 || true)
	if [ -z "$_sp_line" ]; then
		printf 'none'
		return 1
	fi
	set -f
	# shellcheck disable=SC2086
	set -- $_sp_line
	set +f
	[ "$#" -ge 1 ] || {
		printf 'none'
		return 1
	}
	case "$1" in
		'' | '-' | '0' | *[!0-9]*)
			printf 'none'
			return 1
			;;
	esac
	printf '%s' "$1"
	return 0
}

# agent_post_update_ok <label> — the Agent's own health subcommand reflects
# recent Hub connections, so it cannot prove a fresh restart. Instead require
# a live launchd PID that survives a short bounded stabilization period.
agent_post_update_ok() {
	_ap_label="$1"
	_ap_settle="${BESZEL_UPDATE_SETTLE_SECS:-3}"
	svc_running "$_ap_label" || return 1
	case "$_ap_settle" in
		'' | *[!0-9]*) _ap_settle=3 ;;
	esac
	if [ "$_ap_settle" != "0" ]; then
		sleep "$_ap_settle" || true
	fi
	svc_running "$_ap_label" || return 1
	return 0
}

# hub_port_from_plist <plist> — conservative parse of the existing generated
# Hub plist's "0.0.0.0:<port>" argument. Never falls back to a default: when
# the configured port cannot be proven, the caller must abort the update.
# Ambiguous plists (zero or several distinct ports) are rejected.
hub_port_from_plist() {
	_hp_file="${1:-}"
	[ -f "$_hp_file" ] || return 1
	_hp_ports=$(sed -n 's/.*0\.0\.0\.0:\([0-9][0-9]*\).*/\1/p' "$_hp_file" 2> /dev/null | sort -u || true)
	_hp_n=$(printf '%s' "$_hp_ports" | grep -c '[0-9]' || true)
	[ "$_hp_n" = "1" ] || return 1
	_hp_port="$_hp_ports"
	valid_port "$_hp_port" || return 1
	printf '%s' "$_hp_port"
	return 0
}

# --------------------------------------------- safe plist value parsing ---

# The generated plists are XML property lists. iOS 12 plutil has no reliable
# extraction flags, so values are read with a conservative text parser tuned
# to the known installer-generated structure (<key> and <string> elements one
# per line). The parser never sources, evals or executes plist content; it
# fails closed on anything ambiguous. XML entities are decoded explicitly.
xml_decode() {
	printf '%s' "$1" | sed -e 's/&lt;/</g' -e 's/&gt;/>/g' -e 's/&quot;/"/g' -e "s/&apos;/'/g" -e 's/&amp;/\&/g'
}

# plist_pairs <file> — normalise one-per-line <key>/<string> elements into a
# KEY:<name> / VAL:<raw> stream. Lines in any other shape are ignored, which
# breaks key/value association downstream and therefore fails closed.
plist_pairs() {
	_pp_file="${1:-}"
	[ -f "$_pp_file" ] || return 1
	sed -n -e 's/^[[:space:]]*<key>\([^<]*\)<\/key>[[:space:]]*$/KEY:\1/p' -e 's/^[[:space:]]*<string>\(.*\)<\/string>[[:space:]]*$/VAL:\1/p' "$_pp_file" 2> /dev/null || return 1
	return 0
}

# plist_value_for <file> <key-name> — print the XML-decoded value of a key
# that must occur exactly once and be immediately followed by its <string>.
plist_value_for() {
	_vf_file="$1"
	_vf_name="$2"
	case "$_vf_name" in
		'' | *[!A-Za-z0-9_]* | *' '*) return 1 ;;
	esac
	_vf_stream=$(plist_pairs "$_vf_file") || return 1
	[ -n "$_vf_stream" ] || return 1
	_vf_n=$(printf '%s\n' "$_vf_stream" | grep -c "^KEY:${_vf_name}\$" || true)
	[ "$_vf_n" = "1" ] || return 1
	_vf_raw=$(printf '%s\n' "$_vf_stream" | grep -A1 "^KEY:${_vf_name}\$" | tail -n 1 || true)
	case "$_vf_raw" in
		VAL:*) _vf_enc=${_vf_raw#VAL:} ;;
		*) return 1 ;;
	esac
	xml_decode "$_vf_enc"
	return 0
}

# agent_port_from_plist <file> — LISTEN value must look like ":<port>".
agent_port_from_plist() {
	_apf_val=""
	_apf_val=$(plist_value_for "$1" "LISTEN") || return 1
	case "$_apf_val" in
		:*) _apf_port=${_apf_val#:} ;;
		*) return 1 ;;
	esac
	valid_port "$_apf_port" || return 1
	printf '%s' "$_apf_port"
	return 0
}

# agent_key_from_plist <file> — KEY value must pass SSH-key validation.
agent_key_from_plist() {
	_akf_val=""
	_akf_val=$(plist_value_for "$1" "KEY") || return 1
	valid_ssh_key "$_akf_val" || return 1
	printf '%s' "$_akf_val"
	return 0
}

wait_for_hub() {
	# $1 = port. Poll the local health endpoint; 0 = healthy.
	_wh_port="$1"
	_wh_i=0
	while [ "$_wh_i" -lt 30 ]; do
		if curl -fsSL --max-time 3 "http://127.0.0.1:${_wh_port}/api/health" > /dev/null 2>&1; then
			return 0
		fi
		sleep 2
		_wh_i=$((_wh_i + 1))
	done
	return 1
}

# ------------------------------------------------------ diagnostics ---

# Diagnostics are strictly read-only: they never write files, never load or
# unload services, and never print the Agent KEY value (only whether a valid
# key is configured).
diag_release_line() {
	# $1 = agent|hub. Prints a human release line.
	_dr_rel=""
	case "$1" in
		agent) _dr_rel=$(state_agent_release 2> /dev/null || true) ;;
		hub) _dr_rel=$(state_hub_release 2> /dev/null || true) ;;
	esac
	if [ -n "$_dr_rel" ]; then
		printf 'known release %s' "$_dr_rel"
	elif [ -f "$(state_path)" ]; then
		printf 'state present but release missing/invalid (legacy or corrupt)'
	else
		printf 'unknown (legacy installation, no state yet)'
	fi
	return 0
}

diag_service_line() {
	# $1 = label. Prints "loaded, PID <n>" / "loaded, not running" /
	# "not loaded" / "load state unknown".
	_ds_label="$1"
	_ds_out=""
	_ds_out=$(launchctl list 2> /dev/null || true)
	if [ -z "$_ds_out" ]; then
		printf 'load state unknown (launchctl unreadable)'
		return 0
	fi
	if ! printf '%s\n' "$_ds_out" | grep -q -F "$_ds_label"; then
		printf 'not loaded'
		return 0
	fi
	_ds_pid=""
	_ds_pid=$(svc_pid "$_ds_label" 2> /dev/null || true)
	if [ -n "$_ds_pid" ] && [ "$_ds_pid" != "none" ]; then
		printf 'loaded, PID %s' "$_ds_pid"
	else
		printf 'loaded, not running'
	fi
	return 0
}

diagnose_agent() {
	_da_bin="${BIN_DIR}/${AGENT_BIN}"
	_da_plist="${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
	_da_bak="${_da_bin}.bak"
	say_info "---- Agent diagnostics ----"
	if [ -e "$_da_bin" ]; then
		if [ -f "$_da_bin" ] && [ ! -L "$_da_bin" ]; then
			say_info "Binary: present (${_da_bin}, regular file)"
		else
			say_info "Binary: present but NOT a regular file (${_da_bin}); treated as untrusted"
		fi
	else
		say_info "Binary: missing (${_da_bin})"
	fi
	if [ -e "$_da_plist" ]; then
		if [ -f "$_da_plist" ] && [ ! -L "$_da_plist" ]; then
			if plist_valid "$_da_plist"; then
				say_info "Plist: present (${_da_plist}, XML valid)"
			else
				say_info "Plist: present (${_da_plist}, XML INVALID)"
			fi
		else
			say_info "Plist: present but NOT a regular file (${_da_plist}); treated as untrusted"
		fi
	else
		say_info "Plist: missing (${_da_plist})"
	fi
	say_info "Service: $(diag_service_line "$AGENT_LABEL")"
	_da_port=""
	_da_port=$(agent_port_from_plist "$_da_plist" 2> /dev/null || true)
	if [ -n "$_da_port" ]; then
		say_info "Agent port: ${_da_port}"
	else
		say_info "Agent port: unreadable (plist missing or unparseable)"
	fi
	if agent_key_from_plist "$_da_plist" > /dev/null 2>&1; then
		say_info "Agent key: configured"
	else
		say_info "Agent key: missing/unreadable"
	fi
	say_info "Release: $(diag_release_line agent)"
	if [ -s "$_da_bak" ]; then
		say_info "Backup: present (${_da_bak})"
	else
		say_info "Backup: absent"
	fi
	if [ -d "${LIB_DIR}/beszel-agent" ]; then
		say_info "Data directory: present (${LIB_DIR}/beszel-agent)"
	else
		say_info "Data directory: missing (${LIB_DIR}/beszel-agent)"
	fi
	return 0
}

diagnose_hub() {
	_dh_bin="${BIN_DIR}/${HUB_BIN}"
	_dh_plist="${LAUNCHD_DIR}/${HUB_LABEL}.plist"
	_dh_bak="${_dh_bin}.bak"
	say_info "---- Hub diagnostics ----"
	if [ -e "$_dh_bin" ]; then
		if [ -f "$_dh_bin" ] && [ ! -L "$_dh_bin" ]; then
			say_info "Binary: present (${_dh_bin}, regular file)"
		else
			say_info "Binary: present but NOT a regular file (${_dh_bin}); treated as untrusted"
		fi
	else
		say_info "Binary: missing (${_dh_bin})"
	fi
	if [ -e "$_dh_plist" ]; then
		if [ -f "$_dh_plist" ] && [ ! -L "$_dh_plist" ]; then
			if plist_valid "$_dh_plist"; then
				say_info "Plist: present (${_dh_plist}, XML valid)"
			else
				say_info "Plist: present (${_dh_plist}, XML INVALID)"
			fi
		else
			say_info "Plist: present but NOT a regular file (${_dh_plist}); treated as untrusted"
		fi
	else
		say_info "Plist: missing (${_dh_plist})"
	fi
	say_info "Service: $(diag_service_line "$HUB_LABEL")"
	_dh_port=""
	_dh_port=$(hub_port_from_plist "$_dh_plist" 2> /dev/null || true)
	if [ -n "$_dh_port" ]; then
		say_info "Hub port: ${_dh_port}"
		if curl -fsSL --max-time 3 "http://127.0.0.1:${_dh_port}/api/health" > /dev/null 2>&1; then
			say_info "Health: reachable (http://127.0.0.1:${_dh_port}/api/health)"
		else
			say_info "Health: unreachable (service may be starting or stopped)"
		fi
	else
		say_info "Hub port: unreadable (plist missing or unparseable)"
		say_info "Health: skipped (port unreadable)"
	fi
	say_info "Release: $(diag_release_line hub)"
	if [ -s "$_dh_bak" ]; then
		say_info "Backup: present (${_dh_bak})"
	else
		say_info "Backup: absent"
	fi
	if [ -d "${LIB_DIR}/beszel-hub" ]; then
		say_info "Data directory: present (${LIB_DIR}/beszel-hub)"
	else
		say_info "Data directory: missing (${LIB_DIR}/beszel-hub)"
	fi
	return 0
}

# ------------------------------------------------- update transactions ---

# stage_new_binary <asset> <final-name> — download, checksum-verify, sign and
# stage the new binary as <final-name>.new. The running service is untouched:
# any failure here (download, checksum, ldid) aborts before downtime.
stage_new_binary() {
	_sn_asset="$1"
	_sn_name="$2"
	_sn_dest="${BIN_DIR}/${_sn_name}"
	_sn_staged="${_sn_dest}.new"
	rm -f "$_sn_staged"
	_sn_verified=""
	_sn_verified=$(fetch_and_verify_binary "$_sn_asset") || return 1
	cp "$_sn_verified" "$_sn_staged" || {
		rm -f "$_sn_staged"
		return 1
	}
	if [ "$(id -u)" = "0" ]; then
		fix_path_owner "$_sn_staged" || {
			rm -f "$_sn_staged"
			say_err "Cannot set ownership on ${_sn_staged}."
			return 1
		}
	fi
	chmod 755 "$_sn_staged" || {
		rm -f "$_sn_staged"
		return 1
	}
	if ! ldid -S "$_sn_staged"; then
		rm -f "$_sn_staged"
		say_err "ldid signing failed for ${_sn_staged}; leaving the running service untouched."
		return 1
	fi
	[ -s "$_sn_staged" ] || {
		rm -f "$_sn_staged"
		return 1
	}
	printf '%s' "$_sn_staged"
	return 0
}

# backup_current_binary <final-name> — transactional copy of the working
# signed binary to <final-name>.bak via <final-name>.bak.new + rename.
backup_current_binary() {
	_bb_name="$1"
	_bb_final="${BIN_DIR}/${_bb_name}"
	_bb_bak="${_bb_final}.bak"
	_bb_tmp="${_bb_final}.bak.new"
	if [ -L "$_bb_final" ]; then
		say_err "Refusing to back up ${_bb_final}: it is a symlink."
		return 1
	fi
	if [ ! -f "$_bb_final" ]; then
		say_err "Expected binary ${_bb_final} is not a regular file."
		return 1
	fi
	if [ ! -s "$_bb_final" ]; then
		say_err "Expected binary ${_bb_final} is empty."
		return 1
	fi
	rm -f "$_bb_tmp"
	cp "$_bb_final" "$_bb_tmp" || {
		rm -f "$_bb_tmp"
		return 1
	}
	[ -s "$_bb_tmp" ] || {
		rm -f "$_bb_tmp"
		return 1
	}
	if [ "$(id -u)" = "0" ]; then
		fix_path_owner "$_bb_tmp" || {
			rm -f "$_bb_tmp"
			return 1
		}
	fi
	chmod 755 "$_bb_tmp" || {
		rm -f "$_bb_tmp"
		return 1
	}
	mv -f "$_bb_tmp" "$_bb_bak" || {
		rm -f "$_bb_tmp"
		return 1
	}
	[ -s "$_bb_bak" ] || return 1
	return 0
}

# rollback_critical — detail block when even the backup cannot be restored.
rollback_critical() {
	say_err "CRITICAL: automatic rollback failed."
	say_err "Binary: ${_rb_final}"
	say_err "Backup: ${_rb_bak}"
	say_err "Plist: ${_rb_plist}"
	say_err "Logs: ${LOG_DIR}/beszel-agent.log ${LOG_DIR}/beszel-agent.err.log ${LOG_DIR}/beszel-hub.log ${LOG_DIR}/beszel-hub.err.log"
	say_err "The backup and all data were left in place; Hub data in ${LIB_DIR}/beszel-hub was not touched."
	return 1
}

# rollback_component <agent|hub> — restore .bak atomically, reload the
# original unchanged plist, verify the previous service, keep state at the
# old release. The backup is never deleted.
rollback_component() {
	_rb_comp="$1"
	_rb_bin=""
	_rb_label=""
	case "$_rb_comp" in
		agent)
			_rb_bin="$AGENT_BIN"
			_rb_label="$AGENT_LABEL"
			;;
		hub)
			_rb_bin="$HUB_BIN"
			_rb_label="$HUB_LABEL"
			;;
		*)
			return 1
			;;
	esac
	_rb_final="${BIN_DIR}/${_rb_bin}"
	_rb_bak="${_rb_final}.bak"
	_rb_plist="${LAUNCHD_DIR}/${_rb_label}.plist"
	if [ ! -s "$_rb_bak" ]; then
		say_err "Rollback unavailable: backup ${_rb_bak} is missing."
		return 1
	fi
	say_info "Rolling back to previous binary..."
	svc_unload "$_rb_plist" > /dev/null 2>&1 || true
	_rb_tmp="${_rb_final}.rollback"
	rm -f "$_rb_tmp"
	cp "$_rb_bak" "$_rb_tmp" || {
		rollback_critical
		return 1
	}
	mv -f "$_rb_tmp" "$_rb_final" || {
		rm -f "$_rb_tmp"
		rollback_critical
		return 1
	}
	if [ "$(id -u)" = "0" ]; then
		fix_path_owner "$_rb_final" || {
			rollback_critical
			return 1
		}
	fi
	chmod 755 "$_rb_final" || {
		rollback_critical
		return 1
	}
	if ! svc_load "$_rb_plist"; then
		rollback_critical
		return 1
	fi
	case "$_rb_comp" in
		agent)
			if ! agent_post_update_ok "$_rb_label"; then
				rollback_critical
				return 1
			fi
			;;
		hub)
			_rb_port=""
			_rb_port=$(hub_port_from_plist "$_rb_plist" 2> /dev/null || true)
			if [ -n "$_rb_port" ]; then
				if ! wait_for_hub "$_rb_port"; then
					rollback_critical
					return 1
				fi
			elif ! svc_running "$_rb_label"; then
				rollback_critical
				return 1
			fi
			;;
	esac
	say_ok "Rollback succeeded."
	return 0
}

# ----------------------------------------------- plist transactions ---

# Plist backups live at fixed installer paths next to the live plist and are
# managed with temp-file + atomic rename only (never cp onto a live path, so
# a surprising destination symlink can never be followed). One previous
# known-good plist backup is kept; it is never confused with the binary .bak.
backup_plist() {
	# $1 = agent|hub.
	_bp_plist=""
	case "$1" in
		agent) _bp_plist="${LAUNCHD_DIR}/${AGENT_LABEL}.plist" ;;
		hub) _bp_plist="${LAUNCHD_DIR}/${HUB_LABEL}.plist" ;;
		*) return 1 ;;
	esac
	_bp_bak="${_bp_plist}.bak"
	if [ ! -f "$_bp_plist" ] || [ -L "$_bp_plist" ]; then
		say_err "Refusing to back up ${_bp_plist}: not a regular file."
		return 1
	fi
	if [ ! -s "$_bp_plist" ]; then
		say_err "Refusing to back up ${_bp_plist}: file is empty."
		return 1
	fi
	_bp_tmp="${_bp_bak}.new"
	rm -f "$_bp_tmp"
	cp "$_bp_plist" "$_bp_tmp" || {
		rm -f "$_bp_tmp"
		return 1
	}
	[ -s "$_bp_tmp" ] || {
		rm -f "$_bp_tmp"
		return 1
	}
	if [ "$(id -u)" = "0" ]; then
		fix_path_owner "$_bp_tmp" || {
			rm -f "$_bp_tmp"
			return 1
		}
	fi
	chmod 644 "$_bp_tmp" || {
		rm -f "$_bp_tmp"
		return 1
	}
	mv -f "$_bp_tmp" "$_bp_bak" || {
		rm -f "$_bp_tmp"
		return 1
	}
	[ -s "$_bp_bak" ] || return 1
	return 0
}

# replace_plist <agent|hub> <validated-tmp> — atomically move a prepared
# plist into place. A stray symlink at the destination is removed (link
# only, target untouched) so repair can proceed; anything else unexpected
# aborts.
replace_plist() {
	_rp_comp="$1"
	_rp_tmp="$2"
	_rp_dest=""
	case "$_rp_comp" in
		agent) _rp_dest="${LAUNCHD_DIR}/${AGENT_LABEL}.plist" ;;
		hub) _rp_dest="${LAUNCHD_DIR}/${HUB_LABEL}.plist" ;;
		*) return 1 ;;
	esac
	[ -f "$_rp_tmp" ] && [ -s "$_rp_tmp" ] || return 1
	if [ -L "$_rp_dest" ]; then
		rm -f "$_rp_dest" || return 1
	fi
	if [ -e "$_rp_dest" ] && { [ ! -f "$_rp_dest" ] || [ -L "$_rp_dest" ]; }; then
		say_err "Refusing to replace ${_rp_dest}: unexpected file kind."
		return 1
	fi
	mv -f "$_rp_tmp" "$_rp_dest" || return 1
	if [ "$(id -u)" = "0" ]; then
		fix_path_owner "$_rp_dest" || return 1
	fi
	chmod 644 "$_rp_dest" || return 1
	return 0
}

plist_rollback_critical() {
	say_err "CRITICAL: configuration rollback failed."
	say_err "Plist: ${_rp_final}"
	say_err "Plist backup: ${_rp_bak}"
	say_err "The service may be stopped; the backup was left in place."
	return 1
}

# rollback_plist <agent|hub> — restore the backed-up plist atomically,
# reload the original configuration and verify the previous service.
# Install state is left untouched.
rollback_plist() {
	_rp_comp="$1"
	_rp_label=""
	case "$_rp_comp" in
		agent) _rp_label="$AGENT_LABEL" ;;
		hub) _rp_label="$HUB_LABEL" ;;
		*) return 1 ;;
	esac
	_rp_final="${LAUNCHD_DIR}/${_rp_label}.plist"
	_rp_bak="${_rp_final}.bak"
	if [ ! -s "$_rp_bak" ]; then
		say_err "Configuration rollback unavailable: backup ${_rp_bak} is missing."
		return 1
	fi
	say_info "Restoring previous configuration..."
	svc_unload "$_rp_final" > /dev/null 2>&1 || true
	_rp_tmp="${_rp_final}.restore"
	rm -f "$_rp_tmp"
	cp "$_rp_bak" "$_rp_tmp" || {
		plist_rollback_critical
		return 1
	}
	mv -f "$_rp_tmp" "$_rp_final" || {
		rm -f "$_rp_tmp"
		plist_rollback_critical
		return 1
	}
	if [ "$(id -u)" = "0" ]; then
		fix_path_owner "$_rp_final" || {
			plist_rollback_critical
			return 1
		}
	fi
	chmod 644 "$_rp_final" || {
		plist_rollback_critical
		return 1
	}
	if ! svc_load "$_rp_final"; then
		plist_rollback_critical
		return 1
	fi
	case "$_rp_comp" in
		agent)
			if ! agent_post_update_ok "$_rp_label"; then
				plist_rollback_critical
				return 1
			fi
			;;
		hub)
			_rp_port=""
			_rp_port=$(hub_port_from_plist "$_rp_final" 2> /dev/null || true)
			if [ -n "$_rp_port" ]; then
				if ! wait_for_hub "$_rp_port"; then
					plist_rollback_critical
					return 1
				fi
			elif ! svc_running "$_rp_label"; then
				plist_rollback_critical
				return 1
			fi
			;;
	esac
	say_ok "Configuration rollback succeeded."
	return 0
}

# ----------------------------------------------- interrupt-safe traps ---

_update_rollback_if_needed() {
	if [ "${_UPDATE_TRAP_BUSY:-0}" = "1" ]; then
		return 0
	fi
	_UPDATE_TRAP_BUSY=1
	if [ -n "${_UPDATE_ACTIVE:-}" ] && [ "${_UPDATE_NEED_ROLLBACK:-0}" = "1" ]; then
		_UPDATE_NEED_ROLLBACK=0
		case "$_UPDATE_ACTIVE" in
			agent) rollback_component "agent" || true ;;
			hub) rollback_component "hub" || true ;;
			agent-plist) rollback_plist "agent" || true ;;
			hub-plist) rollback_plist "hub" || true ;;
			uninstall-agent) rollback_uninstall "agent" || true ;;
			uninstall-hub) rollback_uninstall "hub" || true ;;
		esac
	fi
	_UPDATE_TRAP_BUSY=0
	return 0
}

update_signal_trap() {
	if [ "${_UPDATE_TRAP_BUSY:-0}" = "1" ]; then
		exit 130
	fi
	_update_rollback_if_needed
	cleanup_work_dir
	exit 130
}

update_exit_trap() {
	_update_rollback_if_needed
	cleanup_work_dir
}

# update_begin <agent|hub|agent-plist|hub-plist|uninstall-agent|uninstall-hub>
# Arm the transaction traps (no rollback needed yet); the caller sets
# _UPDATE_NEED_ROLLBACK=1 at the point of no return.
update_begin() {
	_UPDATE_ACTIVE="$1"
	_UPDATE_NEED_ROLLBACK=0
	trap update_exit_trap EXIT
	trap update_signal_trap INT TERM HUP
}

update_end() {
	_UPDATE_ACTIVE=""
	_UPDATE_NEED_ROLLBACK=0
	trap cleanup_work_dir EXIT INT TERM HUP
}

# transact_agent_update <tag> — full safe Agent update. The plist (Hub key,
# port, data path, logging) is preserved byte-for-byte; state is committed
# only after the new Agent passes verification.
transact_agent_update() {
	_ta_tag="$1"
	_ta_plist="${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
	_ta_final="${BIN_DIR}/${AGENT_BIN}"
	if [ ! -f "$_ta_final" ] || [ ! -f "$_ta_plist" ]; then
		say_err "Agent installation is incomplete."
		say_err "Repair support will be added separately."
		return 1
	fi
	if [ -L "$_ta_final" ]; then
		say_err "Refusing to update ${_ta_final}: it is a symlink."
		return 1
	fi
	_ta_staged=""
	_ta_staged=$(stage_new_binary "$AGENT_ASSET" "$AGENT_BIN") || return 1
	backup_current_binary "$AGENT_BIN" || {
		rm -f "${_ta_final}.new"
		return 1
	}
	update_begin "agent"
	_UPDATE_NEED_ROLLBACK=1
	if ! svc_unload "$_ta_plist" > /dev/null 2>&1; then
		say_warn "launchctl unload reported an issue; continuing with replacement."
	fi
	_ta_ok=1
	if ! mv -f "$_ta_staged" "$_ta_final"; then
		say_err "Failed to replace ${_ta_final}."
		_ta_ok=0
	fi
	if [ "$_ta_ok" = "1" ]; then
		if ! svc_load "$_ta_plist"; then
			say_err "New Agent failed to load."
			_ta_ok=0
		fi
	fi
	if [ "$_ta_ok" = "1" ]; then
		if ! agent_post_update_ok "$AGENT_LABEL"; then
			say_err "New Agent failed post-update verification."
			_ta_ok=0
		fi
	fi
	if [ "$_ta_ok" = "1" ]; then
		_UPDATE_NEED_ROLLBACK=0
		_ta_sha=""
		_ta_sha=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$AGENT_ASSET" 2> /dev/null || true)
		if [ -n "$_ta_sha" ]; then
			state_write_component "agent" "$_ta_tag" "$_ta_sha" || say_warn "Agent updated but state recording failed."
		else
			say_warn "Agent updated but state recording failed (checksum entry missing)."
		fi
		update_end
		rm -f "${_ta_final}.rollback"
		say_ok "Agent updated to ${_ta_tag}."
		return 0
	fi
	rm -f "$_ta_staged"
	_UPDATE_NEED_ROLLBACK=0
	if rollback_component "agent"; then
		update_end
		return 1
	fi
	update_end
	return 1
}

# transact_hub_update <tag> — full safe Hub update. The plist and the Hub
# database directory are never rewritten, migrated or re-owned here.
transact_hub_update() {
	_th_tag="$1"
	_th_plist="${LAUNCHD_DIR}/${HUB_LABEL}.plist"
	_th_final="${BIN_DIR}/${HUB_BIN}"
	if [ ! -f "$_th_final" ] || [ ! -f "$_th_plist" ]; then
		say_err "Hub installation is incomplete."
		say_err "Repair support will be added separately."
		return 1
	fi
	if [ -L "$_th_final" ]; then
		say_err "Refusing to update ${_th_final}: it is a symlink."
		return 1
	fi
	_th_port=""
	_th_port=$(hub_port_from_plist "$_th_plist") || {
		say_err "Cannot determine the existing Hub health port safely."
		say_err "Hub update aborted without changing the service."
		return 1
	}
	_th_staged=""
	_th_staged=$(stage_new_binary "$HUB_ASSET" "$HUB_BIN") || return 1
	backup_current_binary "$HUB_BIN" || {
		rm -f "${_th_final}.new"
		return 1
	}
	update_begin "hub"
	_UPDATE_NEED_ROLLBACK=1
	if ! svc_unload "$_th_plist" > /dev/null 2>&1; then
		say_warn "launchctl unload reported an issue; continuing with replacement."
	fi
	_th_ok=1
	if ! mv -f "$_th_staged" "$_th_final"; then
		say_err "Failed to replace ${_th_final}."
		_th_ok=0
	fi
	if [ "$_th_ok" = "1" ]; then
		if ! svc_load "$_th_plist"; then
			say_err "New Hub failed to load."
			_th_ok=0
		fi
	fi
	if [ "$_th_ok" = "1" ]; then
		if ! wait_for_hub "$_th_port"; then
			say_err "New Hub failed health check."
			_th_ok=0
		fi
	fi
	if [ "$_th_ok" = "1" ]; then
		_UPDATE_NEED_ROLLBACK=0
		_th_sha=""
		_th_sha=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$HUB_ASSET" 2> /dev/null || true)
		if [ -n "$_th_sha" ]; then
			state_write_component "hub" "$_th_tag" "$_th_sha" || say_warn "Hub updated but state recording failed."
		else
			say_warn "Hub updated but state recording failed (checksum entry missing)."
		fi
		update_end
		rm -f "${_th_final}.rollback"
		say_ok "Hub updated to ${_th_tag}."
		return 0
	fi
	rm -f "$_th_staged"
	_UPDATE_NEED_ROLLBACK=0
	if rollback_component "hub"; then
		update_end
		return 1
	fi
	update_end
	return 1
}

# ------------------------------------------------------------- agent setup ---

prompt_agent_port() {
	AGENT_PORT=""
	ask_tty "Agent port" AGENT_PORT "$DEFAULT_AGENT_PORT" || exit 1
	[ -z "$AGENT_PORT" ] && AGENT_PORT="$DEFAULT_AGENT_PORT"
	while ! valid_port "$AGENT_PORT"; do
		say_warn "Invalid port '${AGENT_PORT}': enter a number 1-65535."
		AGENT_PORT=""
		ask_tty "Agent port" AGENT_PORT "$DEFAULT_AGENT_PORT" || exit 1
		[ -z "$AGENT_PORT" ] && AGENT_PORT="$DEFAULT_AGENT_PORT"
	done
}

prompt_agent_key() {
	AGENT_KEY=""
	ask_tty "Paste Beszel Hub public key" AGENT_KEY "" || exit 1
	while ! valid_ssh_key "$AGENT_KEY"; do
		say_warn "That does not look like an SSH public key (expected '<type> <base64> [comment]')."
		AGENT_KEY=""
		ask_tty "Paste Beszel Hub public key" AGENT_KEY "" || exit 1
	done
}

do_install_agent() {
	# Returns 0 when an agent install now exists (fresh or pre-existing).
	if agent_installed; then
		say_warn "Existing Beszel Agent installation detected."
		say_warn "Re-running with menu option 4 (Update) updates it; leaving it untouched."
		return 0
	fi
	_agent_src=$(fetch_and_verify_binary "$AGENT_ASSET") || exit 1
	if install_binary "$_agent_src" "$AGENT_BIN"; then
		say_ok "Installed ${BIN_DIR}/${AGENT_BIN} (signed with ldid)."
	else
		_ib_rc=$?
		if [ "$_ib_rc" = "2" ]; then
			say_warn "Existing Beszel Agent installation detected."
			say_warn "Re-running with menu option 4 (Update) updates it; leaving it untouched."
			return 0
		fi
		die "Failed to install ${BIN_DIR}/${AGENT_BIN}."
	fi
	ensure_data_dir "${LIB_DIR}/beszel-agent"
	prompt_agent_key
	prompt_agent_port
	write_agent_plist "$AGENT_KEY" "$AGENT_PORT" "${WORK_DIR}/dev.beszel.agent.plist"
	install_plist "${WORK_DIR}/dev.beszel.agent.plist" "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
	AGENT_KEY="redacted-after-use"
	start_service "$AGENT_LABEL" "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
	if [ -n "${LATEST_TAG:-}" ] && [ -s "${WORK_DIR}/${SUMS_ASSET}" ]; then
		_ia_sha=""
		_ia_sha=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$AGENT_ASSET" 2> /dev/null || true)
		if [ -n "$_ia_sha" ]; then
			state_write_component "agent" "$LATEST_TAG" "$_ia_sha" || say_warn "Agent installed but state recording failed."
		fi
	fi
	say_ok "Agent installed: binary ${BIN_DIR}/${AGENT_BIN}, data ${LIB_DIR}/beszel-agent, plist ${LAUNCHD_DIR}/${AGENT_LABEL}.plist, port ${AGENT_PORT}."
	return 0
}

# --------------------------------------------------------------- hub setup ---

prompt_hub_port() {
	HUB_PORT=""
	ask_tty "Hub port" HUB_PORT "$DEFAULT_HUB_PORT" || exit 1
	[ -z "$HUB_PORT" ] && HUB_PORT="$DEFAULT_HUB_PORT"
	while ! valid_port "$HUB_PORT"; do
		say_warn "Invalid port '${HUB_PORT}': enter a number 1-65535."
		HUB_PORT=""
		ask_tty "Hub port" HUB_PORT "$DEFAULT_HUB_PORT" || exit 1
		[ -z "$HUB_PORT" ] && HUB_PORT="$DEFAULT_HUB_PORT"
	done
}

do_install_hub() {
	# Returns 0 when a hub install now exists (fresh or pre-existing).
	if hub_installed; then
		say_warn "Existing Beszel Hub installation detected."
		say_warn "Re-running with menu option 4 (Update) updates it; leaving it untouched."
		return 0
	fi
	_hub_src=$(fetch_and_verify_binary "$HUB_ASSET") || exit 1
	if install_binary "$_hub_src" "$HUB_BIN"; then
		say_ok "Installed ${BIN_DIR}/${HUB_BIN} (signed with ldid)."
	else
		_ib_rc=$?
		if [ "$_ib_rc" = "2" ]; then
			say_warn "Existing Beszel Hub installation detected."
			say_warn "Re-running with menu option 4 (Update) updates it; leaving it untouched."
			return 0
		fi
		die "Failed to install ${BIN_DIR}/${HUB_BIN}."
	fi
	# The Hub data directory holds user database / accounts / config and is
	# never deleted or reset by this installer; it is only created if missing.
	ensure_data_dir "${LIB_DIR}/beszel-hub"
	prompt_hub_port
	write_hub_plist "$HUB_PORT" "${WORK_DIR}/dev.beszel.hub.plist"
	install_plist "${WORK_DIR}/dev.beszel.hub.plist" "${LAUNCHD_DIR}/${HUB_LABEL}.plist"
	start_service "$HUB_LABEL" "${LAUNCHD_DIR}/${HUB_LABEL}.plist"
	say_info "Waiting for Hub health at http://127.0.0.1:${HUB_PORT}/api/health ..."
	if wait_for_hub "$HUB_PORT"; then
		say_ok "Hub is healthy: http://127.0.0.1:${HUB_PORT}/api/health"
	else
		say_err "Hub did not become healthy. Logs: ${LOG_DIR}/beszel-hub.log ${LOG_DIR}/beszel-hub.err.log"
		say_err "Hub data in ${LIB_DIR}/beszel-hub was left untouched."
		return 1
	fi
	if [ -n "${LATEST_TAG:-}" ] && [ -s "${WORK_DIR}/${SUMS_ASSET}" ]; then
		_ih_sha=""
		_ih_sha=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$HUB_ASSET" 2> /dev/null || true)
		if [ -n "$_ih_sha" ]; then
			state_write_component "hub" "$LATEST_TAG" "$_ih_sha" || say_warn "Hub installed but state recording failed."
		fi
	fi
	say_ok "Hub installed: binary ${BIN_DIR}/${HUB_BIN}, data ${LIB_DIR}/beszel-hub, plist ${LAUNCHD_DIR}/${HUB_LABEL}.plist, port ${HUB_PORT}."
	return 0
}

# ------------------------------------------------------------------- flows ---

print_summary() {
	# $1 = mode label.
	say_info "---- Beszel iOS install summary ($1) ----"
	if agent_installed; then
		_ps_agent_rel=""
		_ps_agent_rel=$(state_agent_release 2> /dev/null || true)
		[ -n "$_ps_agent_rel" ] || _ps_agent_rel="unknown (installed before release tracking)"
		say_info "Agent: ${BIN_DIR}/${AGENT_BIN} | data ${LIB_DIR}/beszel-agent | plist ${LAUNCHD_DIR}/${AGENT_LABEL}.plist | logs ${LOG_DIR}/beszel-agent.log | release ${_ps_agent_rel}"
	fi
	if hub_installed; then
		_ps_hub_rel=""
		_ps_hub_rel=$(state_hub_release 2> /dev/null || true)
		[ -n "$_ps_hub_rel" ] || _ps_hub_rel="unknown (installed before release tracking)"
		say_info "Hub:   ${BIN_DIR}/${HUB_BIN} | data ${LIB_DIR}/beszel-hub | plist ${LAUNCHD_DIR}/${HUB_LABEL}.plist | logs ${LOG_DIR}/beszel-hub.log | release ${_ps_hub_rel}"
	fi
}

flow_agent() {
	ensure_pinned_release || exit 1
	fetch_sums || exit 1
	_agent_was_fresh=0
	if ! agent_installed; then
		_agent_was_fresh=1
	fi
	do_install_agent
	print_summary "agent"
	if [ "$_agent_was_fresh" = "1" ]; then
		say_info "Next: in the Hub UI, add this system using Host/IP 127.0.0.1 (same device) and Port: ${AGENT_PORT}."
	else
		say_info "The Agent was already installed; add this system in the Hub UI using its existing host/port configuration."
	fi
}

flow_hub() {
	ensure_pinned_release || exit 1
	fetch_sums || exit 1
	if do_install_hub; then
		print_summary "hub"
		say_info "Next: open the Hub UI in a browser, create your account, and copy the Hub public key (needed for Agent installs)."
	else
		print_summary "hub (health check failed)"
		exit 1
	fi
}

flow_both() {
	say_info "Installing Hub first, then Agent."
	ensure_pinned_release || exit 1
	fetch_sums || exit 1
	_hub_was_fresh=0
	if ! hub_installed; then
		_hub_was_fresh=1
	fi
	if ! do_install_hub; then
		print_summary "agent+hub (hub health check failed)"
		exit 1
	fi
	if [ "$_hub_was_fresh" = "1" ]; then
		say_info "Hub is new: open the Hub UI in a browser and create your account if needed."
	fi
	say_info "Copy the Hub public key shown in the Hub UI (system setup page)."
	wait_for_enter "When you have the Hub public key ready, continue with the Agent install."
	_agent_was_fresh=0
	if ! agent_installed; then
		_agent_was_fresh=1
	fi
	do_install_agent
	print_summary "agent+hub"
	if [ "$_agent_was_fresh" = "1" ]; then
		say_info "Next: in the Hub UI, add this system using Host/IP: 127.0.0.1 and Port: ${AGENT_PORT}."
	else
		say_info "The Agent was already installed; its existing configuration was left untouched."
	fi
}

# ------------------------------------------------------------------ update ---

# confirm_update <prompt> — Y/n confirmation read from /dev/tty.
confirm_update() {
	_cu_answer=""
	ask_tty "$1 [Y/n]" _cu_answer "Y" || return 1
	case "$_cu_answer" in
		'' | [Yy] | [Yy][Ee][Ss]) return 0 ;;
		*) return 1 ;;
	esac
}

# confirm_destructive <prompt> — y/N confirmation (default No) for actions
# that replace configuration or binaries. Read from /dev/tty.
confirm_destructive() {
	_cd_answer=""
	ask_tty "$1 [y/N]" _cd_answer "N" || return 1
	case "$_cd_answer" in
		[Yy] | [Yy][Ee][Ss]) return 0 ;;
		*) return 1 ;;
	esac
}

# current_release_label <agent|hub> — tracked tag, or the legacy notice.
current_release_label() {
	_cr_cur=""
	case "$1" in
		agent) _cr_cur=$(state_agent_release 2> /dev/null || true) ;;
		hub) _cr_cur=$(state_hub_release 2> /dev/null || true) ;;
		*) _cr_cur="" ;;
	esac
	if [ -n "$_cr_cur" ]; then
		printf '%s' "$_cr_cur"
	else
		printf 'unknown (installed before release tracking)'
	fi
	return 0
}

# update_agent_flow <tag> — confirm and run one safe Agent update.
update_agent_flow() {
	_ua_tag="$1"
	case "$(component_status agent)" in
		complete) ;;
		incomplete)
			say_err "Agent installation is incomplete."
			say_err "Repair support will be added separately."
			return 1
			;;
		*)
			say_err "Beszel Agent is not installed."
			return 1
			;;
	esac
	if release_is_current "agent" "$_ua_tag"; then
		say_info "Agent is already on ${_ua_tag}; nothing to do."
		return 0
	fi
	say_info "Current Agent: $(current_release_label agent)"
	say_info "Latest:        ${_ua_tag}"
	if [ "$(current_release_label agent)" = "unknown (installed before release tracking)" ]; then
		confirm_update "Update Agent to Latest?" || {
			say_info "Agent update cancelled."
			return 0
		}
	else
		confirm_update "Update Agent?" || {
			say_info "Agent update cancelled."
			return 0
		}
	fi
	fetch_sums || return 1
	if transact_agent_update "$_ua_tag"; then
		print_summary "agent update"
		return 0
	fi
	return 1
}

# update_hub_flow <tag> — confirm and run one safe Hub update.
update_hub_flow() {
	_uh_tag="$1"
	case "$(component_status hub)" in
		complete) ;;
		incomplete)
			say_err "Hub installation is incomplete."
			say_err "Repair support will be added separately."
			return 1
			;;
		*)
			say_err "Beszel Hub is not installed."
			return 1
			;;
	esac
	if release_is_current "hub" "$_uh_tag"; then
		say_info "Hub is already on ${_uh_tag}; nothing to do."
		return 0
	fi
	say_info "Current Hub: $(current_release_label hub)"
	say_info "Latest:      ${_uh_tag}"
	if [ "$(current_release_label hub)" = "unknown (installed before release tracking)" ]; then
		confirm_update "Update Hub to Latest?" || {
			say_info "Hub update cancelled."
			return 0
		}
	else
		confirm_update "Update Hub?" || {
			say_info "Hub update cancelled."
			return 0
		}
	fi
	fetch_sums || return 1
	if transact_hub_update "$_uh_tag"; then
		print_summary "hub update"
		return 0
	fi
	return 1
}

# update_both_flow <tag> — Hub first (strong HTTP health check), then Agent.
# Each component keeps its own transaction: a Hub failure leaves the Agent
# untouched; an Agent failure keeps the successfully updated Hub.
update_both_flow() {
	_ub_tag="$1"
	case "$(component_status agent)" in
		complete) ;;
		*)
			say_err "Agent installation is incomplete or missing."
			say_err "Repair support will be added separately."
			return 1
			;;
	esac
	case "$(component_status hub)" in
		complete) ;;
		*)
			say_err "Hub installation is incomplete or missing."
			say_err "Repair support will be added separately."
			return 1
			;;
	esac
	_ub_hub_needs=1
	_ub_agent_needs=1
	release_is_current "hub" "$_ub_tag" && _ub_hub_needs=0
	release_is_current "agent" "$_ub_tag" && _ub_agent_needs=0
	if [ "$_ub_hub_needs" = "0" ] && [ "$_ub_agent_needs" = "0" ]; then
		say_info "Hub is already on ${_ub_tag}; nothing to do."
		say_info "Agent is already on ${_ub_tag}; nothing to do."
		return 0
	fi
	say_info "Current Hub:   $(current_release_label hub)"
	say_info "Current Agent: $(current_release_label agent)"
	say_info "Latest:        ${_ub_tag}"
	confirm_update "Update Agent + Hub?" || {
		say_info "Update cancelled."
		return 0
	}
	fetch_sums || return 1
	if [ "$_ub_hub_needs" = "1" ]; then
		transact_hub_update "$_ub_tag" || {
			say_err "Hub update failed; Agent was left untouched."
			return 1
		}
	else
		say_info "Hub is already on ${_ub_tag}; skipping Hub."
	fi
	if [ "$_ub_agent_needs" = "1" ]; then
		transact_agent_update "$_ub_tag" || {
			say_err "Agent update failed and was rolled back; the Hub update (if any) was kept."
			return 1
		}
	else
		say_info "Agent is already on ${_ub_tag}; skipping Agent."
	fi
	print_summary "agent+hub update"
	return 0
}

# ---------------------------------------------------------- reconfigure ---

# reconfigure_agent_flow — change the Hub public key and/or the Agent port.
# Configuration only: the binary, data directory, .bak file and install
# state are never touched here.
reconfigure_agent_flow() {
	_ra_plist="${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
	if [ "$(component_state agent)" != "COMPLETE" ]; then
		say_err "Agent installation is incomplete; use Repair Agent first."
		return 1
	fi
	_ra_old_key=""
	_ra_old_key=$(agent_key_from_plist "$_ra_plist") || {
		say_err "Cannot interpret the existing Agent configuration safely."
		say_err "Use Repair Agent to recreate it."
		return 1
	}
	_ra_old_port=""
	_ra_old_port=$(agent_port_from_plist "$_ra_plist") || {
		say_err "Cannot interpret the existing Agent configuration safely."
		say_err "Use Repair Agent to recreate it."
		return 1
	}
	say_info "Current Agent port: ${_ra_old_port}"
	_ra_new_port="$_ra_old_port"
	if confirm_destructive "Change port?"; then
		prompt_agent_port || return 1
		_ra_new_port="$AGENT_PORT"
		AGENT_PORT=""
	fi
	_ra_new_key="$_ra_old_key"
	_ra_old_key="redacted-after-use"
	_ra_keep_key=1
	if confirm_update "Keep existing Hub public key?"; then
		:
	else
		prompt_agent_key || return 1
		_ra_new_key="$AGENT_KEY"
		_ra_keep_key=0
	fi
	AGENT_KEY="redacted-after-use"
	if [ "$_ra_new_port" = "$_ra_old_port" ] && [ "$_ra_keep_key" = "1" ]; then
		say_info "No Agent configuration changes requested."
		_ra_new_key="redacted-after-use"
		return 0
	fi
	_ra_tmp="${WORK_DIR}/dev.beszel.agent.plist.reconf"
	rm -f "$_ra_tmp"
	write_agent_plist "$_ra_new_key" "$_ra_new_port" "$_ra_tmp"
	_ra_new_key="redacted-after-use"
	if ! plist_valid "$_ra_tmp"; then
		rm -f "$_ra_tmp"
		say_err "Generated Agent configuration failed validation; nothing was changed."
		return 1
	fi
	backup_plist "agent" || {
		rm -f "$_ra_tmp"
		return 1
	}
	update_begin "agent-plist"
	_UPDATE_NEED_ROLLBACK=1
	if ! svc_unload "$_ra_plist" > /dev/null 2>&1; then
		say_warn "launchctl unload reported an issue; continuing with replacement."
	fi
	_ra_ok=1
	if ! replace_plist "agent" "$_ra_tmp"; then
		say_err "Failed to replace ${_ra_plist}."
		_ra_ok=0
	fi
	if [ "$_ra_ok" = "1" ]; then
		if ! svc_load "$_ra_plist"; then
			say_err "Agent with new configuration failed to load."
			_ra_ok=0
		fi
	fi
	if [ "$_ra_ok" = "1" ]; then
		if ! agent_post_update_ok "$AGENT_LABEL"; then
			say_err "Agent with new configuration failed verification."
			_ra_ok=0
		fi
	fi
	if [ "$_ra_ok" = "1" ]; then
		_UPDATE_NEED_ROLLBACK=0
		update_end
		rm -f "${_ra_plist}.restore"
		say_ok "Agent reconfigured (port ${_ra_new_port})."
		return 0
	fi
	rm -f "$_ra_tmp"
	_UPDATE_NEED_ROLLBACK=0
	if rollback_plist "agent"; then
		update_end
		return 1
	fi
	update_end
	return 1
}

# reconfigure_hub_flow — change the Hub listening port only. The binary,
# data directory (/var/lib/beszel-hub), .bak file and install state are
# never touched here.
reconfigure_hub_flow() {
	_rh_plist="${LAUNCHD_DIR}/${HUB_LABEL}.plist"
	if [ "$(component_state hub)" != "COMPLETE" ]; then
		say_err "Hub installation is incomplete; use Repair Hub first."
		return 1
	fi
	_rh_old_port=""
	_rh_old_port=$(hub_port_from_plist "$_rh_plist") || {
		say_err "Cannot interpret the existing Hub configuration safely."
		say_err "Use Repair Hub to recreate it."
		return 1
	}
	say_info "Current Hub port: ${_rh_old_port}"
	_rh_answer=""
	ask_tty "New Hub port" _rh_answer "$_rh_old_port" || return 1
	[ -n "$_rh_answer" ] || _rh_answer="$_rh_old_port"
	while ! valid_port "$_rh_answer"; do
		say_warn "Invalid port '${_rh_answer}': enter a number 1-65535."
		_rh_answer=""
		ask_tty "New Hub port" _rh_answer "$_rh_old_port" || return 1
		[ -n "$_rh_answer" ] || _rh_answer="$_rh_old_port"
	done
	if [ "$_rh_answer" = "$_rh_old_port" ]; then
		say_info "No Hub configuration changes requested."
		return 0
	fi
	_rh_new_port="$_rh_answer"
	_rh_tmp="${WORK_DIR}/dev.beszel.hub.plist.reconf"
	rm -f "$_rh_tmp"
	write_hub_plist "$_rh_new_port" "$_rh_tmp"
	if ! plist_valid "$_rh_tmp"; then
		rm -f "$_rh_tmp"
		say_err "Generated Hub configuration failed validation; nothing was changed."
		return 1
	fi
	backup_plist "hub" || {
		rm -f "$_rh_tmp"
		return 1
	}
	update_begin "hub-plist"
	_UPDATE_NEED_ROLLBACK=1
	if ! svc_unload "$_rh_plist" > /dev/null 2>&1; then
		say_warn "launchctl unload reported an issue; continuing with replacement."
	fi
	_rh_ok=1
	if ! replace_plist "hub" "$_rh_tmp"; then
		say_err "Failed to replace ${_rh_plist}."
		_rh_ok=0
	fi
	if [ "$_rh_ok" = "1" ]; then
		if ! svc_load "$_rh_plist"; then
			say_err "Hub with new configuration failed to load."
			_rh_ok=0
		fi
	fi
	if [ "$_rh_ok" = "1" ]; then
		if ! wait_for_hub "$_rh_new_port"; then
			say_err "Hub with new configuration failed health check."
			_rh_ok=0
		fi
	fi
	if [ "$_rh_ok" = "1" ]; then
		_UPDATE_NEED_ROLLBACK=0
		update_end
		rm -f "${_rh_plist}.restore"
		say_ok "Hub reconfigured (port ${_rh_new_port})."
		return 0
	fi
	rm -f "$_rh_tmp"
	_UPDATE_NEED_ROLLBACK=0
	if rollback_plist "hub"; then
		update_end
		return 1
	fi
	update_end
	return 1
}

# -------------------------------------------------------------- repair ---

# restart_existing_service <agent|hub> — reload the untouched existing
# plist and verify. No downloads, no binary or state changes.
restart_existing_service() {
	_rs_comp="$1"
	_rs_label=""
	case "$_rs_comp" in
		agent) _rs_label="$AGENT_LABEL" ;;
		hub) _rs_label="$HUB_LABEL" ;;
		*) return 1 ;;
	esac
	_rs_plist="${LAUNCHD_DIR}/${_rs_label}.plist"
	if ! plist_valid "$_rs_plist"; then
		say_err "The existing plist failed validation: ${_rs_plist}"
		return 1
	fi
	svc_unload "$_rs_plist" > /dev/null 2>&1 || true
	if ! svc_load "$_rs_plist"; then
		say_err "Existing service failed to load."
		return 1
	fi
	case "$_rs_comp" in
		agent)
			agent_post_update_ok "$_rs_label" || {
				say_err "Existing Agent failed verification."
				return 1
			}
			;;
		hub)
			_rs_port=""
			_rs_port=$(hub_port_from_plist "$_rs_plist") || {
				say_err "Cannot determine the Hub health port safely."
				return 1
			}
			wait_for_hub "$_rs_port" || {
				say_err "Existing Hub failed health check."
				return 1
			}
			;;
	esac
	say_ok "Service restarted with its existing configuration."
	return 0
}

# restore_missing_binary <agent|hub> <tag> — reinstall only the missing
# binary from the pinned Latest release while preserving the existing
# plist (and Hub database). Records release state after service success.
# On failure the just-placed binary is removed again so the filesystem
# returns to its prior state; install state is left untouched.
restore_missing_binary() {
	_rm_comp="$1"
	_rm_tag="$2"
	_rm_bin=""
	_rm_asset=""
	case "$_rm_comp" in
		agent)
			_rm_bin="$AGENT_BIN"
			_rm_asset="$AGENT_ASSET"
			;;
		hub)
			_rm_bin="$HUB_BIN"
			_rm_asset="$HUB_ASSET"
			;;
		*)
			return 1
			;;
	esac
	_rm_final="${BIN_DIR}/${_rm_bin}"
	_rm_plist="${LAUNCHD_DIR}/dev.beszel.${_rm_comp}.plist"
	if [ "$(component_state "$_rm_comp")" != "PLIST_ONLY" ]; then
		say_err "Binary restore needs a plist without a usable binary; state changed."
		return 1
	fi
	case "$_rm_comp" in
		agent)
			agent_key_from_plist "$_rm_plist" > /dev/null 2>&1 || {
				say_err "Cannot interpret the existing Agent configuration; refusing to guess it."
				return 1
			}
			agent_port_from_plist "$_rm_plist" > /dev/null 2>&1 || {
				say_err "Cannot interpret the existing Agent configuration; refusing to guess it."
				return 1
			}
			;;
		hub)
			hub_port_from_plist "$_rm_plist" > /dev/null 2>&1 || {
				say_err "Cannot interpret the existing Hub configuration; refusing to guess it."
				return 1
			}
			;;
	esac
	_rm_staged=""
	_rm_staged=$(stage_new_binary "$_rm_asset" "$_rm_bin") || return 1
	if [ -L "$_rm_final" ]; then
		rm -f "$_rm_final" || {
			rm -f "$_rm_staged"
			return 1
		}
	fi
	if [ -e "$_rm_final" ]; then
		say_err "A binary appeared at ${_rm_final}; refusing to overwrite it."
		rm -f "$_rm_staged"
		return 1
	fi
	mv -f "$_rm_staged" "$_rm_final" || {
		rm -f "$_rm_staged"
		return 1
	}
	if [ "$(id -u)" = "0" ]; then
		fix_path_owner "$_rm_final" || {
			rm -f "$_rm_final"
			return 1
		}
	fi
	chmod 755 "$_rm_final" || {
		rm -f "$_rm_final"
		return 1
	}
	svc_unload "$_rm_plist" > /dev/null 2>&1 || true
	if ! svc_load "$_rm_plist"; then
		say_err "Restored binary failed to load; removing it again."
		svc_unload "$_rm_plist" > /dev/null 2>&1 || true
		rm -f "$_rm_final"
		return 1
	fi
	case "$_rm_comp" in
		agent)
			if ! agent_post_update_ok "dev.beszel.${_rm_comp}"; then
				say_err "Restored binary failed verification; removing it again."
				svc_unload "$_rm_plist" > /dev/null 2>&1 || true
				rm -f "$_rm_final"
				return 1
			fi
			;;
		hub)
			_rm_port=""
			_rm_port=$(hub_port_from_plist "$_rm_plist" 2> /dev/null || true)
			if [ -z "$_rm_port" ] || ! wait_for_hub "$_rm_port"; then
				say_err "Restored binary failed verification; removing it again."
				svc_unload "$_rm_plist" > /dev/null 2>&1 || true
				rm -f "$_rm_final"
				return 1
			fi
			;;
	esac
	_rm_sha=""
	_rm_sha=$(sums_hash_for "${WORK_DIR}/${SUMS_ASSET}" "$_rm_asset" 2> /dev/null || true)
	if [ -n "$_rm_sha" ]; then
		state_write_component "$_rm_comp" "$_rm_tag" "$_rm_sha" || say_warn "Binary restored but state recording failed."
	else
		say_warn "Binary restored but state recording failed (checksum entry missing)."
	fi
	say_ok "Binary restored from ${_rm_tag}; existing configuration preserved."
	return 0
}

# repair_agent_flow — conservative Agent repair preserving configuration.
repair_agent_flow() {
	_rpa_st=$(component_state "agent")
	case "$_rpa_st" in
		ABSENT)
			say_err "Agent is not installed. Use Install Agent."
			return 1
			;;
		COMPLETE)
			if svc_running "$AGENT_LABEL"; then
				say_ok "Agent service is already running."
				return 0
			fi
			say_info "Agent installed but not running; restarting existing service..."
			if restart_existing_service "agent"; then
				return 0
			fi
			say_info "Existing configuration will be preserved."
			if confirm_destructive "Replace Agent binary with Latest release?"; then
				ensure_pinned_release || return 1
				fetch_sums || return 1
				if transact_agent_update "$LATEST_TAG"; then
					return 0
				fi
				return 1
			fi
			say_info "Binary replacement declined."
			return 1
			;;
		PLIST_ONLY)
			say_info "Agent binary missing; existing configuration will be preserved."
			ensure_pinned_release || return 1
			fetch_sums || return 1
			if restore_missing_binary "agent" "$LATEST_TAG"; then
				return 0
			fi
			return 1
			;;
		BINARY_ONLY)
			say_info "Agent configuration is missing."
			say_info "Repair requires recreating Agent configuration."
			if confirm_destructive "Continue?"; then
				:
			else
				say_info "Repair cancelled."
				return 0
			fi
			prompt_agent_key || return 1
			prompt_agent_port || return 1
			_rpc_tmp="${WORK_DIR}/dev.beszel.agent.plist.repair"
			rm -f "$_rpc_tmp"
			write_agent_plist "$AGENT_KEY" "$AGENT_PORT" "$_rpc_tmp"
			AGENT_KEY="redacted-after-use"
			AGENT_PORT=""
			if ! plist_valid "$_rpc_tmp"; then
				rm -f "$_rpc_tmp"
				say_err "Generated Agent configuration failed validation; nothing was changed."
				return 1
			fi
			replace_plist "agent" "$_rpc_tmp" || return 1
			svc_unload "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" > /dev/null 2>&1 || true
			if ! svc_load "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"; then
				say_err "Agent with recreated configuration failed to load; removing it again."
				rm -f "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
				return 1
			fi
			if ! agent_post_update_ok "$AGENT_LABEL"; then
				say_err "Agent with recreated configuration failed verification; removing it again."
				svc_unload "${LAUNCHD_DIR}/${AGENT_LABEL}.plist" > /dev/null 2>&1 || true
				rm -f "${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
				return 1
			fi
			say_ok "Agent configuration recreated; existing binary and data preserved."
			return 0
			;;
	esac
	return 1
}

# repair_hub_flow — conservative Hub repair preserving plist and database.
repair_hub_flow() {
	_rph_st=$(component_state "hub")
	case "$_rph_st" in
		ABSENT)
			say_err "Hub is not installed. Use Install Hub."
			return 1
			;;
		COMPLETE)
			_rph_port=""
			_rph_port=$(hub_port_from_plist "${LAUNCHD_DIR}/${HUB_LABEL}.plist" 2> /dev/null || true)
			if [ -n "$_rph_port" ] && svc_running "$HUB_LABEL" && curl -fsSL --max-time 3 "http://127.0.0.1:${_rph_port}/api/health" > /dev/null 2>&1; then
				say_ok "Hub service is already healthy."
				return 0
			fi
			say_info "Hub installed but unhealthy; restarting existing service..."
			if restart_existing_service "hub"; then
				return 0
			fi
			say_info "Existing configuration and database will be preserved."
			if confirm_destructive "Replace Hub binary with Latest release?"; then
				ensure_pinned_release || return 1
				fetch_sums || return 1
				if transact_hub_update "$LATEST_TAG"; then
					return 0
				fi
				return 1
			fi
			say_info "Binary replacement declined."
			return 1
			;;
		PLIST_ONLY)
			say_info "Hub binary missing; existing configuration and database will be preserved."
			ensure_pinned_release || return 1
			fetch_sums || return 1
			if restore_missing_binary "hub" "$LATEST_TAG"; then
				return 0
			fi
			return 1
			;;
		BINARY_ONLY)
			say_info "Hub configuration is missing."
			say_info "Repair requires recreating Hub configuration (port only)."
			if confirm_destructive "Continue?"; then
				:
			else
				say_info "Repair cancelled."
				return 0
			fi
			prompt_hub_port || return 1
			_rph_tmp="${WORK_DIR}/dev.beszel.hub.plist.repair"
			rm -f "$_rph_tmp"
			write_hub_plist "$HUB_PORT" "$_rph_tmp"
			HUB_PORT=""
			if ! plist_valid "$_rph_tmp"; then
				rm -f "$_rph_tmp"
				say_err "Generated Hub configuration failed validation; nothing was changed."
				return 1
			fi
			replace_plist "hub" "$_rph_tmp" || return 1
			svc_unload "${LAUNCHD_DIR}/${HUB_LABEL}.plist" > /dev/null 2>&1 || true
			if ! svc_load "${LAUNCHD_DIR}/${HUB_LABEL}.plist"; then
				say_err "Hub with recreated configuration failed to load; removing it again."
				rm -f "${LAUNCHD_DIR}/${HUB_LABEL}.plist"
				return 1
			fi
			_rph_new_port=""
			_rph_new_port=$(hub_port_from_plist "${LAUNCHD_DIR}/${HUB_LABEL}.plist" 2> /dev/null || true)
			if [ -z "$_rph_new_port" ] || ! wait_for_hub "$_rph_new_port"; then
				say_err "Hub with recreated configuration failed health check; removing it again."
				svc_unload "${LAUNCHD_DIR}/${HUB_LABEL}.plist" > /dev/null 2>&1 || true
				rm -f "${LAUNCHD_DIR}/${HUB_LABEL}.plist"
				return 1
			fi
			say_ok "Hub configuration recreated; existing binary and database preserved."
			return 0
			;;
	esac
	return 1
}

# ------------------------------------------------------------ uninstall ---

# Uninstall removes application/service artifacts only. User data
# (/var/lib/beszel-agent, /var/lib/beszel-hub) is ALWAYS preserved by normal
# uninstall; deleting data is a separate explicit purge step. Every removal
# target below is a fixed installer path — nothing is built from user input,
# no glob deletion is used, and symlinks at primary locations fail closed.

# validate_uninstall_targets <agent|hub> — every existing removable
# application file must be an ordinary file. A symlink (or other surprise)
# at the primary binary/plist refuses the whole uninstall; surprising
# backup/log kinds also refuse before anything is touched.
validate_uninstall_targets() {
	_vu_comp="$1"
	_vu_bin=""
	_vu_plist=""
	case "$_vu_comp" in
		agent)
			_vu_bin="${BIN_DIR}/${AGENT_BIN}"
			_vu_plist="${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
			;;
		hub)
			_vu_bin="${BIN_DIR}/${HUB_BIN}"
			_vu_plist="${LAUNCHD_DIR}/${HUB_LABEL}.plist"
			;;
		*)
			return 1
			;;
	esac
	if [ -L "$_vu_bin" ]; then
		say_err "Unexpected symlink detected at ${_vu_bin}."
		say_err "Refusing automatic uninstall; inspect it manually."
		return 1
	fi
	if [ -L "$_vu_plist" ]; then
		say_err "Unexpected symlink detected at ${_vu_plist}."
		say_err "Refusing automatic uninstall; inspect it manually."
		return 1
	fi
	if [ -e "$_vu_bin" ] && [ ! -f "$_vu_bin" ]; then
		say_err "Unexpected file kind at ${_vu_bin}."
		say_err "Refusing automatic uninstall; inspect it manually."
		return 1
	fi
	if [ -e "$_vu_plist" ] && [ ! -f "$_vu_plist" ]; then
		say_err "Unexpected file kind at ${_vu_plist}."
		say_err "Refusing automatic uninstall; inspect it manually."
		return 1
	fi
	_vu_bak1="${_vu_bin}.bak"
	_vu_bak2="${_vu_plist}.bak"
	_vu_log1=""
	_vu_log2=""
	case "$_vu_comp" in
		agent)
			_vu_log1="${LOG_DIR}/beszel-agent.log"
			_vu_log2="${LOG_DIR}/beszel-agent.err.log"
			;;
		hub)
			_vu_log1="${LOG_DIR}/beszel-hub.log"
			_vu_log2="${LOG_DIR}/beszel-hub.err.log"
			;;
	esac
	for _vu_p in "$_vu_bak1" "$_vu_bak2" "$_vu_log1" "$_vu_log2"; do
		if [ -e "$_vu_p" ] && [ ! -f "$_vu_p" ] && [ ! -L "$_vu_p" ]; then
			say_err "Unexpected file kind at ${_vu_p}."
			say_err "Refusing automatic uninstall; inspect it manually."
			return 1
		fi
	done
	return 0
}

uninstall_critical() {
	say_err "CRITICAL: uninstall rollback failed."
	say_err "Binary: ${_un_bin}"
	say_err "Plist: ${_un_plist}"
	say_err "Staged binary: ${_UN_STAGED_BIN:-none}"
	say_err "Staged plist: ${_UN_STAGED_PLIST:-none}"
	say_err "Installer state: $(state_path)"
	say_err "The service may be stopped; staged files and backups were left in place."
	return 1
}

# rollback_uninstall <agent|hub> — move staged files back to their live
# names, reload the service when it was loaded before, and verify. Uses the
# _UN_* transaction globals captured before staging.
rollback_uninstall() {
	_ru_comp="$1"
	_ru_label=""
	case "$_ru_comp" in
		agent) _ru_label="$AGENT_LABEL" ;;
		hub) _ru_label="$HUB_LABEL" ;;
		*) return 1 ;;
	esac
	_un_bin="${BIN_DIR}/beszel-${_ru_comp}"
	_un_plist="${LAUNCHD_DIR}/dev.beszel.${_ru_comp}.plist"
	say_info "Restoring staged application files..."
	if [ -n "${_UN_STAGED_BIN:-}" ] && [ -e "${_UN_STAGED_BIN:-}" ]; then
		if [ -e "$_un_bin" ]; then
			uninstall_critical
			return 1
		fi
		mv -f "$_UN_STAGED_BIN" "$_un_bin" || {
			uninstall_critical
			return 1
		}
	fi
	if [ -n "${_UN_STAGED_PLIST:-}" ] && [ -e "${_UN_STAGED_PLIST:-}" ]; then
		if [ -e "$_un_plist" ]; then
			uninstall_critical
			return 1
		fi
		mv -f "$_UN_STAGED_PLIST" "$_un_plist" || {
			uninstall_critical
			return 1
		}
	fi
	if [ "${_UN_HAD_BIN:-0}" = "1" ] && [ ! -e "$_un_bin" ]; then
		uninstall_critical
		return 1
	fi
	if [ "${_UN_HAD_PLIST:-0}" = "1" ] && [ ! -e "$_un_plist" ]; then
		uninstall_critical
		return 1
	fi
	if [ "${_UN_WAS_LOADED:-0}" = "1" ]; then
		if [ ! -f "$_un_plist" ]; then
			uninstall_critical
			return 1
		fi
		if ! svc_load "$_un_plist"; then
			uninstall_critical
			return 1
		fi
		case "$_ru_comp" in
			agent)
				if ! agent_post_update_ok "$_ru_label"; then
					uninstall_critical
					return 1
				fi
				;;
			hub)
				_ru_port=""
				_ru_port=$(hub_port_from_plist "$_un_plist" 2> /dev/null || true)
				if [ -n "$_ru_port" ]; then
					if ! wait_for_hub "$_ru_port"; then
						uninstall_critical
						return 1
					fi
				elif ! svc_running "$_ru_label"; then
					uninstall_critical
					return 1
				fi
				;;
		esac
	fi
	say_ok "Uninstall rollback succeeded."
	return 0
}

# transact_uninstall <agent|hub> — transactional application uninstall:
# unload, stage the live binary/plist aside, verify the service is gone,
# clear installer state, then finalize artifact deletion. Data directories
# are never touched. No prompts, no network. Returns 0/1.
transact_uninstall() {
	_tu_comp="$1"
	_tu_label=""
	case "$_tu_comp" in
		agent) _tu_label="$AGENT_LABEL" ;;
		hub) _tu_label="$HUB_LABEL" ;;
		*) return 1 ;;
	esac
	_tu_bin="${BIN_DIR}/beszel-${_tu_comp}"
	_tu_plist="${LAUNCHD_DIR}/dev.beszel.${_tu_comp}.plist"
	_tu_bin_bak="${_tu_bin}.bak"
	_tu_plist_bak="${_tu_plist}.bak"
	_tu_log1=""
	_tu_log2=""
	case "$_tu_comp" in
		agent)
			_tu_log1="${LOG_DIR}/beszel-agent.log"
			_tu_log2="${LOG_DIR}/beszel-agent.err.log"
			;;
		hub)
			_tu_log1="${LOG_DIR}/beszel-hub.log"
			_tu_log2="${LOG_DIR}/beszel-hub.err.log"
			;;
	esac
	if [ "$(component_state "$_tu_comp")" = "ABSENT" ]; then
		say_err "No ${_tu_comp} application artifacts to uninstall."
		return 1
	fi
	validate_uninstall_targets "$_tu_comp" || return 1
	_UN_COMP="$_tu_comp"
	_UN_WAS_LOADED=0
	_UN_HAD_BIN=0
	_UN_HAD_PLIST=0
	_UN_STAGED_BIN=""
	_UN_STAGED_PLIST=""
	[ -e "$_tu_bin" ] && _UN_HAD_BIN=1
	[ -e "$_tu_plist" ] && _UN_HAD_PLIST=1
	if svc_loaded "$_tu_label"; then
		_UN_WAS_LOADED=1
	fi
	update_begin "uninstall-${_tu_comp}"
	_UPDATE_NEED_ROLLBACK=1
	if [ -f "$_tu_plist" ]; then
		if ! svc_unload "$_tu_plist" > /dev/null 2>&1; then
			say_warn "launchctl unload reported an issue; verifying service state."
		fi
		if svc_loaded "$_tu_label"; then
			say_err "Service ${_tu_label} is still loaded after unload."
			say_err "Aborting before deleting anything."
			_UPDATE_NEED_ROLLBACK=0
			update_end
			return 1
		fi
	else
		if svc_loaded "$_tu_label"; then
			say_err "Service ${_tu_label} is still loaded but its plist is missing."
			say_err "Refusing automatic uninstall; inspect it manually."
			_UPDATE_NEED_ROLLBACK=0
			update_end
			return 1
		fi
	fi
	_tu_ok=1
	_tu_suffix=".uninstall.$$"
	if [ -e "$_tu_bin" ]; then
		_UN_STAGED_BIN="${_tu_bin}${_tu_suffix}"
		rm -f "$_UN_STAGED_BIN" || true
		if ! mv -f "$_tu_bin" "$_UN_STAGED_BIN"; then
			say_err "Failed to stage ${_tu_bin} aside."
			_tu_ok=0
		fi
	fi
	if [ "$_tu_ok" = "1" ] && [ -e "$_tu_plist" ]; then
		_UN_STAGED_PLIST="${_tu_plist}${_tu_suffix}"
		rm -f "$_UN_STAGED_PLIST" || true
		if ! mv -f "$_tu_plist" "$_UN_STAGED_PLIST"; then
			say_err "Failed to stage ${_tu_plist} aside."
			_tu_ok=0
		fi
	fi
	if [ "$_tu_ok" = "1" ]; then
		if svc_loaded "$_tu_label"; then
			say_err "Service ${_tu_label} is still present after staging."
			_tu_ok=0
		fi
	fi
	if [ "$_tu_ok" = "1" ]; then
		if ! state_clear_component "$_tu_comp"; then
			say_err "Installer state cleanup failed."
			_tu_ok=0
		fi
	fi
	if [ "$_tu_ok" = "1" ]; then
		_UPDATE_NEED_ROLLBACK=0
		update_end
		_tu_final_rc=0
		_tu_leftovers=""
		for _tu_f in "$_UN_STAGED_BIN" "$_UN_STAGED_PLIST" "$_tu_bin_bak" "$_tu_plist_bak" "$_tu_log1" "$_tu_log2"; do
			if [ -n "$_tu_f" ] && [ -e "$_tu_f" ]; then
				if ! rm -f "$_tu_f"; then
					_tu_leftovers="${_tu_leftovers} ${_tu_f}"
					_tu_final_rc=1
				fi
			fi
		done
		if [ "$_tu_final_rc" = "0" ]; then
			say_ok "Application artifacts removed."
			return 0
		fi
		say_warn "Application uninstalled, but some artifacts could not be removed:${_tu_leftovers}"
		return 1
	fi
	_UPDATE_NEED_ROLLBACK=0
	if rollback_uninstall "$_tu_comp"; then
		update_end
		return 1
	fi
	update_end
	return 1
}

# ----------------------------------------------- data purge (explicit) ---

# validate_purge_path <path> <agent|hub> — the path must be exactly the
# expected data directory: non-empty, not /, not LIB_DIR itself, not a
# symlink, an actual directory.
validate_purge_path() {
	_vp_path="${1:-}"
	_vp_comp="${2:-}"
	_vp_want=""
	case "$_vp_comp" in
		agent) _vp_want="${LIB_DIR}/beszel-agent" ;;
		hub) _vp_want="${LIB_DIR}/beszel-hub" ;;
		*) return 1 ;;
	esac
	[ -n "$_vp_path" ] || return 1
	[ "$_vp_path" = "$_vp_want" ] || return 1
	[ "$_vp_path" != "/" ] || return 1
	[ "$_vp_path" != "${LIB_DIR}" ] || return 1
	[ "${LIB_DIR}" != "/" ] || return 1
	[ ! -L "$_vp_path" ] || return 1
	[ -d "$_vp_path" ] || return 1
	return 0
}

# purge_data_dir <agent|hub> — irreversibly remove the validated data
# directory. No prompts here; the caller collects confirmations FIRST.
purge_data_dir() {
	_pg_comp="$1"
	_pg_dir=""
	case "$_pg_comp" in
		agent) _pg_dir="${LIB_DIR}/beszel-agent" ;;
		hub) _pg_dir="${LIB_DIR}/beszel-hub" ;;
		*) return 1 ;;
	esac
	validate_purge_path "$_pg_dir" "$_pg_comp" || {
		say_err "Refusing to purge ${_pg_dir}: failed safety validation."
		return 1
	}
	rm -rf "$_pg_dir" || {
		say_err "Could not completely remove ${_pg_dir}."
		return 1
	}
	if [ -e "$_pg_dir" ]; then
		say_err "Could not completely remove ${_pg_dir}."
		return 1
	fi
	say_ok "Data removed: ${_pg_dir}"
	return 0
}

# offer_agent_data_purge — standalone-safe: refuses while the application is
# installed, reports when no data exists, keeps by default, and purges only
# on the exact typed phrase.
offer_agent_data_purge() {
	_oa_dir="${LIB_DIR}/beszel-agent"
	if [ ! -e "$_oa_dir" ]; then
		say_info "No retained Agent data at ${_oa_dir}."
		return 0
	fi
	if [ "$(component_state agent)" != "ABSENT" ]; then
		say_err "Agent application is still installed; uninstall it before purging data."
		return 1
	fi
	if confirm_update "Keep Agent data at ${_oa_dir}?"; then
		say_info "Agent data preserved."
		return 0
	fi
	say_warn "WARNING: ${_oa_dir} holds Agent state. Purging is permanent."
	say_warn "This cannot be undone by the installer."
	_oa_answer=""
	ask_tty "Type exactly: DELETE AGENT DATA" _oa_answer "" || return 1
	if [ "$_oa_answer" != "DELETE AGENT DATA" ]; then
		say_info "Agent data preserved."
		return 0
	fi
	if purge_data_dir "agent"; then
		return 0
	fi
	return 1
}

# offer_hub_data_purge — standalone-safe Hub equivalent with the strong
# typed confirmation.
offer_hub_data_purge() {
	_oh_dir="${LIB_DIR}/beszel-hub"
	if [ ! -e "$_oh_dir" ]; then
		say_info "No retained Hub data at ${_oh_dir}."
		return 0
	fi
	if [ "$(component_state hub)" != "ABSENT" ]; then
		say_err "Hub application is still installed; uninstall it before purging data."
		return 1
	fi
	if confirm_update "Keep Hub data at ${_oh_dir}?"; then
		say_info "Hub data preserved."
		return 0
	fi
	say_warn "WARNING:"
	say_warn "${_oh_dir} contains your Beszel database, accounts,"
	say_warn "configuration, and historical data."
	say_warn "This cannot be undone by the installer."
	_oh_answer=""
	ask_tty "Type exactly: DELETE HUB DATA" _oh_answer "" || return 1
	if [ "$_oh_answer" != "DELETE HUB DATA" ]; then
		say_info "Hub data preserved."
		return 0
	fi
	if purge_data_dir "hub"; then
		return 0
	fi
	return 1
}

# ------------------------------------------------------ uninstall menu ---

# uninstall_agent_flow — plan, confirm (default NO), transactional app
# removal with data preserved, then the optional separate data decision.
uninstall_agent_flow() {
	if [ "$(component_state agent)" = "ABSENT" ]; then
		say_info "Agent application is not installed."
		if offer_agent_data_purge; then
			return 0
		fi
		return 1
	fi
	say_info "Agent will be uninstalled."
	say_info "Remove:"
	say_info "  ${BIN_DIR}/${AGENT_BIN}"
	say_info "  ${LAUNCHD_DIR}/${AGENT_LABEL}.plist"
	say_info "  Agent backup files"
	say_info "  Agent installer release state"
	say_info "Preserve:"
	say_info "  ${LIB_DIR}/beszel-agent"
	if confirm_destructive "Continue with Agent uninstall?"; then
		:
	else
		say_info "Agent uninstall cancelled."
		return 0
	fi
	if transact_uninstall "agent"; then
		say_ok "Agent application uninstalled."
	else
		return 1
	fi
	if [ -e "${LIB_DIR}/beszel-agent" ]; then
		say_info "Agent data preserved at ${LIB_DIR}/beszel-agent."
	fi
	if offer_agent_data_purge; then
		return 0
	fi
	return 1
}

# uninstall_hub_flow — Hub mirror. The database is preserved by default;
# purging it needs the exact typed phrase afterwards.
uninstall_hub_flow() {
	if [ "$(component_state hub)" = "ABSENT" ]; then
		say_info "Hub application is not installed."
		if offer_hub_data_purge; then
			return 0
		fi
		return 1
	fi
	say_info "Hub will be uninstalled."
	say_info "Remove:"
	say_info "  ${BIN_DIR}/${HUB_BIN}"
	say_info "  ${LAUNCHD_DIR}/${HUB_LABEL}.plist"
	say_info "  Hub backup files"
	say_info "  Hub installer release state"
	say_info "Preserve:"
	say_info "  ${LIB_DIR}/beszel-hub"
	if confirm_destructive "Continue with Hub uninstall?"; then
		:
	else
		say_info "Hub uninstall cancelled."
		return 0
	fi
	if transact_uninstall "hub"; then
		say_ok "Hub application uninstalled."
	else
		return 1
	fi
	if [ -e "${LIB_DIR}/beszel-hub" ]; then
		say_info "Hub data preserved at ${LIB_DIR}/beszel-hub."
	fi
	if offer_hub_data_purge; then
		return 0
	fi
	return 1
}

# uninstall_both_flow — Agent first (so the Hub stays up until the Agent is
# gone), then Hub. Independent transactions, one combined confirmation, then
# separate data decisions per component.
uninstall_both_flow() {
	if [ "$(component_state agent)" = "ABSENT" ] || [ "$(component_state hub)" = "ABSENT" ]; then
		say_err "Agent + Hub uninstall needs both applications installed."
		return 1
	fi
	say_info "Agent + Hub will be uninstalled (Agent first, then Hub)."
	say_info "Remove: both binaries, both plists, backup files, installer release state."
	say_info "Preserve:"
	say_info "  ${LIB_DIR}/beszel-agent"
	say_info "  ${LIB_DIR}/beszel-hub"
	if confirm_destructive "Continue with Agent + Hub uninstall?"; then
		:
	else
		say_info "Uninstall cancelled."
		return 0
	fi
	transact_uninstall "agent" || {
		say_err "Agent uninstall failed; Hub was left untouched."
		return 1
	}
	say_ok "Agent application uninstalled."
	transact_uninstall "hub" || {
		say_err "Hub uninstall failed; Agent remains uninstalled."
		return 1
	}
	say_ok "Hub application uninstalled."
	if [ -e "${LIB_DIR}/beszel-agent" ]; then
		say_info "Agent data preserved at ${LIB_DIR}/beszel-agent."
	fi
	if [ -e "${LIB_DIR}/beszel-hub" ]; then
		say_info "Hub data preserved at ${LIB_DIR}/beszel-hub."
	fi
	if offer_agent_data_purge; then
		:
	else
		return 1
	fi
	if offer_hub_data_purge; then
		return 0
	fi
	return 1
}

u_menu_line() {
	# Print one menu line to /dev/tty when available, else stdout.
	if [ -w /dev/tty ]; then
		printf '%s\n' "$1" > /dev/tty
	else
		printf '%s\n' "$1"
	fi
}

show_uninstall_menu() {
	u_menu_line ""
	u_menu_line "Uninstall"
}

# uninstall_menu — adapts to installed applications and retained data.
# Application entries appear only for installed components; purge entries
# appear only for retained data whose application is already absent.
uninstall_menu() {
	while :; do
		_um_u_agent_st=$(component_state "agent")
		_um_u_hub_st=$(component_state "hub")
		_um_u_agent_app=0
		_um_u_hub_app=0
		[ "$_um_u_agent_st" != "ABSENT" ] && _um_u_agent_app=1
		[ "$_um_u_hub_st" != "ABSENT" ] && _um_u_hub_app=1
		_um_u_agent_data=0
		_um_u_hub_data=0
		[ -e "${LIB_DIR}/beszel-agent" ] && _um_u_agent_data=1
		[ -e "${LIB_DIR}/beszel-hub" ] && _um_u_hub_data=1
		if [ "$_um_u_agent_app" = "0" ] && [ "$_um_u_hub_app" = "0" ] && [ "$_um_u_agent_data" = "0" ] && [ "$_um_u_hub_data" = "0" ]; then
			say_info "No Beszel iOS installation or retained data was detected."
			return 0
		fi
		if [ "$_um_u_agent_app" = "0" ] && [ "$_um_u_agent_data" = "1" ]; then
			say_info "Agent application is not installed, but retained Agent data exists."
		fi
		if [ "$_um_u_hub_app" = "0" ] && [ "$_um_u_hub_data" = "1" ]; then
			say_info "Hub application is not installed, but retained Hub data exists."
		fi
		_um_u_n=0
		_um_u_1=""
		_um_u_2=""
		_um_u_3=""
		_um_u_4=""
		_um_u_5=""
		show_uninstall_menu
		if [ "$_um_u_agent_app" = "1" ]; then
			_um_u_n=$((_um_u_n + 1))
			case "$_um_u_n" in
				1) _um_u_1="uninstall-agent" ;;
				2) _um_u_2="uninstall-agent" ;;
				3) _um_u_3="uninstall-agent" ;;
				4) _um_u_4="uninstall-agent" ;;
				5) _um_u_5="uninstall-agent" ;;
			esac
			u_menu_line "  ${_um_u_n}) Uninstall Agent"
		fi
		if [ "$_um_u_hub_app" = "1" ]; then
			_um_u_n=$((_um_u_n + 1))
			case "$_um_u_n" in
				1) _um_u_1="uninstall-hub" ;;
				2) _um_u_2="uninstall-hub" ;;
				3) _um_u_3="uninstall-hub" ;;
				4) _um_u_4="uninstall-hub" ;;
				5) _um_u_5="uninstall-hub" ;;
			esac
			u_menu_line "  ${_um_u_n}) Uninstall Hub"
		fi
		if [ "$_um_u_agent_app" = "1" ] && [ "$_um_u_hub_app" = "1" ]; then
			_um_u_n=$((_um_u_n + 1))
			case "$_um_u_n" in
				1) _um_u_1="uninstall-both" ;;
				2) _um_u_2="uninstall-both" ;;
				3) _um_u_3="uninstall-both" ;;
				4) _um_u_4="uninstall-both" ;;
				5) _um_u_5="uninstall-both" ;;
			esac
			u_menu_line "  ${_um_u_n}) Uninstall Agent + Hub"
		fi
		if [ "$_um_u_agent_app" = "0" ] && [ "$_um_u_agent_data" = "1" ]; then
			_um_u_n=$((_um_u_n + 1))
			case "$_um_u_n" in
				1) _um_u_1="purge-agent" ;;
				2) _um_u_2="purge-agent" ;;
				3) _um_u_3="purge-agent" ;;
				4) _um_u_4="purge-agent" ;;
				5) _um_u_5="purge-agent" ;;
			esac
			u_menu_line "  ${_um_u_n}) Purge retained Agent data"
		fi
		if [ "$_um_u_hub_app" = "0" ] && [ "$_um_u_hub_data" = "1" ]; then
			_um_u_n=$((_um_u_n + 1))
			case "$_um_u_n" in
				1) _um_u_1="purge-hub" ;;
				2) _um_u_2="purge-hub" ;;
				3) _um_u_3="purge-hub" ;;
				4) _um_u_4="purge-hub" ;;
				5) _um_u_5="purge-hub" ;;
			esac
			u_menu_line "  ${_um_u_n}) Purge retained Hub data"
		fi
		_um_u_n=$((_um_u_n + 1))
		_um_u_back="$_um_u_n"
		u_menu_line "  ${_um_u_n}) Back"
		_UMU_CHOICE=""
		ask_tty "Select" _UMU_CHOICE "" || return 1
		_um_u_action=""
		case "$_UMU_CHOICE" in
			1) _um_u_action="$_um_u_1" ;;
			2) _um_u_action="$_um_u_2" ;;
			3) _um_u_action="$_um_u_3" ;;
			4) _um_u_action="$_um_u_4" ;;
			5) _um_u_action="$_um_u_5" ;;
		esac
		if [ -z "$_um_u_action" ]; then
			if [ "$_UMU_CHOICE" = "$_um_u_back" ]; then
				say_info "Back selected."
				return 0
			fi
			say_err "Invalid selection '${_UMU_CHOICE}'."
			return 1
		fi
		case "$_um_u_action" in
			uninstall-agent) uninstall_agent_flow || say_warn "Agent uninstall did not complete." ;;
			uninstall-hub) uninstall_hub_flow || say_warn "Hub uninstall did not complete." ;;
			uninstall-both) uninstall_both_flow || say_warn "Agent + Hub uninstall did not complete." ;;
			purge-agent) offer_agent_data_purge || say_warn "Agent data purge did not complete." ;;
			purge-hub) offer_hub_data_purge || say_warn "Hub data purge did not complete." ;;
		esac
	done
}

show_update_menu() {
	# $1 = agent|hub|both.
	_su_mode="$1"
	if [ ! -r /dev/tty ] || [ ! -w /dev/tty ]; then
		case "$_su_mode" in
			agent) say_info "Beszel Agent detected." ;;
			hub) say_info "Beszel Hub detected." ;;
		esac
		return 0
	fi
	case "$_su_mode" in
		agent)
			cat > /dev/tty << EOF

Beszel Agent detected.
  1) Update Agent
  2) Back
EOF
			;;
		hub)
			cat > /dev/tty << EOF

Beszel Hub detected.
  1) Update Hub
  2) Back
EOF
			;;
		both)
			cat > /dev/tty << EOF

Beszel Agent + Hub detected.
  1) Update Agent
  2) Update Hub
  3) Update Agent + Hub
  4) Back
EOF
			;;
	esac
	return 0
}

# update_menu — inspect installed components, then dispatch. Downloads
# nothing except the tiny Latest/SHA256SUMS metadata; component binaries are
# fetched only when a chosen component actually needs an update.
update_menu() {
	_um_agent=$(component_status "agent")
	_um_hub=$(component_status "hub")
	if [ "$_um_agent" = "missing" ] && [ "$_um_hub" = "missing" ]; then
		say_info "No Beszel iOS installation was detected."
		return 0
	fi
	if [ "$_um_agent" = "incomplete" ]; then
		say_err "Agent installation is incomplete."
		say_err "Repair support will be added separately."
	fi
	if [ "$_um_hub" = "incomplete" ]; then
		say_err "Hub installation is incomplete."
		say_err "Repair support will be added separately."
	fi
	if [ "$_um_agent" != "complete" ] && [ "$_um_hub" != "complete" ]; then
		return 0
	fi
	ensure_pinned_release || return 1
	if [ "$_um_agent" = "complete" ] && [ "$_um_hub" = "complete" ]; then
		show_update_menu "both"
		_UM_CHOICE=""
		ask_tty "Select" _UM_CHOICE "" || return 1
		case "$_UM_CHOICE" in
			1) update_agent_flow "$LATEST_TAG" ;;
			2) update_hub_flow "$LATEST_TAG" ;;
			3) update_both_flow "$LATEST_TAG" ;;
			4) say_info "Back selected. Nothing was changed." ;;
			*)
				say_err "Invalid selection '${_UM_CHOICE}'."
				return 1
				;;
		esac
	elif [ "$_um_agent" = "complete" ]; then
		show_update_menu "agent"
		_UM_CHOICE=""
		ask_tty "Select" _UM_CHOICE "" || return 1
		case "$_UM_CHOICE" in
			1) update_agent_flow "$LATEST_TAG" ;;
			2) say_info "Back selected. Nothing was changed." ;;
			*)
				say_err "Invalid selection '${_UM_CHOICE}'."
				return 1
				;;
		esac
	else
		show_update_menu "hub"
		_UM_CHOICE=""
		ask_tty "Select" _UM_CHOICE "" || return 1
		case "$_UM_CHOICE" in
			1) update_hub_flow "$LATEST_TAG" ;;
			2) say_info "Back selected. Nothing was changed." ;;
			*)
				say_err "Invalid selection '${_UM_CHOICE}'."
				return 1
				;;
		esac
	fi
}

# ------------------------------------------------------------ repair menu ---

show_repair_menu() {
	# $1 = agent|hub|both.
	_sr_mode="$1"
	if [ ! -r /dev/tty ] || [ ! -w /dev/tty ]; then
		case "$_sr_mode" in
			agent) say_info "Beszel Agent detected." ;;
			hub) say_info "Beszel Hub detected." ;;
		esac
		return 0
	fi
	case "$_sr_mode" in
		agent)
			cat > /dev/tty << EOF

Repair / Reconfigure (Agent)
  1) Diagnose Agent
  2) Repair Agent
  3) Reconfigure Agent
  4) Back
EOF
			;;
		hub)
			cat > /dev/tty << EOF

Repair / Reconfigure (Hub)
  1) Diagnose Hub
  2) Repair Hub
  3) Reconfigure Hub
  4) Back
EOF
			;;
		both)
			cat > /dev/tty << EOF

Repair / Reconfigure (Agent + Hub)
  1) Diagnose Agent
  2) Repair Agent
  3) Reconfigure Agent
  4) Diagnose Hub
  5) Repair Hub
  6) Reconfigure Hub
  7) Back
EOF
			;;
	esac
	return 0
}

# repair_menu — diagnostics, repair and reconfiguration for whichever
# components exist. Loops until Back (or input failure); actions report
# their own outcome and return here.
repair_menu() {
	while :; do
		_rm_agent_st=$(component_state "agent")
		_rm_hub_st=$(component_state "hub")
		if [ "$_rm_agent_st" = "ABSENT" ] && [ "$_rm_hub_st" = "ABSENT" ]; then
			say_info "No Beszel iOS installation was detected."
			return 0
		fi
		_rm_agent_here=0
		_rm_hub_here=0
		[ "$_rm_agent_st" != "ABSENT" ] && _rm_agent_here=1
		[ "$_rm_hub_st" != "ABSENT" ] && _rm_hub_here=1
		if [ "$_rm_agent_here" = "1" ] && [ "$_rm_hub_here" = "1" ]; then
			show_repair_menu "both"
			_RM_CHOICE=""
			ask_tty "Select" _RM_CHOICE "" || return 1
			case "$_RM_CHOICE" in
				1) diagnose_agent ;;
				2) repair_agent_flow || say_warn "Agent repair did not complete." ;;
				3) reconfigure_agent_flow || say_warn "Agent reconfiguration did not complete." ;;
				4) diagnose_hub ;;
				5) repair_hub_flow || say_warn "Hub repair did not complete." ;;
				6) reconfigure_hub_flow || say_warn "Hub reconfiguration did not complete." ;;
				7)
					say_info "Back selected."
					return 0
					;;
				*)
					say_err "Invalid selection '${_RM_CHOICE}'."
					return 1
					;;
			esac
		elif [ "$_rm_agent_here" = "1" ]; then
			show_repair_menu "agent"
			_RM_CHOICE=""
			ask_tty "Select" _RM_CHOICE "" || return 1
			case "$_RM_CHOICE" in
				1) diagnose_agent ;;
				2) repair_agent_flow || say_warn "Agent repair did not complete." ;;
				3) reconfigure_agent_flow || say_warn "Agent reconfiguration did not complete." ;;
				4)
					say_info "Back selected."
					return 0
					;;
				*)
					say_err "Invalid selection '${_RM_CHOICE}'."
					return 1
					;;
			esac
		else
			show_repair_menu "hub"
			_RM_CHOICE=""
			ask_tty "Select" _RM_CHOICE "" || return 1
			case "$_RM_CHOICE" in
				1) diagnose_hub ;;
				2) repair_hub_flow || say_warn "Hub repair did not complete." ;;
				3) reconfigure_hub_flow || say_warn "Hub reconfiguration did not complete." ;;
				4)
					say_info "Back selected."
					return 0
					;;
				*)
					say_err "Invalid selection '${_RM_CHOICE}'."
					return 1
					;;
			esac
		fi
	done
}

print_menu() {
	# Unquoted heredoc is safe here: this text contains no $ or backticks.
	if [ -w /dev/tty ]; then
		cat > /dev/tty << EOF

Beszel iOS Installer (v${INSTALLER_VERSION})
Unofficial community port of Beszel

  1) Install Agent
  2) Install Hub
  3) Install Agent + Hub
  4) Update
  5) Repair / Reconfigure
  6) Uninstall
  7) Exit
EOF
	else
		cat << EOF

Beszel iOS Installer (v${INSTALLER_VERSION})
Unofficial community port of Beszel

  1) Install Agent
  2) Install Hub
  3) Install Agent + Hub
  4) Update
  5) Repair / Reconfigure
  6) Uninstall
  7) Exit
EOF
	fi
}

main() {
	say_info "Beszel iOS Installer v${INSTALLER_VERSION} — unofficial community port of Beszel."
	check_root
	check_device
	check_layout
	check_launchctl
	setup_work_dir
	while :; do
		print_menu
		CHOICE=""
		ask_tty "Select" CHOICE "" || exit 1
		case "$CHOICE" in
			1)
				if check_deps; then
					flow_agent || say_warn "Agent install did not complete."
				fi
				;;
			2)
				if check_deps; then
					flow_hub || say_warn "Hub install did not complete."
				fi
				;;
			3)
				if check_deps; then
					flow_both || say_warn "Agent + Hub install did not complete."
				fi
				;;
			4)
				if check_deps; then
					update_menu || say_warn "Update did not complete."
				fi
				;;
			5)
				if check_deps; then
					repair_menu || say_warn "Repair / Reconfigure did not complete."
				fi
				;;
			6) uninstall_menu || say_warn "Uninstall did not complete." ;;
			7)
				say_info "Exit selected."
				break
				;;
			*)
				say_err "Invalid selection '${CHOICE}': choose 1-7."
				exit 1
				;;
		esac
	done
}

if [ "${BESZEL_INSTALL_LIB_ONLY:-0}" != "1" ]; then
	main "$@"
fi
