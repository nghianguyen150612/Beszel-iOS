# Upstream v0.21 change matrix audit (P002)

## Scope and result

This audit covers the pinned upstream range from the iOS fork point through the
frozen v0.21 target. The canonical one-row-per-commit inventory is
[`upstream-v021-change-matrix.csv`](upstream-v021-change-matrix.csv); the
historical iOS compatibility inventory is
[`upstream-v021-ios-patch-map.csv`](upstream-v021-ios-patch-map.csv). These CSV
files are authoritative; this report summarizes the findings and does not
repeat the 178 rows.

**Audit coverage: 178/178 upstream commits reviewed; 35/35 historical iOS
commits mapped; 0 unresolved matrix rows.** No upstream source was merged and
no production Agent or Hub code was changed.

## Frozen Git references and method

| Reference | SHA / result |
| --- | --- |
| Starting `origin/iOS` / checkout | `14abe465c6eb23b484b8beaa2f0d62c67ac6fe20` |
| Frozen `origin/main` and `upstream/main` target | `bb23f99b3eeaced9608e11e280e7c986dba650be` |
| Merge-base / upstream fork point | `f204dc17e64bba444478c60210d018e43a777193` |
| P001 historical iOS snapshot | `a6b73365147fc8c3a820552a145d370418e5b680` |
| Peeled `v0.20.0` target | `a0dc19eacf6902d2f6934a3efd7ea7277198c40b` |
| Peeled `v0.21.0` target | `ec96c134660c998f7eb044e349e3715b65733fe1` |

Both release tag targets are ancestors of the frozen upstream target. The
comparison range is exactly `f204dc17e64bba444478c60210d018e43a777193..bb23f99b3eeaced9608e11e280e7c986dba650be`.
`git rev-list --reverse --topo-order` returned 178 commits. Every commit's
metadata, parent count, first-parent patch and changed paths were reviewed;
commit-level findings are in the CSV. There are 178 normal commits and no true
merge commits in this range. The subject “Merge commit from fork” at
`50f6fc075d7af0482287acb5c61fbc8a88578335` has one parent, so it is correctly
classified as a normal commit. No merge-resolution diff was omitted.

The P001 comparison `f204dc17..a6b73365` contains 35 historical iOS commits.
The current `origin/iOS` is 37 commits beyond the fork point because it also
contains the two P001 documentation/PR commits; those are not counted as
historical compatibility patches. Tag membership was assigned from Git
ancestry, not dates or subjects. Batch sets are disjoint and their union is
exactly the 178-commit target range.

| Integration batch | Ancestry range | Commits |
| --- | --- | ---: |
| P004 | Fork point through `v0.20.0` | 25 |
| P005 | After `v0.20.0` through `v0.21.0` | 116 |
| P006 | After `v0.21.0` through `bb23f99b3` | 37 |
| **Total** | **Frozen range** | **178** |

The CSV uses semicolons for multiple controlled values, includes every
first-parent changed path as evidence, and marks all 178 rows `Reviewed`.
The standard-library validator reconstructs the range from the frozen SHAs,
checks batch ancestry and exact SHA coverage, and checks the 35-row iOS patch
inventory and patch-ID references.

## Classification counts

Priority and change-type counts are exclusive and each sums to 178.

| Priority | Commits |
| --- | ---: |
| P0 | 21 |
| P1 | 74 |
| P2 | 83 |
| none | 0 |

| Change type | Commits |
| --- | ---: |
| fix | 102 |
| Feature | 28 |
| refactor | 13 |
| security | 6 |
| dependency | 6 |
| test | 8 |
| documentation | 10 |
| CI | 2 |
| other | 3 |

| iOS applicability | Commits | Meaning used in this audit |
| --- | ---: | --- |
| Direct | 13 | iOS build/platform behavior or a changed iOS-owned file/contract |
| shared | 40 | Shared Agent, protocol, or data behavior used by iOS and remote Agents |
| Hub-only | 96 | Hub/API/database/frontend path, including telemetry from remote Agents |
| platform-other | 24 | Platform-specific runtime/tooling outside the iOS Agent target |
| none | 5 | No direct runtime or Hub behavior, such as supporting test/documentation work |

