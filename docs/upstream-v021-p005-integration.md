# Beszel upstream v0.21.0 integration — P005

**Result: P005 source integration PASS.** P005 integrates the pinned upstream v0.21.0 history into the approved integration branch. This is not final iOS acceptance or release approval. Gate B remains pending P006.

## Git history and batch coverage

| Reference | SHA |
| --- | --- |
| Starting integration branch | `11432ec410a6cb03707789b7a871e7985e18d6fe` |
| Frozen product `main` | `bb23f99b3eeaced9608e11e280e7c986dba650be` |
| P004 v0.20.0 target | `a0dc19eacf6902d2f6934a3efd7ea7277198c40b` |
| P005 v0.21.0 target | `ec96c134660c998f7eb044e349e3715b65733fe1` |
| P005 upstream merge commit | `dbe5ce029555ef4ca209092689e3dad9ab5fc7ea` |
| P005 code candidate used by the first hosted runs | `65167c09f9612d8256e0fd39830e5b1d935b8d36` |

The merge command was `git merge --no-ff --no-commit ec96c134660c998f7eb044e349e3715b65733fe1`. The merge commit has the expected parents, in order: integration `11432ec410a6cb03707789b7a871e7985e18d6fe`, then upstream v0.21.0 `ec96c134660c998f7eb044e349e3715b65733fe1`.

The canonical matrix’s exact P005 set matches all 116 commits in `v0.20.0..v0.21.0`, and both endpoints are ancestors of the candidate. P004’s 25 commits remain present. Cumulative history is **141/178**; the 37 commits from v0.21.0 through the frozen final target remain for P006. No P006 commit is included. The JSON record contains the exact 116 SHA values and the pre-merge snapshot tied to the starting integration SHA.

P004’s integration reports and preservation record were left intact.

The merge changed 256 upstream paths. Separate follow-up commits are `1082869c54866a8020c04d33e5d992df6caa48b8` (iOS collector build-tag and localized README reconciliation) and `65167c09f9612d8256e0fd39830e5b1d935b8d36` (status-lock regression test and Bun test-only `localStorage` setup). The Bun setup repairs the test environment; it does not change product runtime code or weaken assertions.

## Merge conflicts and resolutions

1. `internal/entities/systemd/systemd_test.go`: the upstream test now uses `testing/synctest`; the P004 fixture also accounts for host CPU count. The upstream test was retained after confirming its timing fixture still scales with `runtime.NumCPU()` and the assertions remain present.
2. `readme.md`: upstream’s lowercase README conflicts with the product’s established English/Vietnamese README files. The duplicate lowercase file was removed. The localized files retain iOS product claims and now describe upstream monitor history only as Hub results from supported remote Agents; they do not claim local iOS network probing.

No blanket ours/theirs resolution was used. The 12 P0 source paths were reviewed after merge; no security control was weakened.

## Upstream changes retained

The merge brings the v0.21 Agent collectors and models for per-interface Wi-Fi telemetry, package updates, GPU and NVMe SMART data, ZFS, systemd, Docker, and network counters. The Hub gains the v0.21 monitor, alert, history, and authorization work; synchronized WebSocket and SSH ownership; fresh response decoding; cancellation and timeout handling; and the corresponding dashboard, charts, package-update, logs, localization, and alert UI changes. The upstream dependency and Helm updates are included.

The migration tree retains upstream identifiers and order. P005 modifies `internal/migrations/0_collections_snapshot_0_20_0.go` and adds `1790193183_network_monitor_cert.go` followed by `1790273174_network_monitor_server.go`. The full Go suite exercises disposable application test databases, but the dedicated seeded 0.19-to-0.21 upgrade, backup, restore, and interrupted-upgrade procedure was not run. No real `/var/lib/beszel-hub` database was opened or modified.

## P0 security and correctness review

All 12 P005 P0 commits are present. The matching source changes and focused tests passed; the P0 packages also passed the race run.

