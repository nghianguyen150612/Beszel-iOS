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
# Scope: fresh installs of Agent and/or Hub only. Update, repair,
# reconfigure, rollback and uninstall are NOT implemented here.

set -eu
umask 022

INSTALLER_VERSION="0.1.0"
USER_AGENT="beszel-ios-installer/${INSTALLER_VERSION}"

RELEASE_BASE="https://github.com/nghianguyen150612/beszel-ios/releases/latest/download"
SUMS_URL="${RELEASE_BASE}/SHA256SUMS"

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
	say_info "Downloading SHA256SUMS from the Latest release..."
	download "$SUMS_URL" "${WORK_DIR}/${SUMS_ASSET}"
	validate_sums_file "${WORK_DIR}/${SUMS_ASSET}" || die "SHA256SUMS failed validation (expected exactly one entry each for ${AGENT_ASSET} and ${HUB_ASSET}). Aborting before trusting any binary."
	say_ok "SHA256SUMS structure valid."
}

fetch_and_verify_binary() {
	# $1 = asset name (also the release URL leaf); prints the verified path
	# on stdout. Status messages go to stderr so command substitution
	# captures only the path.
	_fb_asset="$1"
	say_info "Downloading ${_fb_asset}..." >&2
	download "${RELEASE_BASE}/${_fb_asset}" "${WORK_DIR}/${_fb_asset}"
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
	launchctl load -w "$2" || die "launchctl failed to load $1."
	if launchctl list 2> /dev/null | grep -q "$1"; then
		say_ok "Service running: $1"
	else
		say_warn "Loaded $1 but could not confirm it in 'launchctl list'."
	fi
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
		say_warn "Update/reconfigure support is not implemented by this installer yet; leaving it untouched."
		return 0
	fi
	_agent_src=$(fetch_and_verify_binary "$AGENT_ASSET") || exit 1
	if install_binary "$_agent_src" "$AGENT_BIN"; then
		say_ok "Installed ${BIN_DIR}/${AGENT_BIN} (signed with ldid)."
	else
		_ib_rc=$?
		if [ "$_ib_rc" = "2" ]; then
			say_warn "Existing Beszel Agent installation detected."
			say_warn "Update/reconfigure support is not implemented by this installer yet; leaving it untouched."
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
		say_warn "Update/reconfigure support is not implemented by this installer yet; leaving it untouched."
		return 0
	fi
	_hub_src=$(fetch_and_verify_binary "$HUB_ASSET") || exit 1
	if install_binary "$_hub_src" "$HUB_BIN"; then
		say_ok "Installed ${BIN_DIR}/${HUB_BIN} (signed with ldid)."
	else
		_ib_rc=$?
		if [ "$_ib_rc" = "2" ]; then
			say_warn "Existing Beszel Hub installation detected."
			say_warn "Update/reconfigure support is not implemented by this installer yet; leaving it untouched."
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
	say_ok "Hub installed: binary ${BIN_DIR}/${HUB_BIN}, data ${LIB_DIR}/beszel-hub, plist ${LAUNCHD_DIR}/${HUB_LABEL}.plist, port ${HUB_PORT}."
	return 0
}

# ------------------------------------------------------------------- flows ---

print_summary() {
	# $1 = mode label.
	say_info "---- Beszel iOS install summary ($1) ----"
	if agent_installed; then
		say_info "Agent: ${BIN_DIR}/${AGENT_BIN} | data ${LIB_DIR}/beszel-agent | plist ${LAUNCHD_DIR}/${AGENT_LABEL}.plist | logs ${LOG_DIR}/beszel-agent.log"
	fi
	if hub_installed; then
		say_info "Hub:   ${BIN_DIR}/${HUB_BIN} | data ${LIB_DIR}/beszel-hub | plist ${LAUNCHD_DIR}/${HUB_LABEL}.plist | logs ${LOG_DIR}/beszel-hub.log"
	fi
}

flow_agent() {
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

print_menu() {
	# Unquoted heredoc is safe here: this text contains no $ or backticks.
	if [ -w /dev/tty ]; then
		cat > /dev/tty << EOF

Beszel iOS Installer (v${INSTALLER_VERSION})
Unofficial community port of Beszel

  1) Install Agent
  2) Install Hub
  3) Install Agent + Hub
  4) Exit
EOF
	else
		cat << EOF

Beszel iOS Installer (v${INSTALLER_VERSION})
Unofficial community port of Beszel

  1) Install Agent
  2) Install Hub
  3) Install Agent + Hub
  4) Exit
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
	fetch_sums
	print_menu
	CHOICE=""
	ask_tty "Select" CHOICE "" || exit 1
	case "$CHOICE" in
		1) flow_agent ;;
		2) flow_hub ;;
		3) flow_both ;;
		4) say_info "Exit selected. Nothing was changed." ;;
		*)
			say_err "Invalid selection '${CHOICE}': choose 1-4."
			exit 1
			;;
	esac
}

if [ "${BESZEL_INSTALL_LIB_ONLY:-0}" != "1" ]; then
	main "$@"
fi
