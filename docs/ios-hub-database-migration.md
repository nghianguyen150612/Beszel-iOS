# P013 — Hub database migration, rollback, and authorization

**Status:** The genuine v0.19→v0.21 migration, full backup/restore, complete rollback, five interruption recoveries, API/security checks, and final-head native CI passed. Synthetic Hub migration run [38848300763](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848300763), installer lifecycle [38848300779](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848300779), and iOS Build [38848296639](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848296639) all passed at code head `db82eb3b4d2066182292d1601b5e2e763ed05f68`. Their checks and artifacts are recorded below and in the machine-readable manifest. PR #15 remains Draft while the final documentation checkpoint and its required checks complete.

## P012 disposition

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

PR [#14](https://github.com/nghianguyen150612/Beszel-iOS/pull/14) was merged normally into `ios/upstream-v021-integration` at `d6239915e404668c10453ab2ec44f797b2f0178e`. Its owner-approved rationale is [the PR comment](https://github.com/nghianguyen150612/Beszel-iOS/pull/14#issuecomment-6206076758). Native RSSI was not measured. `WIFI-ABI-12`, `WIFI-ASSOCIATION-12`, `WIFI-AUTH-12`, and the physical portion of `WIFI-BOUND-12` remain deferred to P018. P016 owns any future evidence-backed signing, entitlement, or helper-packaging work. Gate C's physical evidence is pending, and this decision grants no release authorization.

## Source and fixture provenance

- Latest verified P012 feature HEAD: `2a16d0f46f6fd1c91f40e295190b89fb8713b8cd`; pre-merge integration: `33840093bb465f0b164bb7d66aebc8cd85f86a4e`.
- P012 normal merge SHA / P013 starting integration SHA: `d6239915e404668c10453ab2ec44f797b2f0178e`.
- Feature branch: `ios/hub-database-migration`; base: `ios/upstream-v021-integration`.
- Production `iOS` remains `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; frozen `main` remains `bb23f99b3eeaced9608e11e280e7c986dba650be`.
- The fixture is initialized and reopened by the actual v0.19 Hub from production source `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` (Hub SHA-256 `bdcdf9457e6141575349b8a7967b133be4d866a1c7747b37be3585b0f679ec9b`, Agent SHA-256 `454f82b7cbe74b891c183a0c6a4fa3561f272db6b83d27c606fbde73186162be`, PocketBase `v0.40.2`). It is not a current database relabeled as v0.19.
- The v0.21 candidate is built from the feature checkout using Go `1.27.1`, Bun `1.4.0`, and PocketBase `v0.40.4`.
- All files live in disposable temporary directories outside `/var/lib/beszel-hub`. Passwords, OAuth values, Agent tokens, and SSH keys are synthetic; persistent database/key files are never uploaded or committed.

The recipe and exact manifest generator are [`tests/hub-migration/harness.py`](../tests/hub-migration/harness.py), [`backup.py`](../tests/hub-migration/backup.py), and [`rollup_probe.go`](../tests/hub-migration/rollup_probe.go). The fixture includes two users (admin/superuser and regular), two user-settings rows, one system shared by those users, system details, Agent fingerprint/configuration, Hub SSH identity, system and container metrics, a container, systemd service, SMART device, ZFS pool, alert, and alert history. The manifest captures IDs, roles, relations, timestamps, counts, configuration hash/mode, public-key fingerprint, schema, migration ledger, and SQLite integrity without including private material.

## Migration and preservation

The genuine v0.19 database starts with a 12-entry PocketBase ledger. The v0.21 Hub adds the following five entries, exactly once and in this order:

1. `0_collections_snapshot_0_20_0.go`
2. `1790193183_network_monitor_cert.go`
3. `1790273174_network_monitor_server.go`
4. `1790924727_zfs_pool_status.go`
5. `1790964708_network_monitor_skip_tls_verify.go`

The migrated ledger has 17 entries. The original 12 remain, account and system IDs and roles are unchanged, both settings IDs and system-membership relations remain, the synthetic Agent identity/configuration and Hub SSH public-key fingerprint remain stable, representative system/container history and alert/history remain queryable, and SQLite `integrity_check` returns `ok`. The second v0.21 startup returns health HTTP 200 and leaves the schema hash, counts, ledger, and tested identities unchanged.

The v0.21 migration adds `network_monitors` and `network_monitor_stats`, with system/monitor relations and the accepted configuration/history indexes. The monitor fields include `certInfo` (JSON), `server` (text), and `skipTlsVerify` (bool). It adds `alerts.state` (JSON), adds `NetworkMonitorLoss` and `NetworkMonitorCert` to the `alerts.name` options, tightens alert create/update ownership rules, adds the alert system/name index, adds `alerts_history.monitor_name` (text), and adds `zfs_pools.status` (text). The genuine v0.19 fixture already has `containers.updatable`, `zfs_pools.display_name`, and `zfs_pools.raw`; the migration confirms they remain unchanged. The harness compares the complete schema and permits only these ordered alert-option additions and other observed migration deltas, and checks the new monitor collection rules, all indexes, relations, and legacy collection preservation. Full schema and IDs are in [`ios-hub-database-validation.json`](ios-hub-database-validation.json).

## Backup, rollback, and interrupted recovery

The cold backup procedure stops Agent writers and the Hub, verifies process exit, checks space, and archives the complete data directory and checkpoint state. It includes `data.db`, present WAL/SHM sidecars, `auxiliary.db`, config, Hub SSH key, old Hub and Agent executables, service definitions, installer state, and file metadata. It writes SHA256 manifests, verifies the archive and inventory, and restores to another disposable directory. Negative tests cover missing files, active writers, insufficient space, invalid checksums, and interrupted archive creation.

Hosted P013 run [38848300763](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848300763) passed on source head `db82eb3b4d2066182292d1601b5e2e763ed05f68`. Its synthetic-only artifact [12023723159](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848300763) has SHA-256 `01793741b798c851b6ade669be3c3d568928bb04e41d80e618ed56ef57d6a5af`; the manifest SHA-256 is `ce7b307de85eb5d9238227a06ea9d15bc1bdf03a6c77319486a449ace4845a17`. The verified cold archive SHA-256 is `acd575d1cba0a530d744899c453908bb64d5c8413f7a199459f6303345ee70a9`; the content-hashed inventory SHA-256 is `5396858d151cbb090211407717c7d09b5d7428d4d742c686ab0e536993f744d0`. The 24-entry metadata inventory records paths, owner/group, modes, types, and sizes; config and Hub SSH key are `0600`, binaries `0755`, service files and installer state `0644`. The workflow artifact contains no database archive, private key bytes, passwords, or tokens.

The old v0.19 restore reproduces the original ledger, schema hash, IDs/counts, settings, system relations, history, configuration hash, and SSH fingerprint; old Hub health is HTTP 200. The migrated candidate is quarantined separately. Rollback restores the complete archive plus both original executables and service/configuration state, then opens only that compatible restore with v0.19. It returns health HTTP 200 and preserves the original state. No binary-only downgrade is accepted.

Failure injection covers: before migration invocation; in the migration sequence at `1790273174_network_monitor_server.go`; after schema migrations but before normal startup; an injected readiness HTTP 503 before the first healthy candidate check; and a controlled restart after a healthy v0.21 start. The partially migrated copy is not opened with v0.19. Each case restores the verified complete checkpoint and confirms v0.19 health and invariants. Failure logs and per-case checksums are in the run artifact.

## API, authentication, authorization, and history

The migrated Hub returns `/api/health` HTTP 200. Synthetic users authenticate; settings, systems, memberships, and historical records remain accessible to their owners. The migration harness exercises monitor create/update/read/list, creates a synthetic monitor-result row through the superuser API, queries that result through the authorized history list, checks result relations and values, restarts the Hub, and verifies persistence and cascade cleanup. It exercises alert create/update/list/delete and alert-history reads. A nonmember cannot read or modify another system's monitor or another user's alert, and monitor/alert/history queries do not expose foreign records.

A separate fresh-Hub integration test injects failure while saving the first superuser after the regular-user write. The failed attempt leaves zero regular users and exactly one `_@b.b` setup placeholder, which stays singular across restart so setup can be retried. It then races two real first-user HTTP requests. Exactly one becomes the owner (`200`); the other is denied (`403`); successful setup replaces the placeholder with exactly one admin user and matching superuser.

OAuth tests verify new OAuth users always receive `role=user`, repeat login preserves an administrator-assigned role, and missing provider ID or email does not create a user or external-auth link. `TestOAuthDuplicateIdentityUsesExistingAccount` uses the actual OAuth route to confirm a repeated provider ID with a changed email resolves to the original account, does not create an account or duplicate identity link, preserves its administrator role, and ignores attempted role escalation in create data. `TestOAuthAccountLinkingPreservesExistingRole` links a verified existing readonly account by email while OAuth `createData` requests admin; it confirms the original account ID and readonly role remain, with only one account and external-auth relation. Trusted-header authentication now fails closed: the configured identity header is ignored unless `TRUSTED_PROXY_IPS` contains the direct peer. API tests cover the header disabled, unset/empty/blank allowlists, allowed IPv4 CIDRs, multiple networks, invalid-only and mixed CIDRs, untrusted direct peers, and a malformed forwarded identity. Parser tests cover IPv4, IPv6, CIDRs, mapped addresses, and malformed peers. This is an intentional security hardening; deployments using trusted headers must configure the trusted proxy addresses.

Systemd detail responses are `Cache-Control: private, max-age=60` and include `Vary: Authorization`. Two synthetic users exercise authorized, unauthorized, shared-system, and unauthenticated requests; unauthorized responses contain no private detail and do not reach the Agent. Alert tests exercise CRUD ownership, system membership and reassignment, subscriptions, notification delivery after access revocation, and history. Monitor ingestion tests cover foreign monitor spoofing, realtime and historical writes, stale/reconnected state, and monitor ownership.

## Retention and rollup

The v0.21 policy remains unchanged:

| Resolution | Retention |
| --- | --- |
| 1 minute | 1 hour |
| 10 minutes | 12 hours |
| 20 minutes | 1 day |
| 120 minutes | 7 days |
| 480 minutes | 30 days |

Alert history keeps the newest 200 per user once the count exceeds 250. Existing tests cover retention boundaries for system, container, and monitor statistics; alert history by user; duplicate/missing samples; weighted response sums, sample counts, losses and timestamps; sparse and disabled monitor rollups; and repeat cleanup/rollup behavior. The new migration probe reruns the v0.21 migration ledger on the actual stopped migrated fixture and immediately runs the production rollup manager against disposable rows, verifying system, container, and monitor aggregates, then runs retention cleanup twice and checks stable results. A cron-boundary test verifies the registered hourly cleanup at minute 8 and ten-minute rollup scheduling across a startup just after the cleanup tick. It uses the scheduler's parser with controlled timestamps; the live 10-minute/hourly wall-clock waits are not used as test synchronization.

## Evidence and gate state

The full machine-readable manifest and exact backup/restore hashes are in [`ios-hub-database-validation.json`](ios-hub-database-validation.json). Backup details are in [`ios-hub-backup-restore-evidence.md`](ios-hub-backup-restore-evidence.md); security tests are in [`ios-hub-security-validation.md`](ios-hub-security-validation.md). The previous validation JSON is preserved verbatim in [`ios-hub-database-validation-prior-checkpoint.json`](ios-hub-database-validation-prior-checkpoint.json), retaining earlier P012 and P013 pending/blocked observations as historical evidence.

The prior P013 regression checkpoint `1a0fde78` passed audit Python (54), upstream-sync (10), patch sentinels (45), bootstrap (54), installer (814), frozen Bun 1.4.0 install/catalog/frontend tests and production build, and Linux Agent/Hub builds; the genuine frontend has 95 files (88 JavaScript assets). At current code head `db82eb3b4d2066182292d1601b5e2e763ed05f68`, the canonical Go suite passed. Fresh race runs passed for migrations, users/OAuth, bootstrap, trusted proxy, cache policy, and network-monitor ownership/systems; the new verified-email OAuth account-linking test passed normally and under `-race`. Earlier P013 focused race runs also passed alerts and records/retention. A broader upstream Hub race run previously exposed a pre-existing race in `TestAgentWebSocketIntegration`: a test reads exported `connectionManager.State` without the mutex used by its writer. That unrelated race remains documented and is not hidden.

All seven P013 acceptance cases (`IOS-HUB-01` through `IOS-HUB-03`, `DB-BACKUP-01`, `DB-MIG-01`, `DB-RESTORE-01`, and `DB-INTERRUPT-01`) have meaningful synthetic hosted migration/restore evidence at code head `db82eb3`; exact outcomes are in `ios-hub-database-validation.json`. Final-head native iOS validation at `db82eb3` is [run 38848296639](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848296639): Go `1.27.1`, Bun `1.4.0`, Xcode `15.4` with iPhoneOS SDK `17.5`; both artifacts are ARM64 iOS `MH_EXECUTE`, platform 2, minimum iOS `12.0`, built with `CGO_ENABLED=1`, and include 95 Hub frontend assets. The artifact records exact VCS revision `db82eb3`, `vcs.modified=false`, runtime patch inspection, checksums, and no private MobileWiFi linkage. The release job was skipped. The Agent SHA-256 is `45fd38e73923b317e7ae3a2585e137763fd47e47622e9e7bd2488cddeb803fcd`; the Hub SHA-256 is `6fc8fc33865ac9353199d22eb5c68c2c97b428fd1a3777a498c3075d7c4564c4`. Binary ZIP artifact [12024229163](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848296639) SHA-256: `77a90315205ee8126237f8736a1408d641465c431a1f37513c9d2794d68ef8ba`; native validation artifact [12024711683](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848296639) SHA-256: `67002c4cc17150607d85682d42f7d8893d5481a53213f4887bdcfb0b64094d28`. Hosted Linux evidence and cross-compilation do not establish a real iOS upgrade or on-device launch. Gate D still requires P014 and P015. P018 still owns A7/device validation, including native RSSI; no release or deployment is authorized.

## P014 handoff

P014 may start only when requested after P013's normal merge. Carry forward the exact migrated schema and API contracts, preserve the fail-closed proxy rule and the full data-plus-binary recovery model, keep native Wi-Fi RSSI unavailable, and retain the P018 physical obligations. Do not run P014 automatically.