Subsystem counts below are overlapping commit counts: a commit may affect
several areas. Broad counts are followed by selected subsystem tags used in
the canonical `areas` field.

| Broad area | Commits |
| --- | ---: |
| Agent | 63 |
| Protocol | 32 |
| Hub | 43 |
| Database | 12 |
| Frontend | 79 |
| Platform | 8 |
| CI | 4 |
| Tests | 97 |
| Documentation | 20 |
| Dependencies | 12 |
| Installer | 0 |

| Subsystem | Commits | Subsystem | Commits |
| --- | ---: | --- | ---: |
| Agent: CPU/RAM | 5 | Agent: Cgroups | 2 |
| Agent: Disk I/O | 14 | Agent: Network | 7 |
| Agent: Wi-Fi RSSI | 7 | Agent: Package updates | 5 |
| Agent: NVMe SMART | 7 | Agent: ZFS | 6 |
| Agent: GPU | 7 | Agent: Systemd | 6 |
| Agent: Containers | 6 | Agent: Polling/errors | 13 |
| Protocol: WebSocket | 18 | Protocol: SSH | 8 |
| Protocol: TLS | 2 | Protocol: CBOR/JSON | 22 |
| Protocol: Headers/auth | 2 | Hub: Authorization | 8 |
| Hub: API | 12 | Hub: Sessions/users | 3 |
| Hub: Polling | 27 | Hub: Alerts | 10 |
| Hub: Monitor records | 15 | Database: Migrations | 7 |
| Database: History | 4 | Database: Records | 20 |
| Frontend: Dashboard | 18 | Frontend: Network monitors | 21 |
| Frontend: Charts | 18 | Frontend: Alerts UI | 5 |
| Frontend: Package updates UI | 6 | Frontend: Wi-Fi charts | 6 |
| Frontend: Logs | 3 | Frontend: Settings | 4 |
| Frontend: Localization | 13 | Frontend: Dependencies | 6 |
| Platform: Linux | 16 | Platform: Windows | 4 |
| Platform: macOS | 2 | Platform: Docker/release | 6 |
| Dependencies: Go | 6 | Dependencies: Bun/npm | 6 |

The `areas` values also record platform/iOS, embedded-assets, history, and
other more specific tags where present. The broad and detailed counts are not
additive.

## P0 review

These patches change authentication/authorization, cross-system data isolation,
migration state, serialization, or connection lifecycle. The table records the
invariant that a later integration must preserve; file paths are taken from
the reviewed diffs.

