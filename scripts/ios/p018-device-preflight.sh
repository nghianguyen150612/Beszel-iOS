#!/bin/sh
# P018 read-only, redacted iPad preflight. No service, disk, network or config mutation.
# Run directly on the reference iPad from an already working SSH/local terminal.
set -u
umask 077

say() { printf '%s\n' "$*"; }
section() { printf '\n=== %s ===\n' "$1"; }
value() {
  _p018_label=$1
  shift
  _p018_result=$("$@" 2>/dev/null) || _p018_result='UNAVAILABLE'
  printf '%s: %s\n' "$_p018_label" "$_p018_result"
}
hashfile() {
  _p018_file=$1
  if [ -L "$_p018_file" ]; then
    say "REFUSE HASH (symlink): $_p018_file"
  elif [ ! -f "$_p018_file" ]; then
    say "ABSENT: $_p018_file"
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$_p018_file"
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$_p018_file"
  else
    say "NO SHA256 UTILITY: $_p018_file"
  fi
}

case "${1:-}" in
  '') ;;
  *) say 'Usage: sh p018-device-preflight.sh (no arguments)'; exit 2 ;;
esac

say "P018 PRE-FLIGHT ONLY. Read-only. Never treat this as on-device v0.21 PASS."
say "Capture output securely; remove usernames, IP addresses and other identifiers before sharing."

section 'Identity / OS'
value 'Kernel' uname -srm
if command -v sysctl >/dev/null 2>&1; then
  value 'Hardware model' sysctl -n hw.machine
  value 'Architecture' sysctl -n hw.cputype
fi
if command -v plutil >/dev/null 2>&1; then
  value 'iOS version' plutil -key ProductVersion /System/Library/CoreServices/SystemVersion.plist
fi
value 'Effective uid' id -u

section 'Rootful preflight / utility presence'
for _p018_cmd in launchctl ldid tar ssh scp sha256sum shasum; do
  if command -v "$_p018_cmd" >/dev/null 2>&1; then
    say "AVAILABLE: $_p018_cmd"
  else
    say "MISSING: $_p018_cmd"
  fi
done
for _p018_path in /usr/local/bin /var/lib/beszel-agent /var/lib/beszel-hub /var/lib/beszel-ios /Library/LaunchDaemons; do
  if [ -L "$_p018_path" ]; then
    say "SYMLINK PATH — INSPECT: $_p018_path"
  elif [ -d "$_p018_path" ]; then
    say "DIRECTORY: $_p018_path"
  else
    say "ABSENT: $_p018_path"
  fi
done

section 'Installed versions and hashes (no key material)'
for _p018_file in /usr/local/bin/beszel-agent /usr/local/bin/beszel-hub /usr/local/bin/beszel-ios /var/lib/beszel-ios/manager.sh; do
  hashfile "$_p018_file"
done

section 'Service registration (labels only)'
if command -v launchctl >/dev/null 2>&1; then
  _p018_services=$(launchctl list 2>/dev/null | grep -E 'dev[.]beszel[.](agent|hub)' || true)
  if [ -n "$_p018_services" ]; then
    printf '%s\n' "$_p018_services"
  else
    say 'No matching service label, or launchctl list unavailable.'
  fi
else
  say 'launchctl unavailable'
fi

section 'Filesystem capacity'
if command -v df >/dev/null 2>&1; then
  df -k /var 2>/dev/null || say 'Could not read /var capacity.'
fi

section 'Summary'
say 'No file was written, service restarted, network contacted, or database touched.'
say 'Do not run install/update/reboot until a verified cold backup and physical recovery plan are approved.'
say 'P018 Gate F remains PENDING until real device measurements and reboot checks are recorded.'
