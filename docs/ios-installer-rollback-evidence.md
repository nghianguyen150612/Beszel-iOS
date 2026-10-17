# P016 installer rollback evidence

**Status:** Hub update rollback and retry recovery pass isolated fault-injection tests. The production iPad remains out of scope; `IOS-INSTALL-05` is owned by P018.

## Transaction boundary

The Hub update transaction covers Hub data, Agent data when installed, both current and backup executables, both LaunchDaemon plists and backups, install state, the persistent manager, and the four installer-managed service logs. The set is fixed in `hub_snapshot_collect`; arbitrary paths from state or an archive are not accepted.

Before services stop, the installer validates release provenance and checksums, confirms the candidate assets are iOS ARM64 Mach-O executables, checks backup tools and disk space, and validates the managed path chain. It records the prior running and loaded state. It then stops the Agent writer before the Hub and waits for both LaunchDaemon state and the prior process IDs to clear.

The installer inventories every allowed present path and records file SHA-256, size, type, UID/GID, and mode. It archives that complete set into a root-owned private transaction directory, hashes the archive and manifests, extracts the archive into private staging, and compares the extracted inventory before accepting the backup. The durable journal records transaction ID, release, phase, and prior service state. Recovery accepts only the journal's version, field set, transaction identity, phase, and matching verified backup.

Only after the verified backup exists does the update replace binaries or plists. The Hub starts first. The installer verifies the configured loopback health endpoint, starts the Agent, and waits for its health/connectivity probe. It writes install state only after health succeeds, restores the original loaded/running state, then removes the journal. If recovery or service restoration is uncertain, both services are stopped and the journal remains for retry or manual inspection.

## Fault-injection evidence

`sh tests/install-sh-test.sh` passed **873 tests, 0 failures** at the current P016 worktree state. The P016 additions cover backup preflight, failed quiescence, missing or unsafe Hub data, symlink and hardlink rejection, candidate staging, signing, health, migration, state commit, journal commit, archive restore, service restore, and retry recovery.

The migration mock changes a disposable database file's schema marker and replaces a synthetic Hub SSH identity when the candidate Hub is launched. Failed health then verifies that database bytes, key identity, binary, plist, install state, metadata, and prior service state return to the pre-update snapshot. The Agent transaction gate now waits for an authenticated Hub WebSocket or SSH connection marker; process liveness alone does not satisfy it. Separate injected restore and old-service health failures verify that services remain stopped and the journal survives until a later successful retry.

These installer failpoints model a migration mutation; they do not run PocketBase or assert SQLite schema integrity. The independent P013 hosted fixture does run the genuine Hub binaries against synthetic disposable v0.19/v0.21 PocketBase data and records complete post-migration rollback and recovery evidence in [the P013 backup and restore record](ios-hub-backup-restore-evidence.md). Its database archive, database contents, and private key bytes are not committed or uploaded.

## Scope and limits

All P016 installer tests use temporary path overrides and mocked `launchctl`, download, signing, and executable behavior. They do not access `/var/lib/beszel-hub` on a production device, change a LaunchDaemon, or install a binary. Linux tests do not establish iOS code-signing trust or physical startup behavior. P018 must verify runtime behavior, reboot/reactivation, and service persistence on the reference iPad before `IOS-INSTALL-05` can pass.

## P016 final-head hosted fixture

At P016 source `57dbc0a77af5d366e1ca9519f0119ecd260dc6c6`, hosted P013 run [39026256183](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39026256183) passed the genuine disposable v0.19 → v0.21 PocketBase fixture. Its uploaded manifest and failure logs are artifact `12098566938`; GitHub reports SHA-256 `4d29e4008042ced38a9e083493b9de6e794ce7f87cce2736294a106f385a0046`, matching the downloaded ZIP. The manifest confirms a verified cold backup, v0.21 health and ordered migrations, full restore to v0.19 with SQLite integrity and record/schema invariants intact, and all injected interrupted-migration recovery cases passing. It contains synthetic disposable database data and no production data.