| Upstream commit | Reviewed invariant and evidence |
| --- | --- |
| `50f6fc075d7af0482287acb5c61fbc8a88578335` | First-user creation checks the one-time bootstrap state and creates user/superuser records in one transaction, so concurrent bootstrap or partial failure cannot leave a half-created account. `internal/users/users.go` |
| `4bf70700f20cc2994774995dca5e725c7173996b` | `TRUSTED_PROXY_IPS` can constrain `TRUSTED_AUTH_HEADER` by parsed IP/CIDR. Unset/empty remains unrestricted; a configured but wholly invalid allowlist trusts nobody. `internal/hub/api.go` |
| `f7fd3ef403085b64dc8a21a02011105b2251333a` | Newly OAuth-created users start as `user`; existing users retain their role. `internal/users/users.go`, `internal/hub/hub.go` |
| `b0bc72741126b2eadaf284e803827261eac82d54` | The v0.20 collection snapshot absorbs the container `updatable` field and removes two migrations already folded into it. Verify upgrade ordering and preserve existing collection fields. `internal/migrations/0_collections_snapshot_0_20_0.go` |
| `187dc886a96f7124090f7e5a258c9c720bd2cb2c` | SSH reconnect resynchronizes monitor configuration and retries failed sync without failing regular stats collection. `internal/hub/systems/network_monitors.go`, `system.go` |
| `8bf6917fe07ce0a33c9ae29a70bcea45410b0a7e` | Slow pool collection is cached/refreshed under a mutex; Hub data deadline handling no longer turns a slow response into a reconnect loop. `agent/storage_pool.go`, `internal/hub/systems/system.go` |
| `24ec18a937b1b18d606eef2f56728230326557e6` | Each Agent response decodes into a fresh struct, avoiding stale omitted CBOR fields and shared realtime/regular-fetch mutation. `internal/hub/systems/system.go`, `internal/hub/transport/transport.go` |
| `3f20ecae503117ff052061136b080b8f8cce2df9` | Connection state is synchronized and asynchronous WebSocket handshake failures do not stop reconnect attempts. `agent/connection_manager.go` |
| `b684ac9910652bdc39b61982b079fb5213dcc07a` | SSH handshake has a deadline that is cleared after successful handshake, bounding a silent peer's effect on the Hub updater. `internal/hub/systems/system.go` |
| `09a277f07487c1e78a82eb331b0e8dcc4906e753` | System status reads/writes use synchronized accessors after publication, including alert transitions. `internal/hub/systems/system.go`, `system_manager.go` |
| `921ded1af0f955ddaba1dabafe0d361f1d12fb7d` | WebSocket connection lifecycle access is synchronized/atomic across goroutines. `internal/hub/ws/ws.go` |
| `a77ed345d76daf7676dbf2d7b68873ffdd240537` | A reporting system may update only monitor records it owns; history is written only after ownership is confirmed. `internal/hub/systems/system.go` |
| `0484c54919ff071e21e1db3197060fc8277213e0` | Alert subscription, delivery, history, and evaluation paths check the user's current system access. User/system ownership fields are immutable through normal updates; Hub startup reapplies owner rules after the collection snapshot. `internal/alerts/*`, `internal/hub/collections.go`, migration snapshot |
| `24881fbfdf3feba4567e41dc011ea283716873ee` | Systemd detail responses are `private` and vary by `Authorization`, preventing shared proxy-cache disclosure. `internal/hub/api.go` |
| `97db8bd199b10f90ce854ddc913491d43fda9216` | SSH request cancellation closes the in-flight connection/session and serializes transport state. `internal/hub/transport/ssh.go` |
| `1997984325de5057bee0bd5b309fa534dac5d639` | `SSHTransport` is the single owner of a system's SSH connection; polling, monitor sync, and ZFS share it and reconnect once. `internal/hub/transport/ssh.go`, `internal/hub/systems/system.go` |
| `4ffd83677d18fe285ce1647da9d64ed4b2798f40` | Agent WebSocket/SSH fallback, listener and connection state handle retries and shutdown without leaving a stuck service. `agent/client.go`, `connection_manager.go`, `server.go` |
| `36c14e903d73b226a2df5bbf7ab4fbf1943c2906` | Agent avoids sending a duplicate stats payload to legacy Hub versions. `agent/server.go` |
| `7305eda972cb423bd51ac18f4966312b346360d0` | Hub keepalive pings let a slow but active Agent survive between polls. `internal/hub/systems/system.go`, `internal/hub/ws/ws.go` |
| `d91f06f3c533b231346dc1d0ca5bf0bca265e462` | Keepalive stops after three minutes of Agent silence so stale connections are not kept alive indefinitely. `internal/hub/systems/system.go`, `internal/hub/ws/ws.go` |
| `f8d3fe1817b2ac977c051243d6ffc9077a000ba3` | Scalar JSON fields use `omitzero` under json v2; verify response shape under both encoders. CBOR tags are unchanged. `internal/entities/*`, `internal/hub/server.go` |

