# Beszel-iOS Device Validation

Evidence-oriented record of validation performed for the Beszel-iOS community
port. This file is updated as validation runs complete.

## Status summary

| Area | Result |
| --- | --- |
| Automated installer test suite | **Passed** (551 / 0; printed summary agrees with PASS count) |
| `sh -n` syntax check (install.sh + tests) | **Passed** |
| `shellcheck -s sh install.sh` | **Passed** (clean) |
| `shellcheck -s sh tests/install-sh-test.sh` | Pre-existing SC2329 info-only notes on test-harness mock helpers; no installer finding |
| `git diff --check` | **Passed** |
| Go test suite | Not run in this environment (pre-existing `dist/` embed asset absence; unrelated to installer; no Go source changed) |
| Real-device end-to-end matrix (incl. reboot validation) | **Passed** — see below |

## Real-device validation

**Result: PASSED.**

The full on-device matrix was executed against the reference device,
finishing with a real reboot validation. The installer
version under test on device was `0.4.0`; with all checks passing it is
promoted to `1.0.0` with no binary change (binaries remain `v0.19.0-ios.1`).

No purge confirmation phrases (`DELETE AGENT DATA` / `DELETE HUB DATA`) were
typed at any point. The release-candidate safety backup at
`/var/backups/beszel-ios-rc-20260917-071749` was retained and never deleted
or overwritten.

### Validation environment

- Date: 2026-09-17 (UTC); device local time 2026-09-18 +07 during post-reboot checks
- Branch: `iOS`, HEAD `5f413619`
- Installer version under test on device: `0.4.0` (promoted to `1.0.0` after validation)
- Binary release: `v0.19.0-ios.1` (unchanged; no Go/binary source changed, so no `v0.19.0-ios.2`)
- Reference device (only tested compatibility):
  - iPad mini 2 (iPad4,4 / A1489), Apple A7, arm64
  - iOS 12.5.7 (Build 16H81)
  - semi-untethered Amethyst / Procursus environment
- Stock/non-jailbroken iOS compatibility is not claimed.

### Pre-reboot matrix

Validated before the reboot:

- diagnostics Agent, diagnostics Hub
- no Agent key leak (key value never printed/logged/stored)
- Agent no-op reconfigure, Hub no-op reconfigure
- healthy Agent repair, healthy Hub repair
- Agent uninstall cancellation, Hub uninstall cancellation
- controlled Agent uninstall with `/var/lib/beszel-agent` preserved
- Agent reinstall with plist/binary/PID/listen restored, reconnect, metrics restored, battery telemetry restored
- controlled Hub uninstall with `/var/lib/beszel-hub` and Hub DB preserved
- Hub reinstall with `/api/health` 200, existing account / systems / config / historical stats preserved
- Agent reconnected with stats resuming and increasing after reinstall

### Post-reboot validation (fresh evidence 2026-09-17/18)

The human operator rebooted the real iPad, allowed a normal iOS boot, and
manually reactivated the existing semi-untethered Amethyst jailbreak, then
confirmed SSH key authentication works again. Jailbreak reactivation after a
full reboot is expected on a semi-untethered jailbreak and is not classified
as a Beszel failure. No reboot was repeated from this session.

Device identity (STEP 33):

- `id -u` = `1002` (non-root SSH user)
- `uname -a` = `Darwin Nghias-iPad 18.7.0 Darwin Kernel Version 18.7.0 ... RELEASE_ARM64_S5L8960X iPad4,4 arm Darwin`
- `hw.machine` = `iPad4,4`
- `hw.cputype` = `16777228` (CPU_TYPE_ARM64)
- `kern.osrelease` = `18.7.0`, `ProductVersion` 12.5.7, `BuildVersion` 16H81

LaunchDaemon persistence (STEP 34):

- `launchctl list dev.beszel.agent`: loaded, `PID = 203`, `LastExitStatus = 0`
- `launchctl list dev.beszel.hub`: loaded, `PID = 198`, `LastExitStatus = 0`
- Binaries and plists present:
  - `/usr/local/bin/beszel-agent` (9251952 bytes), `/usr/local/bin/beszel-hub` (30800208 bytes)
  - `/Library/LaunchDaemons/dev.beszel.agent.plist` (1022 bytes), `/Library/LaunchDaemons/dev.beszel.hub.plist` (938 bytes)

Ports (STEP 35):

- `*.45876 LISTEN` plus an ESTABLISHED `127.0.0.1:45876 <-> 127.0.0.1:49179` Agent/Hub pair
- `*.8090 LISTEN` plus ESTABLISHED Hub connections

Hub health (STEP 36):

- `curl -fsS http://127.0.0.1:8090/api/health` = `{"message":"API is healthy.","code":200,"data":{}}`, exit 0

Agent reconnect / fresh stats (STEP 37):

- `systems` row: `iPadServer|127.0.0.1|45876|up` (host/port unchanged)
- Second system `MyLaptop|100.121.124.78|45876|up` also present
- Fresh per-minute iPad stats arriving post-reboot with advancing timestamps:
  - `2026-09-17 23:33:26.141Z`, then `23:34:26.083Z`, then `23:35:26.083Z` (device clock 23:35:34 UTC at final sample)