| Upstream SHA | Required invariant | Preservation check and focused test |
| --- | --- | --- |
| `8bf6917` | Slow Agent collections do not cause reconnect loops | Background cache refresh and bounded WebSocket collection; `TestStaleUtilityCachesRefreshInBackground`, `TestFetchDataTimeoutKeepsWebSocketOpen` — PASS |
| `24ec18a` | Each Agent response decodes into fresh data | Omitted fields do not reuse stale values; `TestFetchDataViaWebSocketDoesNotRetainOmittedFields` — PASS |
| `3f20eca` | Async WebSocket handshake failures allow reconnect | Reconnect ticker survives stale disconnect; `TestConnectionManager_TickerSurvivesStaleDisconnect` — PASS |
| `b684ac9` | SSH handshake has a bounded deadline | Silent peer timeout; `TestConnectHandshakeTimeout` — PASS |
| `09a277f` | Shared system status reads and writes are synchronized | Getter/swap locking reviewed; `TestSystemStatusAccessSynchronized` — PASS under race detector |
| `921ded1` | WebSocket lifecycle access is synchronized | Atomic connection lifecycle and concurrent close; `TestWsConnConcurrentClose` — PASS under race detector |
| `a77ed34` | Monitor results are written only by their owning system | Ownership enforcement; `TestNetworkMonitorResultOwnership` — PASS |
| `0484c54` | Alert access, subscriptions, and delivery use current authorization | Record, foreign-system delivery, subscription, revoked access, and monitor access tests — PASS |
| `24881fb` | Systemd proxy details cannot be shared through an auth-insensitive cache | Private cache policy and `Vary: Authorization`; `TestSystemdInfoCachePolicy` — PASS |
| `97db8bd` | SSH cancellation closes the correct pending request/session | `TestSSHRequestCancellation` and shared-connection cancellation — PASS under race detector |
| `1997984` | Each system has one coordinated SSH transport owner | Connection initialization/ownership and concurrent request tests — PASS under race detector |
| `4ffd836` | Agent WebSocket/SSH fallback and shutdown cannot remain stuck | Fallback, reconnect, and shutdown lifecycle tests — PASS |

The full P0 test and race package results are recorded in `upstream-v021-p005-preservation.json`.

## iOS patch preservation and platform behavior

All 13 logical iOS patches remain accounted for. The A7 runtime patch, iOS build workflow, iOS battery parser, installer lifecycle, release routing, upstream-sync checks, and product-specific build settings remain intact. The shared `agent/system.go` change retains the iOS CPU-probe exclusion and `adjustPlatformSystemDetails()` hook; `system_platform_ios.go`, `AppleARMPMUCharger` collection, and the Darwin-only macOS battery source remain correctly selected.

P005’s macOS Wi-Fi collector invokes `/usr/bin/osascript`, and its GPU collectors invoke macOS tools. The follow-up repair gates both collectors to macOS and selects safe no-op iOS fallbacks. `GOOS=ios GOARCH=arm64 go list` selects `wifi_unsupported.go` and `gpu_darwin_unsupported.go`; Darwin selects the upstream collectors. Missing iOS Wi-Fi RSSI remains absent rather than represented as zero. Remote monitor results/history are available through the shared Hub models. Native iOS RSSI remains owned by P012. Native ICMP permissions and iOS monitor probing remain unvalidated and are not claimed as supported.

The generic release and Docker workflows still exclude `v*-ios.*`; installer paths and the pinned installer engine hash are unchanged. No release tag, release, or deployment was created.

