# P030: PARTIAL — RESUMABLE

P029 received HEAD `7fbed064a0a154d4865103b222665b9f1449c845` was verified before edits; terminal evidence reconciliation is P029 HEAD/base `1f8cac7b` (documentation only). P029 is host/native qualified within its implemented scope at source `4fbf4ed27ce0a6e3c921b998031081a1bff70d8e`; see `ios-p029-handoff.md` and the terminal evidence. Artifact `14005295997` is preserved. It does not qualify P030 changes.

P030 branch: `ios/p030-upstream-integration`, stacked on `ios/p029-upstream-integration`. Official `henrygd/beszel` main fetched at `39f9a0765311e83213e5379ea27a9e9811475b58`; compared against actual P029 reviewed baseline `83a494424594b966e69cac529084cec640b7ee09`, not just ancestry against the iOS branch. Full refreshed interval and explicit deferred work: `ios-p030-upstream-ledger.csv`.

## First implemented batch

- `927ac992de0428b415984ef704256e64215ab0a5`: prevent agent-selected IDs from taking over another system's container, systemd, SMART or ZFS records. Conditional upserts retain the owner and reject colliding updates; tests cover each writer, including both ZFS paths.
- `87d61ea2b140f8027b44afe5afc4782a8bc5746e`: sanitize null container/filesystem/ZFS entries before publishing a decoded agent poll; remove null SMART attributes. Existing alert and rollup guards were already equivalent and retained. Upstream tests exercise real CBOR websocket input, rollups and SMART extraction.

PocketBase 0.40.6, CBOR 2.10.0, Shoutrrr, Go 1.27.4/A7 runtime, iOS 12 ARM64, Hub/Agent packaging, `no_self_update`, migration contracts, all 34 locale catalogs, frontend virtualization/pagination and security assertions are retained. No schema or wire-format change is introduced. New optional MCP is deferred as a complete feature/auth review; dependency relocation, frontend concurrency, health-marker and localization follow-ups remain separate ledger entries.

## Qualification

Local and hosted results will be recorded at the committed source checkpoint. Until then new-source host/security/migration/native/provenance qualification is **NOT RUN** as a complete set. P029 results are not reused as P030 proof.

Production modifications **NONE**. No tags, releases, merges, scheduled watcher changes, production database/repository access or device operations. Physical-device validation **DEFERRED — OWNER APPROVED**; device tests **NOT RUN**. This branch is not production-release-ready.
