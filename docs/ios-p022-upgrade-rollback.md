# P022 installer, migration and rollback review

**Candidate status: pending exact P022 hosted requalification.** The details below describe the repository contract and carried synthetic evidence; they do not establish that a physical iPad can install or run these binaries.

## Installed contract

The rootful lifecycle engine is `scripts/ios/install-beszel.sh`, reached through the checksum-pinned `install.sh` bootstrap. The engine installs `beszel-agent` and `beszel-hub` under `/usr/local/bin`, stores Agent and Hub state under `/var/lib/beszel-agent` and `/var/lib/beszel-hub`, and keeps installer state and manager under `/var/lib/beszel-ios`. Managed LaunchDaemons are `/Library/LaunchDaemons/dev.beszel.agent.plist` and `dev.beszel.hub.plist`, with labels `dev.beszel.agent` and `dev.beszel.hub`. The CLI name is `beszel-ios`. The rootless layout is explicitly unsupported. Installed Go binaries are signed with `ldid -S`; this is not an Apple developer signature and does not prove trust or launch behavior on device.

The public bootstrap remains pointed at the `iOS` production branch and verifies the lifecycle-engine SHA before execution. The engine resolves the published GitHub Latest redirect once, validates the tag, pins asset downloads to that tag, and verifies the exact `SHA256SUMS` entries before trusting binaries. The current public default is still the published `v0.19.0-ios.1` release observed during P022 inventory. P022 artifacts are selected for review by source SHA and workflow artifact ID; they are never substituted into Latest or automatically installed. A future approved promotion can stage an explicit immutable tag and checksum set through the guarded tag workflow. Changing the installer default is outside P022.

## Lifecycle coverage and limits

P016/P021 disposable installer tests cover fresh/bootstrap checks, rootful and Procursus path detection, checksum and Mach-O rejection, component-specific and combined Agent/Hub flows, agent-only and hub-only mode, reconfigure, repair, ordinary uninstall without purge, explicit purge, transaction backups, staged replacement, failed checksum/extraction/health, service restoration, interrupted updates, rollback, and retry. These tests use path overrides and mocked `launchctl`, network download, signing, and process behavior. The exact P022 installer workflow must rerun them on the final source.

Low disk capacity and unavailable release behavior are checked before service mutation by the lifecycle engine's preflight and pinned-release resolution. The P022 exact-head test must confirm those failures remain fail-closed. Hosted shell tests do not execute `ldid`, `launchctl`, or the binaries on a device.

## Hub migration and rollback

The P013 fixture uses the real v0.19.0 production Hub to initialize synthetic historical data, then opens it with the integrated Hub. It checks ordered migrations, restart idempotence, existing administrator and regular-user roles, user settings, SSH identity, systems and membership, alerts and alert history, network-monitor history, CPU socket metadata, historical telemetry, ZFS pool fields, container update metadata, PocketBase health, schema/record invariants, and SQLite integrity. It also tests a cold backup and restore, original Hub recovery, post-migration rollback, and interruption at five migration/update phases. The exact P022 source checkpoint `3b7f14eef0971b5e0636e98bcd7246878f89c381` passed P013 run `39699212188`; its sanitized fixture-manifest and failure-log artifact `12392591121` (25,581 bytes) downloaded and matched GitHub SHA-256 `0e0474d7b152b590342640d75e868664d4a50a59a75017ba06a377b3aec39c46`. It records synthetic data, a 24-entry cold backup, restored record/role/system invariants, SQLite integrity `ok`, Hub health HTTP 200, post-migration rollback, and recovery of every injected failure point; database and key bytes are not in the archive. The final evidence-document PR head must still rerun P013.

The installer update transaction protects Hub data, Agent data when installed, binaries, plists, installer state, manager, logs, and the prior service state. It restores the old state only after validating the complete backup inventory and checksum. If service restoration is uncertain, services stay stopped and the journal is retained for retry or operator inspection.

| Situation | Binary-only rollback | Database restore | Status |
|---|---|---|---|
| Failed update before Hub migration opens data | Not sufficient by itself | Restore verified cold backup if data or transaction state changed | Covered by transaction failpoints; exact P022 rerun pending |
| Failed migration or failed health after schema mutation | **Unsafe** to point an older Hub at migrated data | Stop writers, verify cold backup, restore complete data/configuration/identity and old binaries, then start old Hub | P013 synthetic restore and interruption evidence passed on P021; exact P022 rerun pending |
| Candidate Hub healthy, then operator chooses downgrade | **Unsafe** unless schema compatibility is independently proved | Restore the full pre-upgrade snapshot, not just `data.db`; keep the migrated database isolated for investigation | Require external operator approval and verified cold backup |
| Agent-only update with Hub unchanged | Restore previous Agent binary and launchd state from verified transaction snapshot | Not ordinarily applicable | Installer lifecycle tests; exact P022 rerun pending |

PocketBase schema rollback is not inferred from reinstalling an older binary. Any migration that mutates the database requires a cold, integrity-checked backup and explicit operator approval before real-device upgrade. P022 uses synthetic data only and does not access `/var/lib` on a physical device.