Other high-risk but non-P0 behavior is called out in P1: certificate-expiry
inspection, the opt-in TLS verification bypass, DNS monitor resolver settings,
and `HUB_HEADERS`. `d89f844a2a0abc7474555f0635344245456f82d6` adds an explicit
per-monitor `skipTlsVerify` option that defaults false and applies only to HTTP
probes; do not broaden its scope. `a99fe5e99725edf2a8ff582f5b21793fc88731f2`
uses a separate unverified TLS dial only to read certificate metadata; the
actual HTTPS probe retains normal certificate verification.
`d324ccc0977d0347e4f94529490fefbc72e54900` parses custom `HUB_HEADERS`,
rejects malformed names/values without echoing secret values, and ignores
reserved `X-Token`/`X-Beszel` overrides; preserve the normal authentication
handshake.

## P1 features and compatibility changes

* `bb1b39928e317592159e65dbcac1de12510d49b8` adds network monitoring end to
  end: Agent probes and CBOR sync, Hub monitor/history/alert storage, migrations,
  retention, and embedded frontend views. Review it as one schema/protocol/UI
  feature; the later ownership and authorization fixes above are part of its
  safety contract.
* `86ab0fae8b7b2d87251a90a4fa56138d17cca553` and follow-ups add per-interface
  Wi-Fi RSSI, compact history and charts. The upstream Darwin collector uses
  `/usr/bin/osascript`; iOS shares Darwin build selection, but does not provide
  macOS AppleScript. There is no native iOS Wi-Fi collector in this range. P003
  must route the Darwin collector away from iOS or provide an explicit supported
  no-op; do not claim iOS RSSI support from this code.
* `c25408651f076c59c13f35f27016461ada8a9093` adds pending package-update
  counts; `8613cfe548c882b57c5748efd7007ecbe2ba2553` adds full update rows
  through a request/API and system-page table;
  `bc2278e7e9820825ce990fb2fe7fca4b9fe9a1c3`
  adds xbps parsing. Preserve the message/version compatibility and parser
  behavior for remote Linux Agents.
* `7626e5e3c84806a67d9e10c8791a6aca9991d2ce` adds systemd service logs through a new Agent action and Hub
  endpoint. The iOS-hosted Hub still needs to consume logs from supported
  remote Agents even though iOS itself has no systemd collector.
* Network-monitor additions include a configurable DNS resolver
  (`f7528a02082a6fd2a54857470e40ff5a54782801`), HTTPS certificate expiry
  (`a99fe5e99725edf2a8ff582f5b21793fc88731f2`), the explicit TLS-check option
  (`d89f844a2a0abc7474555f0635344245456f82d6`), and compare/view/chart changes. Their
  migrations, IDs, polling intervals and alert history must remain compatible.
* `017ebb091aa64596f9545c89e67f6c880f8bb121`, `32d469285382521bae4ad3f368a7045f32ff2f5e`,
  and `bb23f99b3eeaced9608e11e280e7c986dba650be` carry ZFS status and
  root-dataset capacity corrections through Agent, Hub persistence and UI.
  NVMe SMART parsing/width/rescan fixes and GPU JSON/power fallback changes
  alter shared telemetry and need fixture parity, even where the collector is
  platform-specific.
* User view/language settings, package-update UI, log timestamp controls, and
  chart lifecycle fixes affect the embedded frontend. The Hub builds that same
  frontend on iOS, so regenerate `dist` with the pinned Bun workflow.

## P2 platform-specific and supporting changes

Linux-only LXC/cgroup and nl80211 behavior, Windows Wi-Fi adapter labels,
systemd availability checks and journal collection, Docker/Helm packaging,
OpenWrt installer behavior, noncritical chart/layout changes, translation
catalogs, test-fixture portability and changelog updates are recorded as P2 or
`none` applicability as appropriate. They do not create native iOS collectors.
Some still affect the data or UI an iOS-hosted Hub receives from remote Linux,
Windows or macOS Agents; those rows are marked `Hub-only` or `shared`, not
discarded as iOS no-ops.

