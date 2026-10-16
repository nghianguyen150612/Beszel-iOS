# P013 Hub backup and restore evidence

**Result:** The genuine hosted synthetic v0.19→v0.21 upgrade, full cold backup, original v0.19 restore, post-migration rollback, and five interruption recoveries passed. This is Linux hosted evidence using disposable synthetic state; it is not a physical iOS upgrade test.

## Source and run identity

- P013 starting integration SHA: `d6239915e404668c10453ab2ec44f797b2f0178e`.
- Tested feature source: `db82eb3b4d2066182292d1601b5e2e763ed05f68`.
- Hosted migration workflow: [run 38848300763](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848300763).
- Synthetic evidence artifact: [12023723159](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848300763), SHA-256 `01793741b798c851b6ade669be3c3d568928bb04e41d80e618ed56ef57d6a5af`; manifest SHA-256 `ce7b307de85eb5d9238227a06ea9d15bc1bdf03a6c77319486a449ace4845a17`.
- The artifact contains `fixture-manifest.json` and diagnostic logs only. It does not contain the database archive, database contents, SSH private-key bytes, passwords, or tokens.

The fixture was initialized and reopened by the actual Beszel Hub `0.19.0` built from production source `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` (Hub SHA-256 `bdcdf9457e6141575349b8a7967b133be4d866a1c7747b37be3585b0f679ec9b`, PocketBase `v0.40.2`). The old Agent binary SHA-256 is `454f82b7cbe74b891c183a0c6a4fa3561f272db6b83d27c606fbde73186162be`. The integrated v0.21 candidate used Go `1.27.1`, Bun `1.4.0`, and PocketBase `v0.40.4`. All fixture files were under disposable temporary directories outside `/var/lib/beszel-hub`.

## Cold full-state backup

The harness stopped the synthetic Agent writers and Hub, confirmed no writer process remained, checked available space, then made the archive outside fixture staging. The full checkpoint included 24 entries: the complete Hub data directory, `data.db`, present `data.db-wal`/`data.db-shm`, `auxiliary.db` and its sidecars, Hub configuration, synthetic Hub SSH identity, old Hub and Agent executables, service definitions, installer state, and directory/file metadata. It restored the checkpoint to a separate directory and compared the complete inventory.

Run 38848300763 evidence:

- Archive SHA-256: `acd575d1cba0a530d744899c453908bb64d5c8413f7a199459f6303345ee70a9`.
- Content-hashed inventory SHA-256: `5396858d151cbb090211407717c7d09b5d7428d4d742c686ab0e536993f744d0`.
- Entries: 24; `data.db-wal`, `data.db-shm`, `auxiliary.db`, `auxiliary.db-wal`, and `auxiliary.db-shm` were included.
- Synthetic Agent writer PIDs at archive time: none; Hub exit was confirmed.
- Free bytes before archive: `89,188,982,784`; required threshold: `71,719,054`.
- Restored inventory matched; SQLite `PRAGMA integrity_check` returned `ok`.
- The public machine-readable manifest records sanitized path, type, size, UID/GID, and mode for each entry. The synthetic fixture used UID/GID `1001:1001`; config and Hub SSH key were `0600`, old Hub and Agent binaries `0755`, service plists and installer state `0644`, and the database/SQLite sidecars `0644`.

The inventory includes content hashes for private fixture files, so only its SHA-256 and sanitized metadata are committed. The archive and full inventory are not attached to GitHub. Negative tests reject missing required files, a live Hub, a live Agent writer, insufficient space, a checksum mismatch, and an interrupted `tar` process. A failed or incomplete archive is never published as a valid checkpoint.

## Genuine v0.19 fixture and migration

The actual v0.19 startup/bootstrap path seeded two synthetic accounts (admin/superuser and regular user), two user settings, one shared system, system details, Agent fingerprint/configuration, Hub SSH identity, system and container metrics, a container, systemd service, SMART device, ZFS pool, alert, and alert history. Record IDs, roles, relations, counts, timestamps, schema, migration ledger, configuration hash/mode, key fingerprint, and SQLite integrity were captured.

The 12-entry v0.19 migration ledger became 17 entries after the ordered five v0.21 migrations. The first v0.21 startup and second startup both returned health HTTP 200. Schema, ledger, counts, ownership, record IDs, settings, history, config hash, and key identity remained stable; `integrity_check` returned `ok`. The harness records and validates the full schema delta: new monitor collections/relations/indexes and certificate/server/TLS fields; alert `state`, the `NetworkMonitorLoss` option and tightened create/update rules/index; alert-history `monitor_name`; and ZFS `status`. It rejects unlisted schema changes and confirms `containers.updatable` plus ZFS `display_name`/`raw` existed in v0.19 and survived migration unchanged.

The migrated API flow created and updated monitor and alert records, queried a non-empty monitor-history result and alert history, rejected a nonmember's reads and writes, restarted the Hub, verified persistence, and confirmed delete/cascade behavior. The real rollup manager ran immediately after migration against disposable system, container, and monitor samples; expected aggregates and response sums/counts/losses were verified. Retention cleanup removed expired samples and remained stable on a second pass.

## Restore with the original v0.19 Hub

Before v0.21 migration, a separate extraction of the cold checkpoint was opened with the original v0.19 Hub. `/api/health` returned HTTP 200. The original account IDs and roles, user-settings IDs, system ID and membership, committed WAL-backed metric ID `megyurmgq1eaqz7`, configuration hash `d3606a4014c9d50145ac52b426dc564ee149d1435dbec88d68208e8448062d7b`, SSH public-key fingerprint `SHA256:1hCMtN4vBwY33YpGukPPOERoXqyO+YaElR1Q26giFBE`, pre-upgrade schema, and migration ledger were preserved.

## Complete rollback after v0.21 migration

The v0.21 database was kept separate from the original backup and preserved for inspection. Rollback verified the backup checksum, quarantined the migrated candidate, restored the complete pre-upgrade data and metadata plus both original executables and service/configuration/installer state, and only then started v0.19 against the restore. The old Hub returned HTTP 200; schema hash, ledger, record counts, user/system/history IDs, roles, settings, configuration hash, and SSH fingerprint matched. SQLite integrity returned `ok`. Binary-only downgrade is not accepted as rollback.

## Interrupted migration recovery

Five deterministic disposable failure points were injected:

1. Before migration invocation; the candidate launcher exited before opening the database.
2. During schema migration at `1790273174_network_monitor_server.go`.
3. After schema changes but before normal Hub startup.
4. Before the first healthy candidate response, with readiness returning injected HTTP 503.
5. During controlled service restart after a previous healthy v0.21 start.

For each case the candidate was stopped and diagnostic logs preserved. No partial 0.21 database was opened by v0.19. Each recovery verified and extracted the entire original checkpoint plus old executables/configuration, then confirmed v0.19 health HTTP 200, original schema and migration ledger, record counts/IDs, account roles, system membership, settings, configuration and SSH identity. The artifact contains per-case logs and hashes.

## Scope limits

The fixture used only synthetic disposable data. No production data directory, user database, iPad, LaunchDaemon, signing configuration, package installation, release, or deployment was touched. The P012 Wi-Fi RSSI deferral remains in force; P018 owns physical validation. Gate D still requires P014 and P015.
