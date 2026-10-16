# Hub database backup, upgrade, and rollback plan

**Production state is persistent.** `/var/lib/beszel-hub` contains the Hub's
PocketBase database and related persistent state. It is never disposable and is
not a migration-test directory. P003 performs no migration and makes no claim
that a v0.21 binary has migrated a database.

That statement describes P003 scope. P013 later performed the synthetic, disposable v0.19→v0.21 migration and full restore tests recorded below; no production state was opened or modified.

## Source inspection and migration behavior

The verified 0.19 iOS baseline and frozen upstream target were inspected at
`origin/iOS` (`7e29c185458f79be274bb09ca57a0be9fc1b63f7`) and `upstream/main`
(`bb23f99b3eeaced9608e11e280e7c986dba650be`). Both Hub entry points blank-import
`internal/migrations`, configure PocketBase with `DefaultDataDir`, register
`migratecmd` against the migrations directory, create `hub.NewHub`, and call
`StartHub`/`pb.Start`. PocketBase owns the migration ledger and runs registered
migrations during startup. The Hub's `onAfterBootstrapAndMigrations` hook
explicitly defers Hub initialization to `OnServe` because PocketBase
initialization must follow database migrations. Hub startup then applies
settings and collection authorization rules before it registers routes, starts
the server, and initializes polling/cron work.

The inspected 0.19 migration set includes the 0.19 collection snapshot and
one-off pool/container field migrations. Upstream v0.21 replaces the snapshot
with `0_collections_snapshot_0_20_0.go`, removes an old pool update migration,
and adds/renames migrations for network-monitor certificate/server/TLS fields
and ZFS pool fields. The v0.20 snapshot defines `network_monitors` and
`network_monitor_stats`, including indexes on system/type/created,
monitor/type/created, and type/created. It also incorporates container
`updatable` and ZFS `display_name`/`raw` fields. Later migrations add
`network_monitors.certInfo` (JSON), `server` (text), `skipTlsVerify` (bool), and
`zfs_pools.status` (text). These migration functions add fields and their down
paths remove them; later writes to new fields therefore cannot be assumed to
survive a downgrade. The collection snapshot also carries schema, indexes, and
rules. Alert authorization rules are reapplied during Hub startup. No down
migration or binary-only rollback is treated as a safe restoration of the 0.19
database.

History behavior also changes outside migration files. Upstream `records.go`
adds per-monitor history aggregation from 1m to 10m, 20m, 120m, and 480m tiers;
the Hub runs that rollup job every ten minutes. `records_deletion.go` retains
1m samples for one hour, 10m for 12 hours, 20m for one day, 120m for seven days,
and 480m for 30 days across system, container, and monitor statistics. The
hourly deletion job also trims alert history to the newest 200 records per user
once a user has more than 250. These are intentional retention rules, but their
first scheduled run after upgrade can remove data. The disposable fixture must
test these boundaries with a controlled clock and verify monitor rollups and
existing system history separately.

The observed device data directory contains `data.db`, a recently written
`data.db-wal`, `auxiliary.db`, and `id_ed25519`; Hub configuration is also stored
under the PocketBase data directory. PocketBase/SQLite may use `data.db-shm`
and other sidecars. The backup therefore captures the **entire** directory,
including hidden files and any sidecars present, instead of selecting files by
name. The Hub binary, Agent binary, LaunchDaemon plists, and
`/var/lib/beszel-ios/install-state` are captured as a separate checkpoint set.

## Backup method and preconditions

Use a service-stop and filesystem archive. Do not copy `data.db` while the Hub
is writing. SQLite WAL can contain committed transactions not yet checkpointed;
copying only the main database can produce an incomplete backup. Do not delete
WAL/SHM files manually. On the reference iPad, run these steps as root; service
files and paths must match the installed system.

1. Schedule a maintenance window and measure the complete data-directory size
   (`D`) and checkpoint-set size (`C`). Require backup storage with at least
   `D + C + 20%` free before archiving, data-volume headroom of at least
   `D + 20%` for quarantine/restore, and scratch storage of at least
   `3 × D + C + 20%` for the source fixture, migration copy, and restore copy.
   If the device cannot meet these bounds, stop and arrange protected external
   storage; do not free space by deleting application data. Keep the backup under a protected path such as
   `/var/backups/beszel-hub/<UTC timestamp>/`, outside application staging and
   outside `/var/lib/beszel-ios`.
2. Record current Agent/Hub versions, binary checksums, file ownership/modes,
   LaunchDaemon state, installer state, database file inventory, and available
   space. Do not record file contents or secret values.
