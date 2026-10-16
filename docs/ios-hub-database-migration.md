# P013 Hub Database Migration and Recovery

**Status:** In progress. Only the genuine v0.19 fixture and original-version reopen are validated so far. Migration, backup, rollback, interruption, authorization, retention, full regression, and hosted native gates remain pending.

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

## Current evidence

**Phase A fixture: PASS.** The harness built Beszel `0.19.0` from the production `iOS` source SHA and Beszel `0.21.0` from the P013 starting integration tree, with real frozen-lockfile frontend builds using Bun 1.4.0. The synthetic 0.19 database had 20 initialized collections; `PRAGMA integrity_check` returned `ok`. The fixture contains two users (admin/superuser and regular), two user settings, one owned system, one agent fingerprint, and one each of system details, system history, container, container history, systemd service, SMART device, ZFS pool, alert, and alert history. The original fixture reopened in the genuine 0.19 binary with user IDs, roles, system ID/ownership, and seeded record IDs preserved.

The exact source and binary hashes, full collection schema, SQLite schema hash, migration ledger, counts, IDs, relations, timestamps, configuration hash/mode, and synthetic SSH public fingerprint are recorded in [`ios-hub-database-validation.json`](ios-hub-database-validation.json). Fixture databases and private keys remained in a disposable temporary directory and were removed after the run. No migration or rollback result is claimed in this checkpoint. The backup archive, 0.21 migration, second 0.21 startup, restore, interruption, API/security, monitor/alert ownership, history-retention, regressions, and final-head CI cases are still pending.
