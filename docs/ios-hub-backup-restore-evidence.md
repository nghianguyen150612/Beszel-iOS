# P013 Hub backup and restore evidence

**Status:** Cold full-state backup and pre-upgrade restore passed in a hosted synthetic fixture run. Rollback from a migrated v0.21 state is still pending and is not claimed here.

## Run identity

- Feature branch: `ios/hub-database-migration`
- Tested head: `415cf3914c0347be53fe02796a6f063f1bbbced0`
- Hosted run: [38840950368](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38840950368)
- Artifact: `p013-hub-fixture-manifest`, ID `12020648066`, ZIP SHA-256 `a18e3974f2fde420ffcb539dd11fa672140792b325c1026cc5cfb03253fa4209`
- Manifest SHA-256: `f93cf69ab8432ca8ae8f2157ef184ce00a18eed3da86883299ce5ea9bde8a396`
- Fixture: synthetic only. No production database, user credential, or private SSH key was archived or committed.

## Cold backup procedure and inventory

The harness stopped the genuine synthetic v0.19 Hub with SIGKILL after committing a metric over the API, then confirmed its PID had exited. No synthetic Agent writer was running. It measured free space before archive creation and required at least 120% of the complete staged checkpoint size. The backup was written atomically to a sibling temporary directory outside fixture staging, then checksummed and verified before publication. It was restored to a separate empty location.

The 24-entry archive contains:

- The complete Hub data tree, including `data.db`, present `data.db-wal`, `data.db-shm`, `auxiliary.db`, configuration, synthetic SSH key and all other files.
- Old v0.19 Hub and Agent executables.
- Hub and Agent LaunchDaemon property lists.
- Installer state and version/checksum metadata.
- A content-free inventory of relative paths, file hashes, sizes, modes, owners, groups, timestamps, and symlink targets.

Archive SHA-256: `41877eaabae8bba97e24a8d2dc50aa6c2b41f1a067f99759455360d8a302ee58`.

Inventory SHA-256: `fc3d1063defbc53ff8eeaa25b28f0b2cf47fbb8a8be61f3f3a17f47db687c821`.

The synthetic filesystem had 89,190,199,296 bytes available before backup and required 71,719,054 bytes. Archive entries and inventory matched; SHA256SUMS covered both archive and inventory. Safe extraction restored the recorded file and directory metadata, then strict content/hash/mode/owner inventory comparison passed.

## Backup negative cases

Focused Python tests cover a missing required key, a live Hub, a live Agent writer, insufficient available space, an invalid archive checksum, a real interrupted `tar` process, and successful extraction/inventory comparison. The hosted run executed all seven tests successfully. The interrupted archive was never published as a valid backup; the incomplete archive remained unreadable.

## Restore to v0.19

The complete archive was extracted separately and checked against the recorded inventory. SQLite `PRAGMA integrity_check` returned `ok`. The exact genuine v0.19 binary from the archived restore set started the restored directory; `/api/health` returned HTTP 200. API checks confirmed the same two account IDs and roles (admin/superuser and regular user), settings IDs, system ID and membership, committed WAL-backed metric row ID `u09cya8r9xvoba2`, and SSH public-key fingerprint `SHA256:hdTG7/5F3r/e6lKoc8GQp3PqzDjNuN1lQaLXfiQxucQ`. The original 0.19 Hub and Agent binary SHA-256 values are in the JSON manifest.

A second independent extraction of the same cold archive was migrated with v0.21; the original backup remained verified. Migration and second-start details are in [the migration report](ios-hub-database-migration.md) and the full [machine-readable manifest](ios-hub-database-validation.json).

## Scope and remaining recovery gate

This proves the synthetic cold archive is complete and that a clean pre-upgrade restore runs with 0.19. It does **not** prove a rollback after v0.21 has modified a database, nor recovery after a partially applied migration. P013 still requires both tests to restore the entire compatible state set (data, old binaries, configuration, service definitions, and installer state) from a verified backup. Binary-only rollback is not accepted. No production upgrade is authorized.