3. Stop the Agent first so no new local stats arrive, then stop the Hub so all
   PocketBase/SQLite writers and cron work exit. On the reference iPad, unload
   `dev.beszel.agent` and then `dev.beszel.hub`:

   ```sh
   launchctl unload /Library/LaunchDaemons/dev.beszel.agent.plist
   launchctl unload /Library/LaunchDaemons/dev.beszel.hub.plist
   ```

   Verify both processes have exited. If a writer remains, abort; do not archive
   a live database.
4. Create a root-only backup directory (`0700`). Its contents are
   `hub-data.tar` (the complete persistent data tree), `checkpoint-state.tar`
   (the exact binaries/plists/installer state needed to recover the 0.19
   service), `SHA256SUMS`, and an operator inventory containing versions,
   timestamps, file paths, owners/modes, sizes and free-space measurements.
   The inventory must not contain secret values. Use a filesystem archive that
   preserves directory structure, file modes, ownership, symlinks, and all
   files. For the iPad, the installed BSD `tar` may be used as follows; confirm
   these options on the target before an authorized production maintenance:

   ```sh
   umask 077
   backup_id=$(date -u +%Y%m%dT%H%M%SZ)
   backup_dir="/var/backups/beszel-hub/$backup_id"
   mkdir -p "$backup_dir"
   chmod 700 "$backup_dir"
   tar -cpf "$backup_dir/hub-data.tar" -C /var/lib beszel-hub
   tar -cpf "$backup_dir/checkpoint-state.tar" -C / \
     usr/local/bin/beszel-agent \
     usr/local/bin/beszel-hub \
     Library/LaunchDaemons/dev.beszel.agent.plist \
     Library/LaunchDaemons/dev.beszel.hub.plist \
     var/lib/beszel-ios/install-state
   (cd "$backup_dir" && shasum -a 256 hub-data.tar checkpoint-state.tar > SHA256SUMS)
   (cd "$backup_dir" && shasum -a 256 -c SHA256SUMS)
   tar -tf "$backup_dir/hub-data.tar" >/dev/null
   tar -tf "$backup_dir/checkpoint-state.tar" >/dev/null
   ```

   If an optional file is absent, record that fact and archive the remaining
   required files. Do not silently omit a file that exists. On the iPad, restored
   service/data ownership is normally `root:wheel`; Linux disposable harnesses
   use the fixture's expected owner (normally `root:root`). Preserve recorded
   modes; do not apply recursive permission changes to the live tree.
5. Verify the hash manifest, archive listings, archive sizes, expected database
   and key paths, recorded owner/modes, and sufficient free space. Store a
   second copy on protected storage with access limited to the operator. The
   manifest contains hashes, not credentials or database contents.
6. Demonstrate extraction and Hub 0.19 startup against a **separate disposable
   restore copy** before the checkpoint is accepted for a production upgrade.

The backup is sensitive: it includes accounts, sessions/configuration,
historical data, the Hub SSH private key, service metadata, and possibly SQLite
sidecars. Keep it root-only and encrypted at rest; do not upload it to GitHub,
attach it to a PR, or copy it into disposable application staging.

## Offline disposable migration test

Run this progression only on a scratch VM/workspace or isolated device data
directory. Never point it at `/var/lib/beszel-hub` on the production iPad.

1. Create a synthetic 0.19-compatible PocketBase database using the supported
   0.19 Hub initialization/test-app path or repository fixtures. Do not hand
   edit a current collection snapshot and call it a 0.19 database.
2. Seed representative synthetic users/roles, settings, systems, config,
   fingerprints, metric samples and time ranges, alerts/history, and relevant
   collection rows. Record IDs, relations, counts, and timestamps.
3. Create the consistent backup above; verify archive and hash manifest; prove
   that the test restore copy opens with the 0.19 binary.
4. Clone the backup into a fresh disposable data directory. Run the future 0.21
   Hub binary against that clone with isolated ports and no production
   credentials/endpoints.
5. Capture startup/migration logs and PocketBase migration ledger. Compare
   collection names, fields, indexes, rules, IDs, relations, counts, and
   representative values. Start the candidate a second time to check
   idempotence.
6. Exercise `/api/health`, login/authorization fixtures, settings, systems,
   historical stats, alerts, monitor create/update/delete/results/history, and
   frontend assets. Check that retention/rollup work does not delete fixture
   history unexpectedly.