| Logical iOS patch | Owner | P005 status |
| --- | --- | --- |
| `IOS-A7-RUNTIME` | P007 | PRESERVED; Go 1.27.1 positive and fail-closed fixtures passed |
| `IOS-AGENT-BATTERY` | P008 | PRESERVED; iOS parser and Darwin exclusion retained |
| `IOS-AGENT-BUILD-PROBE` | P007 | PRESERVED; hosted Agent/Hub build passed |
| `IOS-AGENT-SYSTEM-HOOK` | P008 | RECONCILED; upstream `system.go` changes retain iOS CPU guard and platform hook |
| `IOS-FRONTEND-BUILD` | P015 | UPSTREAM UPDATE VALIDATED; frozen Bun 1.4.0 install and genuine build passed |
| `IOS-HUB-BUILD-PIPELINE` | P015 | PRESERVED; frontend embed and hosted build passed |
| `IOS-INSTALLER-ARCH-DETECTION` | P016 | PRESERVED; installer checks passed |
| `IOS-INSTALLER-BOOTSTRAP-INTEGRITY` | P016 | PRESERVED; SHA-pinned bootstrap checks passed |
| `IOS-INSTALLER-LIFECYCLE` | P016 | PRESERVED; 814 lifecycle checks passed |
| `IOS-INSTALLER-VALIDATION` | P017 | PRESERVED; local and hosted installer suites passed |
| `IOS-PORT-DOCUMENTATION` | P017 | RECONCILED; localized READMEs retained through the README conflict |
| `IOS-RELEASE-ISOLATION` | P016 | PRESERVED; generic release and Docker iOS-tag exclusions remain |
| `IOS-UPSTREAM-SYNC-CI` | P017 | PRESERVED; sync tests and patch sentinels passed |

## Validation

| Check | Result |
| --- | --- |
| Matrix and readiness validators | PASS; exact P005 set 116/116, 13 logical patches mapped |
| Audit unit tests | PASS; 14 tests |
| Canonical Go suite, `env -u NETWORK go test -tags 'testing no_ui' ./...` | PASS |
| Focused P0 tests and Go race run | PASS across Agent, Hub systems, WebSocket, SSH transport, and alerts |
| Linux Agent and Hub builds | PASS |
| iOS/macOS Go build-tag selection | PASS; iOS fallbacks and Darwin collectors selected as intended |
| A7 runtime patch fixtures | PASS; known Go 1.27.1 shape patches, four unknown-shape cases fail closed without modifying the fixture |
| Bun frozen install | PASS with Bun 1.4.0; lockfile unchanged by installation |
| Frontend tests | PASS; 44 tests, 149 expectations |
| Frontend production build | PASS; genuine generated assets; mixed-import and large-chunk warnings were emitted |
| Upstream sync tests / iOS patch sentinels | PASS; 10 / 44 checks |
| Bootstrap / installer tests | PASS; 54 / 814 checks |
| `git diff --check` | PASS |

Hosted checks ran on code candidate `65167c09f9612d8256e0fd39830e5b1d935b8d36`:

- [iOS Build — run 38421092219](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38421092219): **success**. Go 1.27.1, Xcode 15.4/iPhoneOS SDK 17.5; Agent and Hub artifacts are Mach-O arm64 with minimum iOS 12.0 metadata; checksums verified. The `release-ios` job was skipped for this feature branch.
- [iOS installer lifecycle — run 38421092500](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38421092500): **success**. Bootstrap checksum, installer lifecycle, patch sentinels, and sync tests passed.

Hosted builds establish target architecture and deployment metadata; they do not establish that the binaries run without SIGILL on a physical A7 device.

## Deferred obligations and gate status

| Work | Owner | Status |
| --- | --- | --- |
| Integrate remaining 37 frozen upstream commits | P006 | Pending; P006 was not started |
| Device metrics, battery, native iOS runtime, and A7 SIGILL validation | P007, P008, P018 | Deferred; no physical iPad was available |
| iOS transport compatibility and native monitor/ICMP validation | P009, P010, P017, P018 | Deferred; cross-version combinations and native ICMP permissions were not established here |
| Procursus package updates and native Wi-Fi RSSI adapter | P011, P012 | Deferred; current iOS fallbacks remain safe |
| Seeded 0.19-to-0.21 database upgrade, consistent backup, full restore, and interruption recovery | P013, P017 | Deferred; no real database was touched |
| Browser-level API, login, and permission behavior; dependency advisory triage | P014, P015, P017 | Deferred beyond unit tests and production build |
| Physical-device service/reboot validation and release authorization | P018 | Deferred |

**P005 integration gate: PASS.** **Gate B: 141/178 commits integrated; pending P006.** This is source integration only. Device, database-restore, and runtime obligations remain release-blocking. The product `iOS` branch and frozen `main` were not modified; no release or deployment occurred.
