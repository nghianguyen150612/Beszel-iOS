# P018 — iPad mini 2 A7 device validation and Gate F handoff

**Milestone:** P018. **Status:** NOT RUN / DEVICE UNAVAILABLE. **Gate F:** PENDING. No release authorized.

## Baseline and verified candidate

- P017 normal merge / integration: `f9d6500c7a68572dc8e464dd305d79bf6c1e79a9`.
- Binary-tested product commit: `5a7351f752376e206a3e1809b3698bb03cc87b98`.
- Verified hosted candidate: [iOS Build 39095687398](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39095687398), artifact ID `12124073871`; matching independent archive SHA-256 `698335f09da4fe8f920b14176fe820f3733bd1a24064a3dede14456a5ce044c6`.
- Agent SHA-256: `87dff966e93d324e2b83a5029c749ce4578f906f099c388811a8ad025f73a4df`.
- Hub SHA-256: `54751f8191faddb58e8c933d0dff0929387cfd9cbe68e64b7c37c34a0bc309df`.
- Both iOS ARM64 Mach-O binaries target minimum iOS 12.0. Hosted SDK/toolchain and A7 runtime verification are in `docs/ios-release-candidate-artifacts.json`.
- Go multipart advisory `GO-2026-6608` HIGH and build-only braces `GHSA-vfj7-8cjw-p6xm` HIGH remain OPEN. No public release without explicit residual-risk decision.

**Physical reference:** iPad mini 2 A1489 / iPad4,4, Apple A7, iOS 12.5.7, Amethyst with Procursus **rootful** jailbreak. It is semi-untethered; operator must manually reactivate jailbreak after a full reboot. Prior v0.19 physical checks are historical and cannot pass v0.21 tests.

## Important stop conditions

1. Do not run `beszel-ios install` or `beszel-ios update` against Latest: v0.21 is not yet published; it may select v0.19. Do not use the remote one-line installer while trying to test P017 candidate.
2. Do not replace binaries/LaunchDaemons or reboot until verified backup, known-good v0.19 recovery binaries and **physical access to the device** are secured. SSH-only access is insufficient for a jailbreak-reactivation/reboot test.
3. Never touch production `/var/lib/beszel-hub` for destructive migration experiments. Use an isolated disposable copy, after stopping every writer. DB must never be copied live without a validated consistent snapshot mechanism.
4. Do not perform data purge, trust-cache changes, tweak injection, speculative entitlement injection or MobileWiFi private acquisition.
5. No SSH password, private key, account secret, original DB, personal history, IP address or unredacted full device logs may be committed to GitHub.
6. Record an explicit operator authorization before any signed installation/upgrade. Halt on integrity, signing, storage, launch or backup concerns.

## Phase 0 — Read-only preflight

From an existing secure terminal on the iPad, inspect the read-only `scripts/ios/p018-device-preflight.sh`, transfer it through an authorized channel, then run:

```sh
sh /var/tmp/p018-device-preflight.sh
```

The script prints only OS identity, safe hardware identifiers, utility/path presence, binary checksums, Beszel service labels and free space. It does not contact network services, modify files, run installer commands or restart services. Sanitize its output before sharing.

Host-side safety harness: `sh tests/ios-p018-device-preflight-test.sh`. It checks the script with guarded host command stubs. This is not simulated hardware evidence. In the current ChatGPT Work execution, no authorized iPad terminal/SSH session or target endpoint was available, so Phase 0 remains NOT RUN ON DEVICE; no network scan or connection attempt was made.

Verify device model `iPad4,4`, iOS `12.5.7`, rootful paths `/usr/local/bin` and `/var/lib`, Procursus tooling, `ldid`, current `beszel-ios status both` (run manually as read-only), and expected older installed binaries.

## Phase 1 — Recovery readiness (human/operator approval)

