# Beszel iOS Device Validation

Evidence-oriented record of validation performed for the Beszel iOS community
port. This file is updated as validation runs complete.

## Status summary

| Area | Result |
| --- | --- |
| Automated installer test suite | **Passed** (546 / 0) |
| `sh -n` syntax check (install.sh + tests) | **Passed** |
| `shellcheck -s sh` (install.sh + tests) | **Passed** |
| `git diff --check` | **Passed** |
| Go test suite | Not run in this environment (pre-existing `dist/` embed asset absence; unrelated to installer) |
| Real-device end-to-end matrix | **Not performed** — see below |

## Real-device validation

**Result: NOT PERFORMED.**

The execution environment for this validation run had no authorized SSH
access, no SSH agent, and no known device host entry for the target iPad.
Real-device validation is therefore blocked for every on-device scenario.

Concrete missing validation (PROMPT_8_BLOCKED):

- Device baseline snapshot (uname, machine, iOS version, disk, service PIDs,
  ports, Hub health, data sizes)
- Raw-script fetch and `/bin/sh -n` on the iPad itself
- Diagnostics run on device (Agent + Hub, no mutation)
- No-op reconfigure (Agent + Hub)
- Repair / no-op restart checks
- Current-release Update no-op behavior on device
- Uninstall cancellation behavior on device
- Purge refusal (wrong confirmation phrase) on device
- Controlled Agent uninstall (keep data) / reinstall / reconnect / battery
- Controlled Hub uninstall (keep data) / reinstall reusing existing DB /
  existing account + config + history preservation / Agent reconnect
- Reboot persistence (LaunchDaemon reload, jailbreak reactivation behavior,
  Hub health, Agent reconnect)
- LAN / Tailscale access verification
- Battery telemetry (`ioreg -r -c AppleARMPMUCharger`) and Beszel exposure
- Runtime resource observation (RSS, memory, disk)

No on-device mutation, backup, or data change was performed.

## Validation environment

- Date: 2026-09-17
- Installer version under test: `0.4.0`
- Binary release under test: `v0.19.0-ios.1`
- Local raw-script SHA-256: `5502258c3289acd344f1128fde596fe963be84ca79a673967f7b6d7974e5060e`
- Live raw-script SHA-256 (fetched during validation): identical
- Branch: `ios`

## Static hardening review (no device required)

A fresh security/correctness review of `install.sh` was performed covering:

- `set -e` interactions and rollback return codes
- Trap arming / disarming and re-entrancy (`update_begin` / `update_end`,
  `cleanup_work_dir`)
- INT / TERM / HUP handling and exit code 130 semantics
- `/dev/tty` reads (`read_tty`, `ask_tty`, `wait_for_enter`) and failure
  behavior when no terminal is present
- Path quoting throughout
- Symlink checks on live binaries and plists before mutation
- Exact-path purge guards (`validate_purge_path`) and no wildcard destructive
  deletes
- No `rm -rf` outside `WORK_DIR` and guarded purge paths
- No recursive `chown` / `chmod`
- Agent KEY redaction (value never printed, logged, or stored in state)
- Plist parser ambiguity (one-per-line `<key>`/`<string>`; fails closed on
  duplicates or malformed associations)
- XML escaping (`xml_escape`) and decoding (`xml_decode`)
- State parser injection resistance (parsed key by key with grep / parameter
  expansion; never sourced)
- Release-tag injection resistance (`tag_from_latest_url` allowlist,
  `valid_release_tag` regex)
- Pinned-release behavior (all assets for a transaction come from one tag)
- Checksum enforcement (download -> `validate_sums_file` -> per-asset
  `sums_hash_for` -> `verify_file` before any install)
- `ldid` signing before execution
- Stale `.new` / `.bak` / `.rollback` / `.restore` handling
- Uninstall state cleanup (exact paths, `rmdir` only when empty)
- Offline uninstall (no network or `ldid` required)
- Reinstall with retained data (data directories never touched)

No concrete defect was found. The installer version was therefore kept below
`1.0.0`.

## Test matrix

Legend: **Pass** = observed and passing. **Not tested** = not performed.
**Blocked** = blocked by missing real-device access.

| Scenario | Result |
| --- | --- |
| Automated regression suite (546 tests) | Pass |
| `sh -n` install.sh / tests | Pass |
| `shellcheck -s sh` install.sh / tests | Pass |
| `git diff --check` | Pass |
| Raw-script hash match vs repository | Pass |
| Device baseline snapshot | Not tested |
| Raw-script fetch + `/bin/sh -n` on iPad | Not tested |
| Diagnostics (Agent + Hub, no mutation) | Not tested |
| No-op reconfigure (Agent) | Not tested |
| No-op reconfigure (Hub) | Not tested |
| Repair / no-op restart | Not tested |
| Current-release Update no-op | Not tested |
| Uninstall cancellation | Not tested |
| Purge refusal (wrong phrase) | Not tested |
| Agent uninstall (keep data) | Not tested |
| Agent reinstall + reconnect + battery | Not tested |
| Hub uninstall (keep data) | Not tested |
| Hub reinstall reusing existing DB | Not tested |
| Hub account / config / history preservation | Not tested |
| Reboot persistence | Not tested |
| LAN / Tailscale access | Not tested |
| Battery telemetry | Not tested |
| Resource observations | Not tested |

## Known limitations

- Real-device end-to-end validation has not been performed from this
  environment. The installer is believed correct by static review and the
  automated suite, but the 1.0.0 gate requires the on-device matrix above.
- The Go test suite could not be run here because the `dist/` frontend embed
  asset is absent from the checkout; this is an environment issue, not a
  code defect, and no Go source was modified during this prompt.
- Compatibility is not claimed for A8+ devices, arm64e, newer iOS versions, or
  rootless jailbreaks. They remain untested.