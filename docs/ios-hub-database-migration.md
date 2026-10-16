# P013 Hub Database Migration and Recovery

**Status:** In progress. Hosted synthetic evidence now passes for the genuine v0.19 fixture, cold full-state backup and separate v0.19 restore, real v0.19→v0.21 migration, and second-start stability. Complete post-migration rollback, migration interruption recovery, API/security/ownership, retention, full regressions, and final-head native CI remain pending.

## P012 scope disposition

```text
P012 native Wi-Fi RSSI:
DEFERRED — REFERENCE DEVICE UNAVAILABLE

P012 fallback/source/CI:
ACCEPTED FOR INTEGRATION

Native RSSI measured:
NO

Normal iOS Wi-Fi RSSI:
UNAVAILABLE / NIL

P018 physical validation:
PENDING

P013:
AUTHORIZED
```

PR [#14](https://github.com/nghianguyen150612/Beszel-iOS/pull/14) was merged normally into `ios/upstream-v021-integration` with merge commit `d6239915e404668c10453ab2ec44f797b2f0178e`. Its final-head iOS Build passed at [run 38837362864](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38837362864); installer lifecycle passed at [run 38837362750](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38837362750). The approved decision and evidence limits remain in PR #14 and its linked comments. Native Wi-Fi RSSI has no measured result. `WIFI-ABI-12`, `WIFI-ASSOCIATION-12`, `WIFI-AUTH-12`, and the physical iOS part of `WIFI-BOUND-12` remain deferred to P018. P016 owns any later evidence-backed signing, entitlement, or helper packaging change. No release is authorized.

## P013 starting source

The actual P013 starting integration SHA is `d6239915e404668c10453ab2ec44f797b2f0178e`, discovered after the P012 merge. P013 is based on that tree. The production `iOS` ref remains `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; frozen `main` remains `bb23f99b3eeaced9608e11e280e7c986dba650be`.

## Genuine v0.19 fixture recipe

[`tests/hub-migration/harness.py`](../tests/hub-migration/harness.py) archives the actual production v0.19 source at `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`, builds its pinned frontend with frozen Bun 1.4.0 dependencies, and builds the real Hub binary. It separately builds the current integration Hub and frontend. It starts v0.19 in a temporary directory outside `/var/lib/beszel-hub`; it does not create a modern database and relabel it.

The fixture uses synthetic credentials only and seeds records through the supported first-user endpoint, PocketBase record APIs with the disposable synthetic superuser, and the supported v0.19 `config.yml` system configuration. It contains an administrator/superuser, a regular user, per-user settings, a system with both users, system details, agent fingerprint/configuration, a Hub SSH identity, system and container history, a container, systemd service, SMART device, ZFS pool, alert, and alert history. It reopens the original database with the genuine v0.19 binary and checks identities, roles, system ownership, and seeded record IDs before migration testing begins.

The generated manifest records exact source and binary hashes, application versions, collection schema, SQLite schema hash and integrity result, migration ledger, record counts and relations, account/system IDs, timestamps, configuration hash and mode, and the SSH public-key fingerprint. The database, synthetic credentials, and private key remain disposable temporary state and are not committed.

## Hosted checkpoint evidence

The exact feature head `415cf3914c0347be53fe02796a6f063f1bbbced0` passed [P013 workflow run 38840950368](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38840950368). The uploaded synthetic manifest artifact is `12020648066` (SHA-256 `a18e3974f2fde420ffcb539dd11fa672140792b325c1026cc5cfb03253fa4209`); the manifest SHA-256 is `f93cf69ab8432ca8ae8f2157ef184ce00a18eed3da86883299ce5ea9bde8a396`. It expires 2026-10-30. The artifact contains a JSON manifest only, not a database or private key.

**Phases A, B, and C: PASS for the hosted synthetic harness run.** The runner built Beszel `0.19.0` at production source `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` and Beszel `0.21.0` at feature source `415cf3914c0347be53fe02796a6f063f1bbbced0`, with Go 1.27.1 and Bun 1.4.0. The actual v0.19 startup/bootstrap path initialized a 20-collection database; `PRAGMA integrity_check` returned `ok`. The fixture has two accounts (administrator/superuser and regular user), per-user settings, one shared system, agent identity/configuration, and system/container history, container, systemd, SMART, ZFS, alert, and alert-history records.

The cold archive was created after Hub shutdown and confirmation that no Agent writer remained. Its 24-entry inventory includes the complete Hub data tree, `data.db`, `data.db-wal`, `data.db-shm`, `auxiliary.db`, Hub and Agent binaries, LaunchDaemon definitions, installer state, modes, ownership, and timestamps. Archive SHA-256: `41877eaabae8bba97e24a8d2dc50aa6c2b41f1a067f99759455360d8a302ee58`; inventory SHA-256: `fc3d1063defbc53ff8eeaa25b28f0b2cf47fbb8a8be61f3f3a17f47db687c821`. The backup helper also passed missing-file, active-writer, insufficient-space, bad-checksum, interrupted-`tar`, and exact-restore-inventory tests.

A separate extraction reopened under the actual v0.19 binary: health HTTP 200; account IDs and roles, settings IDs, system ID and membership, a committed WAL-backed metric row, and SSH public-key fingerprint were preserved. The v0.21 binary then migrated a fresh backup clone and returned health HTTP 200. All v0.19 migration-ledger entries remained; the five expected v0.20/v0.21 schema migrations ran once and in order. Legacy collections and alert owner rules survived; new monitor collections, monitor indexes/relations, certificate/server/TLS fields, container `updatable`, and ZFS `display_name`/`raw`/`status` fields were checked. Account/system/history/config/key invariants and SQLite integrity passed. A second v0.21 startup returned health HTTP 200 without changing schema hash, record counts, migration ledger, or tested identities.

The complete manifest, including schema, migration ledger, record IDs/counts, synthetic timestamps and relation checks, hashes, and toolchain versions, is in [`ios-hub-database-validation.json`](ios-hub-database-validation.json). Detailed archive/restore evidence is in [`ios-hub-backup-restore-evidence.md`](ios-hub-backup-restore-evidence.md). These results do **not** yet establish `DB-RESTORE-01`: a rollback from an already migrated database, or `DB-INTERRUPT-01`: recovery after migration/startup failure. Those, functional monitor/alert API cases, security/ownership, retention, full regressions, and final-head native CI remain open. No real user data or production service was used; no release is authorized.