7. Restore the untouched backup into another disposable directory, start the
   0.19 binary, and verify health plus original account, settings, system,
   history, and key-identity invariants.
8. Interrupt a candidate migration/startup at controlled checkpoints and test
   failed-start recovery. Never open a partially migrated DB with 0.19. Recovery
   must restore the complete backup and old binary together.
9. Keep the fixture recipe, hashes, migration logs, before/after invariants,
   restore output, and failure-injection results as P013/P017 evidence. The
   upgraded binary does not exist at P003, so these steps are specified but not
   reported as passed.

## Restore and emergency recovery

Treat a migration failure, failed Hub startup, schema/ledger mismatch, missing
account/history, hash mismatch, or unexpectedly changed key as an emergency.
Stop Agent first and Hub second. Keep both stopped until the recovery set is
verified. Never start the old binary on a migrated or partially migrated data
directory.

1. Verify `SHA256SUMS` and archive listings before extraction. If a checksum or
   archive is bad, do not overwrite the current data; locate the second
   verified copy.
2. Preserve the failed data directory and logs under a timestamped quarantine
   name outside staging. Do not delete it during incident diagnosis.
3. Restore `hub-data.tar` to `/` so `/var/lib/beszel-hub` is restored as a whole;
   restore the checkpoint archive with the old binaries, plists, and installer
   state. Preserve recorded owner/modes and recheck them before startup.
4. Validate hashes and file inventory. If `sqlite3` is available, run an
   integrity check against a disposable copy of the restored database, not the
   only recovery copy. Verify expected `data.db`/sidecars, `auxiliary.db`, key,
   config and migration files before service start.
5. Start the restored Hub first, then validate it, and start Agent only after
   Hub health is confirmed:

   ```sh
   launchctl load /Library/LaunchDaemons/dev.beszel.hub.plist
   # Check /api/health and account/system/history invariants without secrets.
   launchctl load /Library/LaunchDaemons/dev.beszel.agent.plist
   ```

   Check reconnect and fresh telemetry. If health or invariants fail, stop the
   Hub again and keep both services down.
6. If the restore fails validation, keep services down and escalate to the
   verified secondary backup. Do not repeatedly start different Hub versions
   against uncertain state.

## Production rollback gate

A candidate cannot be approved for production upgrade until all of the following
are demonstrated on disposable data and the operator has a verified recovery
set:

* A consistent full backup is verified outside disposable staging.
* Restore is demonstrated with the old 0.19 Hub binary.
* Original accounts, settings, systems, keys, configuration, and history survive.
* New schema, monitor/alert behavior, retention, and migration ledger are checked.
* Interrupted migration and failed-start recovery are demonstrated.
* Installer rollback is reconciled with restoring the database and service state;
  binary-only rollback is explicitly insufficient.

P003 specifies this process only. No production database, migration, installer,
or release action is performed by this document.


## P013 hosted synthetic evidence — 2026-10-16

The genuine v0.19→v0.21 Hub migration workflow passed on feature source `db82eb3b4d2066182292d1601b5e2e763ed05f68` in [run 38848300763](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848300763). It created and reopened a synthetic database with the actual supported v0.19 Hub path, stopped writers, captured and restored a 24-entry cold checkpoint, migrated a separate copy with v0.21, verified schema/data/ledger/second-start behavior, rolled back the complete data-plus-binary state, and recovered from five migration/startup/restart interruptions. The v0.19 restored Hub health returned HTTP 200 in the pre-upgrade, full-rollback, and each interrupted-recovery case.

The workflow artifact [12023723159](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38848300763) has SHA-256 `01793741b798c851b6ade669be3c3d568928bb04e41d80e618ed56ef57d6a5af`; its manifest SHA-256 is `ce7b307de85eb5d9238227a06ea9d15bc1bdf03a6c77319486a449ace4845a17`. Cold archive SHA-256 is `acd575d1cba0a530d744899c453908bb64d5c8413f7a199459f6303345ee70a9`; content-hashed inventory SHA-256 is `5396858d151cbb090211407717c7d09b5d7428d4d742c686ab0e536993f744d0`. The artifact contains a synthetic manifest and diagnostics only. Full backup details and per-case evidence are in [`ios-hub-backup-restore-evidence.md`](ios-hub-backup-restore-evidence.md) and [`ios-hub-database-validation.json`](ios-hub-database-validation.json).

This hosted result verifies the synthetic P013 database and recovery acceptance cases. It does not validate migration on the iPad, authorize production upgrade, or authorize release. P018 retains the physical-device obligations.
