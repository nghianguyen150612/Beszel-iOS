# P026 Physical Validation Handoff

**Status: PLAN ONLY — no device access in P025.** P018 Gate F remains **PENDING**.

## Reference target and candidate identity

- Device: iPad mini 2 A1489 (`iPad4,4`), Apple A7 / ARM64.
- OS: iOS 12.5.7 (build 16H81).
- Jailbreak: Amethyst / Procursus rootful. The installer does not support a rootless layout.
- Source SHA/tree SHA, native workflow run URL/ID, artifact IDs, Agent and Hub SHA-256, `SHA256SUMS`, native validation manifest SHA-256, frontend asset digest, Go/Xcode/SDK versions, and minimum iOS target: **pending exact merged-source qualification**.

P026 must use the immutable candidate manifest produced by the hosted consumer for the final integration merge SHA. The manifest and PR checkpoint must identify the same native workflow run and artifact IDs. Never identify the candidate only by branch name, `Latest`, a filename, or a GitHub-reported ZIP digest. P025 will record the final manifest location in the PR after qualification; until then, this document does not identify installable bytes.

## Transfer and preflight

1. Fetch the payload and validation evidence by the recorded artifact IDs from the recorded workflow run using an authenticated GitHub path. Preserve the downloaded archive and record its independently calculated digest when the supported API provides the ZIP bytes.
2. Check the embedded Agent and Hub files against `SHA256SUMS` and the separately recorded binary SHA-256 values before transfer to the device. Verify the source SHA, run ID, artifact IDs, and toolchain in the native validation manifest.
3. Transfer the exact verified files over the operator's authenticated device channel. Do not use a mutable branch URL, `Latest`, a public mirror, or a shell pipeline from a network URL.
4. Before mutation, confirm the target is `iPad4,4`, ARM64, iOS 12.5.7, and the expected rootful Procursus environment. Abort on a mismatch; do not use a stock or rootless device as a substitute.
5. Save the candidate manifest, checksum output, operator, UTC time, and device identity in the P026 record. Do not record Agent keys, Hub private keys, or other credentials.

The candidate is not a published release. The existing installer resolves GitHub Releases `Latest`, which currently identifies `v0.19.0-ios.1`; its normal install/update commands do not select this P025 candidate. P026 must use a reviewed candidate-specific installation procedure that consumes the immutable verified files, or stop before mutation. Do not invoke the normal `install` or `update` flow expecting it to install P025 binaries.

## Install layout and service configuration

The supported layout is rootful:

| Component | Binary | Data/configuration | LaunchDaemon | Logs |
| --- | --- | --- | --- | --- |
| Agent | `/usr/local/bin/beszel-agent` | `/var/lib/beszel-agent` | `/Library/LaunchDaemons/dev.beszel.agent.plist`, label `dev.beszel.agent` | `/var/log/beszel-agent.log`, `/var/log/beszel-agent.err.log` |
| Hub | `/usr/local/bin/beszel-hub` | `/var/lib/beszel-hub` | `/Library/LaunchDaemons/dev.beszel.hub.plist`, label `dev.beszel.hub` | `/var/log/beszel-hub.log`, `/var/log/beszel-hub.err.log` |
| Installer state/manager | `/usr/local/bin/beszel-ios` | `/var/lib/beszel-ios` | n/a | n/a |

Default Agent listen port is `45876`; default Hub HTTP port is `8090`. The Agent plist contains `DATA_DIR`, `LISTEN`, `KEY`, `LOG_LEVEL`, and `PATH`; never copy the `KEY` value into logs or evidence. The Hub plist runs `beszel-hub serve --http 0.0.0.0:<port> --dir /var/lib/beszel-hub`. Both jobs use `RunAtLoad` and `KeepAlive`; the Hub has a 10-second throttle. P026 must inspect the generated plists locally and redact secret values before recording evidence.

The installer manages these iOS 12 services with `launchctl load -w`, `unload`, and `stop`. Its supported order is Hub then Agent to start both, Agent then Hub to stop both, and Agent/Hub stop followed by Hub/Agent start to restart both. Read-only `beszel-ios status`, `diagnostics`, and `doctor` may be used before mutation; service commands change device state.

## Trust and signing prerequisites

- A rootful Amethyst/Procursus environment and an operator-authorized root session are required for these paths and LaunchDaemons.
- `curl`, `launchctl`, and a SHA-256 tool are installer dependencies. `ldid` is required to ad-hoc sign binaries staged by the installer.
- P025 validates unsigned native artifacts as iOS ARM64 Mach-O for minimum iOS 12.0. It does not establish Apple code-signing trust, jailbreak trust-cache acceptance, device startup, entitlements, or launchability.
- Before proceeding, the P026 operator must confirm the device's signing behavior with a disposable, recoverable procedure. Stop on a signing or LaunchDaemon rejection; do not weaken checks or add entitlements as an unreviewed workaround.

## Rollback and data protection

Before installation, make a verified, private backup of the current Agent and Hub binaries, both LaunchDaemon plists, installer state/manager, Agent data, Hub data, and relevant service state. Keep the backup outside the directories being replaced and confirm adequate free space. Preserve Hub database and identity/key material; redact all secrets from recorded evidence.

Use the reviewed installer transaction and its verified snapshot/journal for updates. Its tested rollback restores the prior binaries, plists, state, Hub data, Agent data, logs, and prior service state. Do not manually start the old Hub against candidate-migrated data. If recovery is incomplete or uncertain, keep both services stopped and retain the transaction journal and backup for operator recovery. A normal uninstall preserves data; never use `--purge` during acceptance. Test binary rollback and database rollback as separate cases.

## P026 cases to execute under operator control

1. Artifact transfer and SHA-256 verification.
2. Trust/signing acceptance and Agent startup without `SIGILL`.
3. Hub startup and health check.
4. Rootful installer flow and LaunchDaemon start/stop/restart.
5. Reboot and jailbreak reactivation, then service persistence.
6. Battery, RAM, and CPU telemetry; Agent-to-Hub connection.
7. WebSocket/SSH behavior and TLS monitoring.
8. Historical data migration, backup, recovery, and interrupted migration rollback.
9. Uninstall without purge and controlled binary/database rollback.
10. Final operator review of the evidence and remaining limitations.

Native Wi-Fi diagnostics remain disabled; do not enable private MobileWiFi access or infer Wi-Fi telemetry from other collectors. Do not claim stock iOS support, Apple trust, signing acceptance, device startup, or production readiness before the relevant physical evidence and release-owner review exist.

No device access, installation, or service operation was performed for this P025 handoff.