The dependency commits `195afc10b8924ef4d6afc5d409eb8a8e1c43b96c` and
`e30e980854408d46b2e8837a14a0c0d771a18545` were checked against the code paths
and the pinned lockfiles. The P001 lock resolves valibot 1.3.1 and Vite 7.1.11;
the current advisory database reports valibot `<=1.4.1` and several Vite
advisories through 7.3.4. The app calls Valibot `parse`/`safeParse`, not the
affected `flatten()` helper; the CI production path runs `vite build`, not the
Vite development server. The frozen target resolves valibot 1.5.0 and Vite
7.3.7, for which the audit endpoint returned no direct advisories. Five other
findings remain at the target (`@lingui/cli`, `@lingui/vite-plugin`, `braces`,
`chokidar`, and `micromatch`) and belong in P014 dependency triage. These facts support a P2 dependency-hardening classification; they do
not prove an exploitable production path or a clean whole dependency graph.

## Cross-subsystem dependency relationships

| Change surface | Depends on / feeds | Integration consequence |
| --- | --- | --- |
| Agent collectors and monitor tasks | `internal/common` CBOR/WebSocket actions and `internal/entities` models | Preserve field tags, optional values, retry/timeout semantics and old-Agent behavior. |
| Hub polling and SSH transport | Agent handshake, monitor sync, SMART/ZFS/systemd responses | Keep one transport owner, cancellation, retry and reconnect resync behavior. |
| Hub records and migrations | Monitor, alert, package-update, ZFS and system fields | Check snapshot/migration order, defaults, indexes, ownership rules and stored history together. |
| Frontend and embedded assets | Hub API shapes, PocketBase records and `internal/site/dist` | Rebuild under frozen Bun and check API compatibility before either iOS Go binary is linked. |
| Remote platform collectors | Common Hub entities/history/UI | Linux/Windows/macOS collector support still matters to telemetry shown by the iOS Hub. |

## Existing iOS patch risk

All 35 historical iOS commits are mapped to 13 logical patch IDs in the patch
CSV. Eleven upstream commits have a changed-path intersection with an iOS
commit; these are review points, not proven line conflicts:

| Shared path | Upstream commits | iOS patch to preserve |
| --- | --- | --- |
| `agent/system.go` | `86ab0fae8b7b2d87251a90a4fa56138d17cca553`, `16e3fbadce2026d5591aa08c6518e1c5754fa1a2`, `4ffd83677d18fe285ce1647da9d64ed4b2798f40`, `01d91728f00d234f68cc6ebd2ab7df09226a2789`, `ef840e0e6d8f066419a351c25312acb2d5323af7` | `IOS-AGENT-SYSTEM-HOOK`: Darwin CPU bypass and `adjustPlatformSystemDetails()` |
| `.github/workflows/docker-images.yml` | `97de1471d7ddb92d3138d9baaecb1f58db0bc06b`, `7748651f5cb3ee0ccd3d3e6141f315ee5d20454e` | `IOS-RELEASE-ISOLATION`: exclude `v*-ios.*` from generic Docker publishing |
| `readme.md` | `e68372dce417fe922c0a27aded5ce4e4bfb66cf3`, `4a5915b141bfa96627f5493e5174ec8bc36734d5` | `IOS-PORT-DOCUMENTATION`: keep port-specific user/contributor content |
| `internal/site/bun.lock` | `1ab38e57c59c470ae579caa268b7d9dcdbda95ba`, `e30e980854408d46b2e8837a14a0c0d771a18545` | `IOS-FRONTEND-BUILD`: retain Bun 1.4.0 and frozen-install compatibility |

The 35-commit map separately accounts for the A7 runtime source patch, iOS
build tags/CGO and iPhoneOS SDK workflow, `AppleARMPMUCharger` battery path,
sysctl/plist metadata, shared Agent hooks, Hub startup/migrations, frontend
embedding, LaunchDaemons, installer state and rollback, checksum pinning,
release assets, tests and documentation. The upstream range does not modify
the iOS installer engine, battery files, A7 patch script or release asset
names. The `agent/system.go` overlap does require preserving both
`runtime.GOOS != "ios"` and `adjustPlatformSystemDetails()`.