- No private keys exposed during verification (Hub key listed by path only; Agent key value never printed).

Data / account / config persistence (STEP 38):

- `/var/lib/beszel-agent` persists (`fingerprint`, 48 bytes, dated Sep 13)
- `/var/lib/beszel-hub` persists (`data.db` born Sep 13, `data.db-wal` freshly written post-reboot; `auxiliary.db`, `id_ed25519` present, key contents never dumped)
- `/var/lib/beszel-ios/install-state` persists with `AGENT_RELEASE=v0.19.0-ios.1` and `HUB_RELEASE=v0.19.0-ios.1`
- Hub DB not reset: `users` = 1 (created 2026-09-13), `_superusers` = 1, `systems` = 2, `user_settings` = 1, `system_details` = 2
- Historical statistics from before Hub uninstall/reinstall/reboot remain (earliest iPad record `2026-09-13 07:00:09.063Z`; iPad `1m` count 106) and new statistics continue after reboot.

Battery telemetry (STEP 39):

- Source `/usr/sbin/ioreg -r -c AppleARMPMUCharger -l` (no `-a`) works after reboot:
  - `CurrentCapacity = 42`, `MaxCapacity = 100`, `AppleRawMaxCapacity = 3989`, `AppleRawCurrentCapacity = 1654`
  - `ExternalConnected = Yes`, `IsCharging = Yes`, `FullyCharged = No`, `BatteryInstalled = Yes`
  - (The operator's immediate post-reboot reading was `CurrentCapacity=40`, `ExternalConnected=No`, `IsCharging=No`; the device was later charging during this session. Both readings prove the source works; Beszel tracked the rise 40 -> 41 -> 43.)
- Beszel receives battery telemetry after reboot: latest iPad `1m` records carry `bat:[41,3]` / `bats:{"Primary":41}`, advancing to `{"Primary":43}` on the next minute.

### Resource observation (observation only, no tuning)

- Agent binary 9251952 bytes; Hub binary 30800208 bytes
- `/var/lib/beszel-hub` 6.2M; `/var/lib/beszel-agent` 4.0K
- `hw.memsize` / `hw.physmem` = 1019215872 (~973 MB); `vm_stat` snapshot: 1744 free pages, 95814 active, 93262 inactive, 2523 speculative, 38863 wired (4096-byte pages)
- Per-process RSS could not be observed from the `uid=1002` SSH session (`ps` reports 0 for the root-owned daemons and passwordless `sudo` is unavailable); recorded here as unobservable rather than estimated.

## Static hardening review (no device required)

A security/correctness review of `install.sh` was performed covering:

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

No concrete defect was found. With the on-device matrix now passing, the
installer is promoted to `1.0.0`.

## Test matrix

Legend: **Pass** = observed and passing.

| Scenario | Result |
| --- | --- |
| Automated regression suite (551 tests) | Pass |
| `sh -n` install.sh / tests | Pass |
| `shellcheck -s sh` install.sh | Pass |
| `git diff --check` | Pass |
| Raw-script hash match vs repository | Pass (verified after push; see below) |
| Device baseline snapshot | Pass |
| Diagnostics (Agent + Hub, no mutation) | Pass |
| No-op reconfigure (Agent) | Pass |
| No-op reconfigure (Hub) | Pass |
| Repair / no-op restart | Pass |
| Uninstall cancellation (Agent + Hub) | Pass |
| Purge refusal (wrong phrase; no purge typed) | Pass |
| Agent uninstall (keep data) | Pass |
| Agent reinstall + reconnect + battery | Pass |
| Hub uninstall (keep data) | Pass |
| Hub reinstall reusing existing DB | Pass |
| Hub account / config / history preservation | Pass |
| Reboot persistence (daemons, ports, health, reconnect, data, battery) | Pass |
| Battery telemetry (`ioreg` + Beszel) | Pass |
| Resource observations | Pass (RSS unobservable as mobile; rest recorded) |

## Restage note

The existing root-owned `/usr/local/sbin/beszel-ios-installer-rc` on the
reference device still contains the `0.4.0` installer and was deliberately not
overwritten during validation. Before the final on-device invocation with the
`1.0.0` installer, restage it and confirm: hash match, root
ownership, `/bin/sh -n`, Diagnose Agent, Diagnose Hub, Hub health 200, Agent
up/listening, and DB/account/config/history intact.

## Known limitations

- Only the reference device above is validated. All other iPhones/iPads, SoCs
  (A8+), iOS versions, and rootless jailbreaks remain untested; do not assume
  the A7 runtime patch is needed or harmless elsewhere.
- After a full reboot, manual semi-untethered jailbreak reactivation is
  required before the custom LaunchDaemons can operate. After reactivation,
  Agent and Hub were verified to return automatically.
- Per-process RSS is not observable from a non-root SSH session on this
  device; resource numbers above are observations only.
- The Go test suite could not be run here because the `dist/` frontend embed
  asset is absent from the checkout; this is an environment issue, not a
  code defect, and no Go source was modified.