1. Confirm stable power, sufficient space, hands-on access, working jailbreak reactivation path and independent recovery SSH/physical-console route.
2. Inventory prior Agent/Hub binaries, `/Library/LaunchDaemons/dev.beszel.*.plist`, manager, install state, configuration and SSH identity by **hash/permissions only**, not by disclosing keys.
3. Choose safe archive destination outside installer staging; fully stop Agent (writer), then Hub, and prove no writer remains before creating a complete, consistent *cold* backup.
4. Include `data.db` and any WAL/SHM, `auxiliary.db`, Hub config/SSH identity, Agent data, prior binaries/plists/installer manager and state. Verify archive inventory and hashes, permissions, recoverability and available storage. Keep a second trusted copy off-device.
5. Test restore into an isolated directory; do not overwrite the live database for a dry run.
6. Document abort/rollback triggers; do not begin candidate deployment until the human operator approves.

## Phase 2 — Candidate staging and controlled native checks

Only after Phase 1 and explicit user approval: obtain the exact P017 artifact ZIP, verify GitHub archive digest, verify both payload SHA256SUMS, Mach-O platform iOS/ARM64/minOS 12.0, compare Go VCS revision and verify absence of unexpected private imports. Stage candidate separately, **without overwriting production**.

Validate `ldid`/trust on the real jailbreak as applicable; do not infer signing works from a Linux/mock signer. Test direct binary launch with disposable storage, under controlled resources and with the legacy runtime patch; capture exit status / SIGILL / loader diagnostics. Prevent accidental dual occupancy of production ports or database files.

## Phase 3 — Runtime and feature acceptance

Use the P017 candidate, only after authorization, on the reference device:

| Case | Real-device evidence required |
| --- | --- |
| `A7-DEVICE-01` | Agent and Hub start, no SIGILL, no unexplained exits; repeated polling, reconnect and safe reactivation |
| `IOS-AGENT-01` | Actual iOS/arm64, model and sysctl metadata agree; unsupported optional fields absent |
| `IOS-AGENT-02` | CPU, RAM, disk, Wi-Fi/VPN/loopback counters with timestamps; resets do not fabricate metrics |
| `IOS-AGENT-03` | Real ioreg battery percentage and charging transitions; unavailable readings omitted |
| `IOS-AGENT-04` | Network monitor HTTP/HTTPS probes, TLS default verification, bounded timeout and isolation |
| `IOS-AGENT-05` | WebSocket/SSH fault, reconnect, timestamps advancing without manual process restart |
| `IOS-AGENT-06` | Unsupported GPU/SMART/ZFS/systemd/native Wi-Fi RSSI remain nil/unavailable, never measured zero |
| `IOS-INSTALL-05` | Correct launchctl load/unload, Hub-before-Agent start and reverse stop; controlled reboot followed by manual jailbreak reactivation and autonomous service restart |
| `DB-DEVICE-01` | Existing account/config/history/SSH identity preserved across strictly authorized candidate upgrade; backup and safe rollback verified without destructive testing of live DB |
| `WIFI-ABI-12 / WIFI-ASSOCIATION-12 / WIFI-AUTH-12 / WIFI-BOUND-12` | P012 native RSSI research only; no privileged calls before ABI/authorization/provenance and bounded-response proof. If blockers remain, record DEFERRED, not PASS |

Measure multiple timestamps and allow enough time to detect stale metrics. Capture sanitized logs, checksums, health responses and real service outcomes.

## Phase 4 — Safe reboot, restore and final release decision

A reboot is a separate operator-approved operation requiring physical recovery, power and working jailbreak reactivation. Verify both LaunchDaemons only **after jailbreak is reactivated**; collect uptime, service status, HTTP health, Agent authenticated Hub connection and new metric timestamps.

On any failure: stop the candidate, restore compatible binaries **with the complete matching cold-state backup** as needed, verify old health/identity/history, and preserve the incident report. Do not reopen a migrated DB with a mismatched old Hub.

**Gate F may PASS only after all required real iPad test cases have trustworthy, sanitized evidence, residual high risks have explicit operator disposition, and the owner explicitly approves release.** Do not create a tag or GitHub Release within P018 without a separate explicit release authorization.

## Checkpoint / quota recovery

Commit only scripts, redacted metadata, test matrices and notes. Keep the PR Draft until device evidence exists. If iPad is unavailable, report `BLOCKED — REFERENCE DEVICE UNAVAILABLE`, current branch/HEAD, completed preflight-code checks and the first exact authorized device action. Never claim hardware PASS using cloud tests.
