# P022 production difference audit

## Compared refs

- Production `iOS`: `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` (`beszel.Version` 0.19.0).
- P021 integration source: `228b50644a8f1a28c61c5f6944698bc1011d0926` (`beszel.Version` 0.21.0).
- Merge base: production SHA `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`.
- Pinned upstream: `b5357343550c8e2493d0e65472a06c1007396384`, reachable from integration.
- The current upstream `main` had advanced to `b97abf7662a7c7a2dd7542ea2062405abd22d4dc` (five commits ahead of the pin, zero behind); P022 does not add those commits.

The full comparison is 417 commits (394 non-merge commits) and 637 changed path entries: 368 added, 266 modified, two renames, and one deletion. The tree contains 519 production paths and 886 integration paths. Full-tree case-fold checks found no case-sensitive/case-insensitive collisions.

| Scope | Changed paths |
|---|---:|
| `.github` | 12 |
| `agent` | 179 |
| `docs` | 99 |
| `internal` | 283 |
| `scripts` | 12 |
| `supplemental` | 9 |
| `tests` | 25 |
| `tools` | 9 |
| `patches` | 1 |
| Root build/version/config files | 8 |

The machine-readable promotion matrix is split across [ios-p022-production-matrix.csv](ios-p022-production-matrix.csv) and [ios-p022-production-matrix-part-2.csv](ios-p022-production-matrix-part-2.csv), joined by `risk_profile_id` to [ios-p022-promotion-profiles.json](ios-p022-promotion-profiles.json). Together the CSV parts inventory all 637 changed paths or renames; each profile supplies production and candidate behavior, compatibility, upgrade/rollback exposure, test evidence, outstanding validation, and both starting/current status. `PENDING` means cloud or device evidence is still required; it is not a pass.

## Historical source accounting

The read-only P002/P003 validators passed on the exact P021 integration checkout: 178/178 frozen upstream audit rows, 35/35 historical iOS commits, all 13 logical iOS patchsets, zero unresolved rows, 24 acceptance cases, and 33 compatibility scenarios. Git ancestry confirms the pinned upstream commit is an integration ancestor and that 41 upstream commits follow the frozen baseline. The P021 reports also record normal merge ancestry for P019 PR #23, P020 PR #24 and P021 PR #25.

The integration snapshot contains 243 commits after the pinned upstream SHA, representing later iOS work, regression corrections and P019–P021 qualification. The validators' complete inventory remains in the existing P002/P003 files; P022 adds no upstream commit.

## Material release-oriented changes

| Area | Production behavior | Candidate behavior | Compatibility and upgrade considerations | Initial status |
|---|---|---|---|---|
| Version and protocol | Beszel 0.19.0 code and schemas | Beszel code version 0.21.0, with upstream history through pinned `b5357343` | Do not represent later upstream `main` commits as an official stable release; verify exact iOS tag format before any future release | `PENDING` |
| Agent telemetry and transport | v0.19 host collectors and wire contracts | iOS-specific CPU, memory, battery, disk and metadata support plus updated CPU, network, package and storage collectors; WebSocket/SSH compatibility fixes | Preserve old payload compatibility, source selection, collector error semantics and transport lifecycle | `PENDING` |
| Hub, OAuth and authorization | v0.19 PocketBase/API, user roles and universal-token route | Updated API/auth flows, protected record ownership, current monitor/history APIs and safe universal-token POST behavior | Existing users and roles must retain ownership; no token URL or unauthenticated mutation regression | `PENDING` |
| Network monitors and TLS alerts | No integrated network monitor history/API/UI at v0.19 | Monitor probes, persisted history, TLS certificate metadata/alerts and per-monitor HTTPS verification setting defaulting securely | New schema/UI fields must migrate in order; certificate inspection must not weaken active probe verification | `PENDING` |
| Hub schema | v0.19 collections and migrations | v0.20 snapshot and ordered monitor, TLS, ZFS status and socket metadata migrations | The removed `1788955200_pool_updates.go` migration is documented in P002 as folded into the v0.20 snapshot; P013 must prove real v0.19 installs retain `updatable`, ZFS `display_name` and `raw`, users and historical records | `PENDING` |
| Frontend/localization | v0.19 embedded UI and catalogs | Updated dashboard, settings, network monitor UI, localization and production assets | Frontend must build from the frozen lockfile and be embedded in Hub; JS advisories remain separately tracked | `PENDING` |
| iOS build/runtime | v0.19 production binaries and earlier toolchain | Go 1.27.2, isolated A7-safe runtime patch, iOS ARM64 device Mach-O, iOS 12 minimum, embedded frontend, clean VCS metadata and checksums | Hosted cross-compilation does not prove the binary launches on Apple A7; `ldid` pseudo-signing occurs at install time, with no Apple distribution identity | `PENDING` (device) |
| Installer and services | Published-release installer with rootful paths and LaunchDaemons | Immutable tag pinned per transaction, SHA verification, ldid signing, update state, backup and rollback contracts | Keep CLI/binary names, `/var/lib` roots, service labels and component modes; database rollback is separate from binary rollback | `PENDING` |
| CI and publication | Existing v0.19 build and tag workflow | Hosted native validation plus guarded iOS release job, with other upstream release/package automation excluded from iOS tags | At P021 baseline, `workflow_dispatch` on a tag could enter publication and a live release was created before post-publish asset checks. P022 is hardening this boundary; no release job is authorized in this PR | `FAIL` at starting baseline; `PENDING` after P022 correction |
| Container/package files | v0.19 Docker/Helm/Debian metadata | Pinned upstream Docker/Helm/package updates | These are separate channels from the rootful iOS package; iOS suffix tags must remain excluded from upstream publication | `PENDING` |

## Migration and deletion disposition

The only path deletion is `internal/migrations/1788955200_pool_updates.go`. Its `zfs_pools.display_name` and `raw` fields are represented in the `0_collections_snapshot_0_20_0.go` schema snapshot; the upstream change matrix explicitly records that the snapshot absorbs the one-off migration. The v0.19 fixture must demonstrate that the applicable pre-existing fields and records survive the actual v0.19-to-current path. The two migration renames are `0_collections_snapshot_0_19_0.go` to `0_collections_snapshot_0_20_0.go` and `1789079746_container_update_available.go` to `1790193183_network_monitor_cert.go`; migration ordering, not filename similarity, is the release contract.

## Release and physical limits

The fork's published Latest release is still `v0.19.0-ios.1`; the candidate code version is `0.21.0`. The source snapshot contains 41 upstream commits beyond its frozen baseline but no evidence that upstream released a stable version newer than v0.21.0. No iOS release tag, package, deployment, production branch, production database, or device was created or modified for this audit. P018 physical Gate F remains `PENDING`.
