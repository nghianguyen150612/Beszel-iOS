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
# Scope: fresh installs of Agent and/or Hub, plus safe transactional updates
# with signed-binary staging, binary backup and automatic rollback.
# Repair, reconfigure and uninstall are NOT implemented here.
#
# Version model: installed binaries are ldid-signed on device, so the hash of
# an installed binary differs from its unsigned release asset. Update
# detection therefore uses the installer-managed state file
# (/var/lib/beszel-ios/install-state), never a hash comparison of the
# installed binary against SHA256SUMS. SHA256SUMS only validates freshly
# downloaded release assets before installation.

set -eu
umask 022

INSTALLER_VERSION="0.2.0"
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
	_sw_sha_norm=$(printf '%s' "$_sw_sha" | tr 'A-F' 'a-f')
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
			_sw_agent_sha="$_sw_sha_norm"
			;;
		hub)
			_sw_hub_rel="$_sw_rel"
			_sw_hub_sha="$_sw_sha_norm"
			;;
	esac
	_sw_dir="${LIB_DIR}/${STATE_SUBDIR}"
	mkdir -p "$_sw_dir" || return 1
	fix_path_owner "$_sw_dir" || true
	chmod 755 "$_sw_dir" || return 1
	_sw_tmp="${_sw_dir}/${STATE_FILE_NAME}.new.$$"
	{
		printf 'STATE_VERSION=%s\n' "$STATE_VERSION"
		printf 'AGENT_RELEASE=%s\n' "$_sw_agent_rel"
		printf 'AGENT_ASSET_SHA256=%s\n' "$_sw_agent_sha"
		printf 'HUB_RELEASE=%s\n' "$_sw_hub_rel"
		printf 'HUB_ASSET_SHA256=%s\n' "$_sw_hub_sha"
	} > "$_sw_tmp" || {
		rm -f "$_sw_tmp"
		return 1
	}
	[ -s "$_sw_tmp" ] || {
		rm -f "$_sw_tmp"
		return 1
	}
	fix_path_owner "$_sw_tmp" || true
	chmod 644 "$_sw_tmp" || {
		rm -f "$_sw_tmp"
		return 1
	}
	mv -f "$_sw_tmp" "$(state_path)" || {
		rm -f "$_sw_tmp"
		return 1
	}
	fix_path_owner "$(state_path)" || true
	return 0
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
	_cd_arch=$(uname -m)
	[ "$_cd_os" = "Darwin" ] || die "Refusing to install: uname -s is '${_cd_os}', not Darwin. This installer targets jailbroken iOS only."
	[ "$_cd_arch" = "arm64" ] || die "Refusing to install: uname -m is '${_cd_arch}', not arm64. This installer targets iOS arm64 only."
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

check_deps() {
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
		chown root:wheel "$_ed_dir" || die "Cannot set ownership on ${_ed_dir}."
		chmod 755 "$_ed_dir" || die "Cannot set permissions on ${_ed_dir}."
	fi
}

check_plist() {
	_cp_file="$1"
	if command -v plutil > /dev/null 2>&1; then
		plutil --lint "$_cp_file" || die "Generated plist failed validation: ${_cp_file}"
		say_ok "Plist valid (plutil): ${_cp_file}"
	elif command -v xmllint > /dev/null 2>&1; then
		xmllint --noout "$_cp_file" || die "Generated plist failed validation: ${_cp_file}"
		say_ok "Plist valid (xmllint): ${_cp_file}"
	elif command -v python3 > /dev/null 2>&1; then
		python3 -c 'import sys,xml.dom.minidom; xml.dom.minidom.parse(sys.argv[1])' "$_cp_file" || die "Generated plist failed validation: ${_cp_file}"
		say_ok "Plist valid (xml parser): ${_cp_file}"
	else
		say_warn "No plist checker available; skipping validation for ${_cp_file}."
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
hub_port_from_plist() {
	_hp_file="${1:-}"
	[ -f "$_hp_file" ] || return 1
	_hp_port=$(sed -n 's/.*0\.0\.0\.0:\([0-9][0-9]*\).*/\1/p' "$_hp_file" 2> /dev/null | head -n 1 || true)
	[ -n "$_hp_port" ] || return 1
	valid_port "$_hp_port" || return 1
	printf '%s' "$_hp_port"
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

# update_begin <agent|hub> — arm the transaction traps (no rollback needed
# yet); the caller sets _UPDATE_NEED_ROLLBACK=1 at the point of no return.
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
  5) Exit
EOF
	else
		cat << EOF

Beszel iOS Installer (v${INSTALLER_VERSION})
Unofficial community port of Beszel

  1) Install Agent
  2) Install Hub
  3) Install Agent + Hub
  4) Update
  5) Exit
EOF
	fi
}

main() {
	say_info "Beszel iOS Installer v${INSTALLER_VERSION} — unofficial community port of Beszel."
	check_root
	check_device
	check_layout
	check_deps
	setup_work_dir
	print_menu
	CHOICE=""
	ask_tty "Select" CHOICE "" || exit 1
	case "$CHOICE" in
		1) flow_agent ;;
		2) flow_hub ;;
		3) flow_both ;;
		4) update_menu ;;
		5) say_info "Exit selected. Nothing was changed." ;;
		*)
			say_err "Invalid selection '${CHOICE}': choose 1-5."
			exit 1
			;;
	esac
}

if [ "${BESZEL_INSTALL_LIB_ONLY:-0}" != "1" ]; then
	main "$@"
fi