## P003–P018 handoff

The repository contains P001/P002 context but does not define official titles
or scope for P003–P018. The P007–P017 labels below are provisional routing
buckets inferred from changed paths and the requested roadmap; reconcile them
with the roadmap before implementation.

| Prompt | Handoff |
| --- | --- |
| P003 | Build the compatibility matrix, risk register, rollback strategy, and iOS 12/A7 acceptance criteria. Start with the 21 P0 invariants and Wi-Fi/Darwin build-tag risk. |
| P004 | Review/integrate the 25 commits through v0.20.0, preserving every upstream commit after the original fork point. |
| P005 | Review/integrate the next 116 commits through v0.21.0. |
| P006 | Review/integrate the final 37 commits through `bb23f99b3`. |
| P007 | Agent collectors, platform/build tags, Wi-Fi, package updates, SMART, ZFS and GPU metrics. |
| P008 | WebSocket, SSH fallback, headers, TLS, serialization and connection lifecycle. |
| P009 | Hub API, auth, sessions, alerts and system ownership. |
| P010 | PocketBase snapshots/migrations, records, history and data compatibility. |
| P011 | Network-monitor model, polling, charts and monitor UI. |
| P012 | Package-update, systemd and log telemetry. |
| P013 | Dashboard, settings, charts, alerts UI and localization behavior. |
| P014 | Go/Bun/npm dependencies, TypeScript and embedded frontend build. |
| P015 | Regression fixtures, audit validator and CI gates. |
| P016 | iOS release isolation, build workflow, installer contracts and checksums. |
| P017 | English/Vietnamese docs, change notes and patch-map reconciliation. |
| P018 | Integrated acceptance and release-candidate validation after implementation; not part of P002. |

P002 stops at this handoff. No P003 implementation or upstream integration was
started.

## Validation, unknowns and limitations

The deterministic validator is
[`scripts/audit/validate-upstream-v021-matrix.py`](../scripts/audit/validate-upstream-v021-matrix.py);
its synthetic negative-case tests are
[`tests/audit/test_validate_upstream_v021_matrix.py`](../tests/audit/test_validate_upstream_v021_matrix.py).
Detailed command output and exit status are recorded in
[`upstream-v021-test-results.txt`](upstream-v021-test-results.txt).

`Unresolved audit rows: 0`. Remaining handoff questions are the absent official
P003–P018 prompt definitions, whether the iOS Agent should explicitly omit the
upstream macOS Wi-Fi collector, and five remaining dependency advisory
findings at the frozen target. The audit ran in a Linux cloud workspace, where
the available `go` command is a non-Go wrapper (`go version` exits 1 with
`Go: Unknown option: version`) and Bun/`xcodebuild` are not installed. To cover
those local environment limits, the GitHub Actions `iOS Build` workflow was
manually dispatched on source commit
`60a70591e290d7ea0f9cfe4181ec6cc37788e8dd` (run:
[38358768044](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38358768044)).
It passed Go and Bun setup, frozen frontend install/build, the iOS ARM64 Agent
and Hub builds, iOS 12 Mach-O checks, checksum generation/verification, and
artifact upload. The `release-ios` job was skipped because the ref was a branch,
not a release tag. This workflow-dispatch run is separate from the PR status
rollup, which reported no checks because audit file paths do not match the PR
workflow filters. The PR reports `MERGEABLE`/`CLEAN`; the branch-protection API
returned 403 to this token, so protection settings could not be read directly.
No Go unit-test suite or physical iPad test is claimed. No release, deployment,
tag, or upstream merge was performed as part of the audit work. Any P002
publication to `iOS` is through the requested PR merge only; no direct push was
made to `main` or `iOS`.
