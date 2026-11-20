# P030: PARTIAL — RESUMABLE

P029 received HEAD `7fbed064a0a154d4865103b222665b9f1449c845` was verified before edits; terminal evidence reconciliation is P029 HEAD/base `1f8cac7bd3a7764f559a6aaea56935bd3152147a` (documentation only). P029 is host/native qualified within its implemented scope at source `4fbf4ed27ce0a6e3c921b998031081a1bff70d8e`; see `ios-p029-handoff.md` and the terminal evidence. Artifact `14005295997` is preserved. It does not qualify P030 changes.

P030 branch: `ios/p030-upstream-integration`, stacked on `ios/p029-upstream-integration`. Official `henrygd/beszel` main fetched at `39f9a0765311e83213e5379ea27a9e9811475b58`; compared against actual P029 reviewed baseline `83a494424594b966e69cac529084cec640b7ee09`, not just ancestry against the iOS branch. Full refreshed interval and explicit deferred work: `ios-p030-upstream-ledger.csv`.

## First implemented batch

- `927ac992de0428b415984ef704256e64215ab0a5`: prevent agent-selected IDs from taking over another system's container, systemd, SMART or ZFS records. Conditional upserts retain the owner and reject colliding updates; tests cover each writer, including both ZFS paths.
- `87d61ea2b140f8027b44afe5afc4782a8bc5746e`: sanitize null container/filesystem/ZFS entries before publishing a decoded agent poll; remove null SMART attributes. Existing alert and rollup guards were already equivalent and retained. Upstream tests exercise real CBOR websocket input, rollups and SMART extraction.

PocketBase 0.40.6, CBOR 2.10.0, Shoutrrr, Go 1.27.4/A7 runtime, iOS 12 ARM64, Hub/Agent packaging, `no_self_update`, migration contracts, all 34 locale catalogs, frontend virtualization/pagination and security assertions are retained. No schema or wire-format change is introduced. New optional MCP is deferred as a complete feature/auth review; dependency relocation, frontend concurrency, health-marker and localization follow-ups remain separate ledger entries.

## Qualification

Application/test source checkpoint: `444e61efab68dd27b01780f5c1534816cd57da25`. Draft PR [#34](https://github.com/nghianguyen150612/Beszel-iOS/pull/34) targets `ios/p029-upstream-integration`. Subsequent evidence-only HEADs must be compared with this checkpoint; they do not replace its source provenance. Exact mappings: `ios-p030-commit-map.csv`.

- **PASS — hosted host/security/frontend**, P017 run `43831987596` on the exact checkpoint: full canonical Go `testing no_ui` regressions, portable builds, vet, module verify/tidy, audit validators, frontend tests/build, strict Bun/npm audits, Go vulnerability and workflow scans. Independently downloaded security ZIPs match authenticated GitHub digests: zero Go findings, zero JavaScript findings, zero zizmor findings. Existing x/crypto and x509 security fixes remain verified.
- **PASS — hosted synthetic migrations**, P013 run `43831994187`, direct and via-P027 paths on the exact checkpoint.
- **PASS — native build and Procursus**, run `43832000386` on the exact checkpoint. Independent uploaded-artifact consumer is pending until terminal evidence below. Release jobs must remain skipped.
- **PASS — local Go 1.27.4**: `go test -p 3 -tags='testing no_ui' ./internal/hub/systems ./internal/records ./internal/alerts`; `go vet -p 2 -tags='testing no_ui' ./...`; `go mod verify`; `go mod tidy -diff`. New ownership/CBOR/rollup regressions included. Existing same-system updates retained.
- **PASS — local Bun 1.4.0** frozen install; 100 tests / 399 assertions; genuine production frontend build; all 98 assets match checked manifest `85d3d897b3d28b78c503d03cca02b2f6111ac5df383ce3c21a7c7d8f639ffb08`. No lockfile or locale catalog changed.
- **PASS — local Python/compatibility**: 138 audit-validator tests, 10 migration helper tests, all 45 iOS patch sentinels. Initial local setup failures (wrong Go launcher for two audit tests; concurrent dist regeneration during vet; initial wrong manifest working directory) and successful corrected runs are retained in `evidence/p030-local-evidence.json`.
- **PASS — independently downloaded P017 source manifest**, artifact `14209147735`, ZIP SHA-256 `c058ec973cee6cc2fd2bf100a49c2618d4b665e3c42e0de8c2b4f54996569e25`, exact source checkpoint, 98 assets, text-manifest hash above. This is test evidence, not a native binary.

P029 results are not reused as P030 proof. Full job/step and artifact metadata: `evidence/p030-ci-snapshot.json`; independent host ZIP verification: `evidence/p030-downloaded-host-artifacts.json`; security summaries are retained alongside them.

Production modifications **NONE**. No tags, releases, merges, scheduled watcher changes, production database/repository access or device operations. Physical-device validation **DEFERRED — OWNER APPROVED**; device tests **NOT RUN**. This branch is not production-release-ready.


## Terminal qualification — implemented batch PASS

- **PASS** run [43831987596](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/43831987596), exact source `444e61efab68dd27b01780f5c1534816cd57da25`.
  - `133179258129` candidate-regression: **PASS**.
  - Artifact `p023-workflow-security-evidence`, ID `14210111269`, archive `sha256:68ab8ded4c97abb6078f9736015a6561288975c0200a3158f460e00d5e2f1eb3`.
  - Artifact `p017-regression-manifest`, ID `14209147735`, archive `sha256:c058ec973cee6cc2fd2bf100a49c2618d4b665e3c42e0de8c2b4f54996569e25`.
  - Artifact `p023-javascript-audit-evidence`, ID `14208523808`, archive `sha256:cf9c7003bd70d787a07b3ab7fc643fbdd04f92903ecd2979139d7811bcb39d93`.
  - Artifact `p023-go-vulnerability-evidence`, ID `14208214555`, archive `sha256:cd987818eafefe1289e5d632fbdbb1487094005812c1cc1f2dcb7beb32857ea1`.
- **PASS** run [43831994187](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/43831994187), exact source `444e61efab68dd27b01780f5c1534816cd57da25`.
  - `133179347697` genuine-hub-fixture (direct): **PASS**.
  - `133179348232` genuine-hub-fixture (via-p027): **PASS**.
  - Artifact `p013-hub-recovery-evidence-direct`, ID `14209910014`, archive `sha256:52bf98d2a6554918f8601d0572af0a5b7f9e557cd9ea33b6bebd34fc105867b2`.
  - Artifact `p013-hub-recovery-evidence-via-p027`, ID `14209895232`, archive `sha256:f5ff0fb9a396c0ec3a1fb7d65f566df8c62a3058afc62616f277532f9c532484`.
- **PASS** run [43832000386](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/43832000386), exact source `444e61efab68dd27b01780f5c1534816cd57da25`.
  - `133179298056` build-ios: **PASS**.
  - `133179298469` procursus-package-updates: **PASS**.
  - `133188479177` Verify uploaded iOS artifact bytes: **PASS**.
  - `133189003742` release-ios: **SKIPPED**.
  - `133189004043` release-ios-validate: **SKIPPED**.
  - Artifact `ios-artifact-consumer-verification`, ID `14210945139`, archive `sha256:c9d41b0085337e2897db8e62ab1e4487a7c783dd60b5e11b722bd83891e7ff9b`.
  - Artifact `ios-native-validation`, ID `14210581791`, archive `sha256:f322a25233198b7b5f42d3254099b36cc086c30e8ddca28935a73562e1a84737`.
  - Artifact `ios-wifi-diagnostic-disabled`, ID `14210442206`, archive `sha256:550662f165e1f8d6ae5613c90648ac261e866b9fa710d034c76810803dbdd0ff`.
  - Artifact `beszel-ios-arm64`, ID `14210402117`, archive `sha256:032c5e5334a4793e616c27ebaf3c86f987e16e128bd527b4809ee9c6793d9507`.

**PASS — independent hosted native consumer and local report ZIP verification.** Consumer output `14210945139` was independently downloaded; ZIP SHA-256 `c9d41b0085337e2897db8e62ab1e4487a7c783dd60b5e11b722bd83891e7ff9b` matches GitHub. Full report: `evidence/p030-hosted-artifact-consumer-verification.json`. Source tree `14e55a423d03f31576b2495461b78ea26c590925`.

- `beszel-agent-ios-arm64`: 9526384 bytes, SHA-256 `54398712123f0d11ef4d945f07ed80984596b99b95065286b18d972b86075008`.
- `beszel-hub-ios-arm64`: 32372296 bytes, SHA-256 `60a195861f217df1614f6c57a8f3092a335e9a91a5515a3dab698277e378a4f8`.

Report verifies both same-run payload ZIPs and hashes, clean exact-SHA VCS metadata, Go 1.27.4, iOS 12.0 arm64/v8.0, A7 runtime instructions, x509 fix, no private Wi-Fi linkage, `no_self_update`, 98 independently rebuilt embedded assets and byte-identical isolated-cache repeat builds. Legacy candidate labels do not authorize a release. P029 payload `14005295997` is preserved separately; P030 payload is `14210402117`. No artifact was deleted or substituted.

## Resumed P030 batch — implementation and exact-source host/migration/native qualification PASS

Verified starting branch HEAD: `206724cfe4d2317e6a0bef93f0fbfb23bcc01ade`; verified remote and PR #34 were Draft and still targeted `ios/p029-upstream-integration`. The P029 base remains `1f8cac7bd3a7764f559a6aaea56935bd3152147a`. Container/health implementation source commit: `699f0ffe1778989272c789b32f19171af59f8dd6`. Source follow-up `ac22b6be505d100ce1ab61111f9f7b37a6712b20` refreshes the checked frontend asset manifest for the P030-generated bundles and is the current branch source.

- **PASS — container refresh regression coverage:** 12 deterministic deferred-promise tests cover concurrent systems, scoped cancellation, out-of-order responses, initial-load ordering, re-add/removal, rapid switching, empty/partial data, error/retry, sort/filter, pagination, and cleanup. TanStack virtualization, shared pagination, search, measured system names, mobile layout, and Agent/Hub requests remain intact.
- **PASS — local frontend:** frozen-lockfile install already passed on the unchanged dependency graph; full `bun test` on this source passed 112 tests / 432 assertions. `bun run build` passed (2,526 modules). Lingui extraction only reordered context metadata in the 34 catalogs; generated ordering-only changes were reverted. No locale catalog content changed. The checked frontend asset manifest was intentionally regenerated in `ac22b6be` for the built assets.
- **PASS — local Go 1.27.4:** in addition to the targeted suites, the full canonical `env -u NETWORK GOTOOLCHAIN=go1.27.4 go test -p 2 -tags='testing no_ui' ./... -count=1 -timeout=20m` passed on manifest source `ac22b6be`. The inherited `NETWORK=caas_neko_desktop` environment caused unrelated `internal/cmd/agent` assertions in the first local attempt; the corrected command matches the hosted workflow's environment isolation. `go mod verify`, `go mod tidy -diff`, and `go vet -p 2 -tags='testing no_ui' ./...` passed. Health marker path, fallback and existing Agent/Hub tests are included.
- **PASS — health marker lifecycle tests:** path resolution does not create files; `Check` does not refresh liveness; Linux fallback/newest marker and idempotent cleanup pass. iOS still uses `/tmp/beszel_health`; authenticated `/tmp/beszel_connected` trust and cleanup code was not changed.
- **FAIL — hosted P017 on earlier source:** run `44010182491`, exact source `699f0ffe1778989272c789b32f19171af59f8dd6`. Frontend tests/build (112 / 432), module verification, full vet, dependency and workflow scans passed; the full Go suite failed only `TestEmbeddedFrontendMatchesManifest` because the fresh build's hashed assets differed from the checked manifest. Commit `ac22b6be` regenerated `docs/ios-frontend-assets-manifest.sha256` from the production build; local `internal/site` and full Go suites now pass.
- **PASS — hosted P013 migration fixtures:** run `44010181995`, exact source `699f0ffe1778989272c789b32f19171af59f8dd6`; direct job `133795060732` and via-P027 job `133795061077` both passed. This does not qualify the later manifest correction.
- **CANCELED — hosted native iOS/A7 build:** run `44010182790` on pre-manifest source `699f0ffe`; superseded by the new-source dispatch before a result or artifact was produced. Earlier artifact `14210402117` remains historical evidence.
- **PASS — corrected exact-source P017 host/security/frontend:** run `44013430839`, exact source `ac22b6be505d100ce1ab61111f9f7b37a6712b20`, job `133805582092`. Canonical Go, frontend, migration-related, workflow and dependency regressions passed. Go source vulnerability scan: symbol level, zero findings; OpenPGP reachability guard scanned 394 source files and passed; workflow scan had zero zizmor findings; Bun/npm OSV and audit summaries report no active findings. Asset-manifest validation passed for 98 files with manifest SHA-256 `481d2944eafaaf6d63c075f0ceff398cd088f5e122da1a476fca28ae4c08a2cc`. Artifacts: `p017-regression-manifest` ID `14298072865`, archive SHA-256 `a927c3811896c314e3b9f9be12534aa71918c7b98d6b726319a60df1a18b9190`; `p023-workflow-security-evidence` ID `14297434092`, SHA-256 `52ee7ed312d99d95db2ccf88438eb7ce4b1b79a1535d3d1e3d15a618f5ac9f7f`; `p023-go-vulnerability-evidence` ID `14297254078`, SHA-256 `ea47ccaea72125555750d0cb0abb9d21c05ec3a7578e95a70e7f5523046ae76c`; `p023-javascript-audit-evidence` ID `14297169338`, SHA-256 `dc8ad341576abec95810075272bd55a7ba66c5edcc0b65b62e37a1bee956919f`.
- **PASS — corrected exact-source P013 synthetic migrations:** run `44013430869`, exact source `ac22b6be505d100ce1ab61111f9f7b37a6712b20`; direct job `133805581866` and via-P027 job `133805581440` both passed. Artifacts: direct ID `14298256211`, SHA-256 `7873739c027255f8b5f3fea15605de1e35f4f918253f358ff11da2ee90d1baa1`; via-P027 ID `14297687632`, SHA-256 `09790da431ad7824243bd0be70717c4a0185cfabb82b4ed363252f966b9ba700`.
- **PASS — corrected exact-source native build and Procursus producer:** run `44013430667`, exact source `ac22b6be505d100ce1ab61111f9f7b37a6712b20`; `build-ios` job `133805679400` and Procursus job `133805679774` passed. Producer artifacts: `beszel-ios-arm64` ID `14298977691`, archive SHA-256 `5a80a3ed2e5028bb4ce96a1f5610481a7677ea27a7c6072c5e806e6f222d883b`; `ios-wifi-diagnostic-disabled` ID `14298713984`, SHA-256 `fcb26eee79f83a4959d51752da4aed153dac5a0695ad982fb9205dde7eaee5df`; `ios-native-validation` ID `14299755577`, SHA-256 `e1cd7314b74dcfd2ebdc707ed6f5b631bf093519231140bef2f2bd85cbbd36cd`.
- **PASS — independent uploaded-artifact consumer:** run `44013430667`, job `133816919553`. It independently downloaded the uploaded payload, rebuilt the frontend and verified source `ac22b6be505d100ce1ab61111f9f7b37a6712b20`, source tree `fde98e6f4ca06b7ba4f7a7a015d5f21dffe0005a`, and binary SHA-256 values: Agent `be78b7704bf8ea6556cb66c21918c33e2f7be2cb87d64fd2d6035f21cb96eaa3`; Hub `6773dc7a547dcb96fe9aac92f92ac7ae214b16429923379fec1cb58ef20518c1`. Consumer manifest artifact `ios-artifact-consumer-verification` ID `14299033774`, archive SHA-256 `2ec61707f64343486628a7cb135a88a052991abad6d68ca89a3154d5e91da21a`. Release jobs `release-ios-validate` (`133819925956`) and `release-ios` (`133819926557`) were **SKIPPED**, as intended.

Review decisions: PocketBase relocation commits `d21e60b6afb4ee5a944fbef343f56facf772d275` and `0b51c7262f2d77fd68bc8f9532539bdbcca23415` remain **DEFERRED** after the module audit. The new vanity import path resolves to `https://ganigeorgiev.com/git/pocketbase/pocketbase`, tag `v0.42.2` at `9873ed8ffdf2e7a99bd57e269a33f7cea51620d9`; the module checksum matches upstream `go.sum`. The first commit changes PocketBase/dbx/fexpr/ozzo module paths and updates PocketBase 0.40.6 to 0.42.2 across 86 files; the follow-up corrects one remaining test import. There is no `vendor` tree, so Go module downloads and checksums govern release builds. Both versions contain the same eight bundled migration files; after normalizing import paths and the moved discussion URL, migration logic is unchanged. The new module requires Go 1.27 and upgrades pure-Go modernc SQLite 1.60.1 to 1.62.0 plus related dependencies. Keep 0.40.6 until direct/via-P013/P027 upgrade fixtures qualify those dependency and schema effects on the iOS/A7 build. Existing Lingui chart time labels are already locale-aware across all 34 catalogs; the `Intl.NumberFormat` unit-style refactor is not needed and would add iOS 12 WebKit compatibility risk. Locale refresh batches remain deferred. API-key/MCP work remains a separate security-sensitive feature group and was not partially imported.

Draft PR [#34](https://github.com/nghianguyen150612/Beszel-iOS/pull/34) remains open and targets `ios/p029-upstream-integration`. Production `iOS` remains `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; no tag, release, merge, production change, or watcher change occurred. Device validation is **DEFERRED — OWNER APPROVED**.

**P030: PARTIAL — RESUMABLE.** Beszel source `ac22b6be505d100ce1ab61111f9f7b37a6712b20` now has exact-source host, migration, native and independent-consumer PASS. Next exact action: continue the deferred PocketBase module relocation review with direct/via-P013 migration fixtures and iOS/A7 qualification before changing module paths. Do not count run `44010182491` or historical artifact `14210402117` as qualification for `ac22b6be`.

## November 18 P030 resume

**P030: PARTIAL — RESUMABLE.** Verified starting HEAD `f0bf280e9f336fd2e0a88a3799e3f0e794dca290`, clean local checkout, and open Draft [PR #34](https://github.com/nghianguyen150612/Beszel-iOS/pull/34) against P029 base `1f8cac7bd3a7764f559a6aaea56935bd3152147a`. The last fully qualified source is `ac22b6be505d100ce1ab61111f9f7b37a6712b20`. Artifact `14298977691` belongs to that checkpoint. The retained changed source is `931999def419959d95462d036f1d86f912d67390`.

| Official Beszel reference | SHA | Commit timestamp |
| --- | --- | --- |
| `main` | `3f93a621d6bc6a92f9c2df12b16c4344dc991ac5` | 2026-11-17T15:02:43-05:00 |
| PocketBase relocation | `d21e60b6afb4ee5a944fbef343f56facf772d275` | 2026-11-12T10:48:17-05:00 |
| Relocation follow-up | `0b51c7262f2d77fd68bc8f9532539bdbcca23415` | 2026-11-12T10:51:41-05:00 |
| Portable swap correction | `067af2d93592747581e28d15e4451303dff61fa6` | 2026-11-17T14:12:38-05:00 |

### PocketBase gate

Implementation `d24df76ccdf80ef5f61bc6079646522368ddd3f6` moved all Go imports, including the P013 rollup probe, and extended P013 to exercise pinned P026, P027, P028 and P029 checkpoints. Exact test source `a13030714043006a8ee65c69377094d6a19bec14` restored generated ordering changes in the 34 catalogs before qualification. The application dependency changes in that candidate were:

| Dependency | Established graph | Candidate graph |
| --- | --- | --- |
| PocketBase | `github.com/pocketbase/pocketbase` 0.40.6 | `ganigeorgiev.com/pocketbase/pocketbase` 0.42.2 |
| dbx | `github.com/pocketbase/dbx` 1.12.0 | `ganigeorgiev.com/pocketbase/dbx` 1.13.0 |
| fexpr | `github.com/ganigeorgiev/fexpr` 0.6.0 | `ganigeorgiev.com/fexpr` 0.7.0 |
| ozzo-validation | `github.com/pocketbase/ozzo-validation/v4` 4.3.0 | `ganigeorgiev.com/pocketbase/ozzo-validation/v4` 4.4.1 |
| modernc SQLite / libc | 1.60.1 / 1.77.1 | 1.62.0 / 1.80.1 |
| modernc cc/v4, ccgo/v4, gc/v3 | 4.29.7, 4.36.1, 3.1.5 | 4.30.6-0.20261101114202-1b9d1a076f6b, 4.38.1, 3.2.0 |

The verified module `go.sum` in the candidate preserves the exact upstream archive and module hashes. PocketBase 0.42.2 resolves to tag commit `9873ed8ffdf2e7a99bd57e269a33f7cea51620d9`, with archive checksum `h1:OBrrcVlHD5ZC2ZTFbKr2YkS0mUROGf+TkLbverK7CxQ=`. Its minimum Go version is 1.27. The application used Go 1.27.4 and the existing A7 runtime patches. PocketBase adds a late 401 response for unknown or expired Authorization tokens, OAuth UI corrections, an auxiliary log vacuum threshold, and optional hook event metadata. Existing application authorization and protocol suites passed with the new module.

The supported migration/rollback boundary in the fixture is a verified **cold** checkpoint of the stopped Hub, both databases and related state, followed by startup on a disposable restored clone. PocketBase's online backup issues separate `VACUUM INTO` operations for `data.db` and `auxiliary.db`. Those operations establish per-database snapshots. The online backup path was source-reviewed, and online restore was **NOT RUN**. Recovery after a failed upgrade uses the complete checkpoint and prior binary.

- **PASS** — exact-source Go 1.27.4 full `testing no_ui` suite, full vet, module verify and tidy; bootstrap, auth and authorization, Hub API, WebSocket/CBOR, storage, alert/notification, ownership, health and embedded asset regressions. WAL reader/writer isolation was exercised separately for the main and auxiliary databases. The first local test invocation preceded frontend asset generation and failed to resolve embedded files. The full run after the real frontend build passed. Frozen Bun 1.4.0 install, production build, 112 tests / 432 assertions and all 98 manifest hashes passed after catalog compilation.
- **PASS** — local direct P013 and four hosted P013 jobs in [44224594028](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/44224594028). The redacted evidence confirms fresh initialization, identities/relations, migration ordering and idempotence, API/auth/session behavior, both integrity checks and five synthetic recovery exercises. The direct archive was downloaded independently and matched its digest.
- **FAIL** — hosted via-P029 job `134534546715` in the same run. The pinned historical binary build failed fetching `modernc.org/sqlite@v1.60.1` from `proxy.golang.org` with an HTTP/2 stream `INTERNAL_ERROR`. The job did not open a fixture. **PASS** — the local via-P029 retry on the exact candidate exercised the historical 0.40.6 binary and then the candidate. Session refresh returned HTTP 200, the schema/counts and migration rows were unchanged, main and auxiliary databases passed integrity and foreign-key checks, and all five cold restore cases passed. The first local attempt ran out of disk while compiling the intermediate binary. The retry after reclaiming build-cache space passed. Redacted local manifest SHA-256 `44bcdf86a6e63e0a1d48d8b01760ae85ac44b740604e40a8cdb8c4a0a126aa94`.
- **FAIL** — hosted P017 run [44224597813](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/44224597813), job `134534558637`, at the JavaScript audit step. The npm lockfile OSV scan exited 1 on the active advisory. The Bun and npm reports also identify [GHSA-fc5q-39xc-vjpf](https://github.com/advisories/GHSA-fc5q-39xc-vjpf) in `picomatch` through 4.0.7. No published fixed version was available. The JS archive `14392850617` independently matched SHA-256 `776e98bf5488975f589ac95b8c49330e6a7957f0aa742cd9a841689a92258fc3`. The independently verified Go scan archive `14392501458`, SHA-256 `d3f1a9e3fc19b667523bb9aecb90a9d029e3ff1fc5e25d5db4b08c920e485d55`, contained zero symbol findings. The audit failure skipped later P017 validation and reporting steps. The OpenPGP and Go disposition validators passed locally after making the production frontend files available to the candidate worktree.

| Hosted P013 path | Job | Redacted artifact | Archive SHA-256 |
| --- | --- | --- | --- |
| Direct | `134534546844` | `14392930273` | `9722e7bfb9f7c8b68c74fac87f518e6d5df5da99b1557a1adb53082ebd6f5e5c` |
| Via P026 | `134534546872` | `14392152256` | `933bc71f8662086aca8113da21ecedf4384de06a2e2f46d4cfb528015107bd0c` |
| Via P027 | `134534546500` | `14392117248` | `4ade29b542aa12f21959405f16ba658298a48952ebca9bef9199d8c02d93b8ba` |
| Via P028 | `134534546714` | `14391973688` | `d5ab464c1734907a25382a73f98106d4b03f0e772dc1b8e20a880c642375c0ee` |

**BLOCKED — PocketBase 0.42.2.** Ordinary revert `474458f7e6c3780568a5d1ec3c3ccfd09b9cca5f` restores the exact start tree, including PocketBase 0.40.6, both Go module manifests, and the original migration workflow. The candidate commit and evidence remain in branch history.

### Independent retained implementation

Source `931999def419959d95462d036f1d86f912d67390` adapts upstream swap telemetry correction `067af2d93592747581e28d15e4451303dff61fa6`. When gopsutil supplies no swap total in `VirtualMemory`, non-Linux desktop/server agents can use `SwapMemory`. Existing populated counters remain authoritative. The test verifies macOS, FreeBSD, OpenBSD and Windows fallback, Linux and iOS exclusion, legacy-compatible values, and unavailable-counter handling. iOS still reports swap as unavailable. There is no database schema, CBOR protocol, dependency, embedded asset or locale change.

QNAP-specific Linux ZFS collection is **DEFERRED**. The Homebrew installer changes are **NOT APPLICABLE** to Procursus and iOS. Locale changes and the complete API-key/MCP feature remain **DEFERRED**. A context/message/plural-key comparison of the four upstream catalog updates found 101 catalog entries where upstream fills a currently untranslated, supported key and one change to an existing translation. Validating placeholders and importing the fill-only entries is a separate batch. All 34 catalogs and English fallback are preserved. Physical-device qualification remains **DEFERRED — OWNER APPROVED**.

- **PASS** — Go 1.27.4 full `testing no_ui` suite, full vet, `go mod verify` and `go mod tidy -diff`. Frozen Bun install, production build and 112 tests / 432 assertions passed on the identical frontend tree. All 98 checked embedded-asset hashes matched.
- **PASS** — retained-source P013 run [44225924399](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/44225924399). Direct job `134538958099`, artifact `14393002280`, archive SHA-256 `5dc513252d306286369c59cb8017e848b7ea9069c779aed592ba67e70c44beda`. Via-P027 job `134538957790`, artifact `14393261244`, archive SHA-256 `aefdb61eb65b3830188e093ad743a753bf473752804b6742d8543a9aabd47a83`.
- **FAIL** — retained-source P017 [44225928437](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/44225928437), job `134538970309`. The npm lockfile OSV scan stopped at the same `picomatch` advisory, also present in the Bun/npm reports. Independently downloaded evidence `14392083684` matched archive SHA-256 `1deb94ed07389296ad6ac4adf353d67bcebef6261318b37993975af1bd33347d`. The strict JavaScript gate and lockfiles were preserved. The source-symbol Go scan and local OpenPGP/disposition checks passed with zero findings. Its independently downloaded archive `14392807162` matched SHA-256 `cf91d9331ae5a924d6987bd1f6cb683923f8b388797c15cc98647fa2ec5724c4`. Workflow security scan passed, artifact `14392422975`, SHA-256 `35a8b9f599888671321aa98c176fc9cd6221ff99492f2f0aa99584ecc8cb6e6c`.

### Native evidence

- **PASS** — PocketBase 0.42.2 candidate `a13030714043006a8ee65c69377094d6a19bec14` native iOS 12/A7 and Procursus run [44224601174](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/44224601174). Build `134534571328`, Procursus `134534571100`, and independent artifact consumer `134541371295` passed. Consumer verified the exact candidate source and tree `fc3e20fa4fbcf7e0b357966fa48636d251b12dea`, Agent SHA-256 `450b3d9bdefbd84e40ccd3fd9254a594896f29ecd91c43e13a603121a21ad6bf`, and Hub SHA-256 `3f64f35aa538e5855378ba176232b2469de23dba4b39f394b5920e93010f1302`. Release jobs `134542598885` and `134542599875` were skipped.

| Candidate native artifact | ID | Archive SHA-256 |
| --- | --- | --- |
| Agent and Hub | `14392823351` | `b0bfff48ee74dec759852636a49bc4ca9f3169556131ffed06a4f74afc157599` |
| Native validation | `14393575857` | `3d5bd1d0c1a9ee10ed4e3332436b4aa40affe41ff90de6a3b3dfeaaac172e131` |
| Disabled Wi-Fi diagnostic | `14393306793` | `eab2873b90d54e274770e555ff97ed71eb3969ba73533f1973c33f04d2db2c4a` |
| Independent consumer report | `14392194531` | `c0b66fca06b52197b1640e3838dc5decf12b1afa6e15a18ff3b93a4c52405d06` |

The consumer report ZIP was independently downloaded and matched GitHub's digest. This proves native qualification for the reverted candidate scope. Full qualification stays **BLOCKED** by P017. The retained source `931999def419959d95462d036f1d86f912d67390` has its own native run [44227046274](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/44227046274), dispatched at that SHA.

Next action is to resolve the `picomatch` advisory in the locked Bun and npm graphs with a compatible published fix and rerun P017 with unchanged audit thresholds. Then reapply the candidate by reversing `474458f7e6c3780568a5d1ec3c3ccfd09b9cca5f` in an ordinary commit, run all five P013 paths, and qualify each gate on that exact SHA before retaining the upgrade. **P030: PARTIAL — RESUMABLE.**

### Retained-source native terminal evidence

**PASS** — run [44227046274](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/44227046274), source `931999def419959d95462d036f1d86f912d67390`, build job `134542648175`, Procursus job `134542648499`, independent consumer job `134550143978`. Release jobs `134550918469` and `134550919691` were skipped. The consumer verified source tree `aaff0c8af6665eea0268bb5e754e8c3b3ec895ca`, Agent SHA-256 `81e3dbece83d7225ad3b5648e02c7961338e0441e0bdacb174a09ab749fdc4cf` and Hub SHA-256 `8fdc248b8efd8c46869bfc396d920f2add22f7a5fcaa0312e82cdf86104b610d`.

| Artifact | ID | Archive SHA-256 |
| --- | --- | --- |
| Agent and Hub | `14394691277` | `6e05da06515ed0637f46e60adfa8f05262afa202ad4bc99372dc4b43df6dfb29` |
| Independent consumer | `14394661164` | `e0a4d571376744d4900f00b28803db86db0046e9f39858b238b2880e9271c6ca` |
| Native validation | `14393873323` | `00813235228e7fcc19dcb2b5dd401aa9ad6de59fab2482f9c4304d626a48309e` |
| Disabled Wi-Fi diagnostic | `14394002861` | `bfe7608feb0ea95851eb064ab58087ba58e6ebc55aae70e33dfe0c6eb84eacea` |

The consumer report ZIP was independently downloaded and its digest matched GitHub. The exact-source serial Go suite and portable Agent/Hub builds passed locally, with clean `vcs.revision` metadata. Full qualification remains **BLOCKED** by the strict JavaScript audit; the last fully qualified source remains `ac22b6be505d100ce1ab61111f9f7b37a6712b20`.

## Picomatch recovery — November 18, 08:18 UTC

Starting HEAD `ef18986282894ee0b5f0dd3b1d0e4707844b2354` matched origin and Draft PR #34 targeted P029. Prior changes after source `931999def419959d95462d036f1d86f912d67390` were documentation only. Last fully qualified source remains `ac22b6be505d100ce1ab61111f9f7b37a6712b20`.

**BLOCKED** — [GHSA-fc5q-39xc-vjpf](https://github.com/advisories/GHSA-fc5q-39xc-vjpf) was updated 2026-11-17T15:56:44Z and remains active, affecting versions <=4.0.7 with no patched version. [OSV](https://api.osv.dev/v1/vulns/GHSA-fc5q-39xc-vjpf) was modified 2026-11-17T16:00:03Z and lists introduced 0 through last affected 4.0.7. [NVD CVE-2026-112366](https://nvd.nist.gov/vuln/detail/CVE-2026-112366) remains awaiting analysis. npm reports latest 4.0.7. The referenced upstream PR #194 is open and its own description says `+(*a)` remains exponential. PR #270 proposes a relevant single-branch correction and PR #246 targets v2; both remain open and unmerged. A correction and compatibility across all glob forms and both major versions have not been independently established. No maintained compatible replacement covering the dependency APIs and all consumers has been established.

Bun 1.4.0 lock SHA-256 `bb6e22344f87c9e34463f78e2b5985585bae036272adaee670f27f77b23d07df`; npm lock SHA-256 `306d9e0df497ab7582a1abde4ff1bfe8428b520727202e6941ea569ddfc0c1c2`. The npm lock and Bun lock include 2.3.2 under `@lingui/cli -> micromatch`, `@lingui/cli -> chokidar -> anymatch` and `chokidar -> readdirp`, and 4.0.4 under `vite -> picomatch` and `vite -> tinyglobby -> picomatch` / `fdir`. npm lock marks all four copies dev-only. Build tooling uses repository configuration and file patterns. No direct application source import of picomatch or micromatch was found. The review does not establish the behavior of every transitive dependency for attacker-controlled build inputs. Keep builds isolated and restrict untrusted repository/configuration inputs; this is containment, not advisory clearance.

**PASS** — independent Linux change `1f1ced78b18078b5ee8c2229b9a0e2c11ed963f3` adapts upstream `c3b60db0d461da49bc502a274c63dd666edfab58` (2026-11-17T14:29:38-05:00), using the QNAP `/proc/lpl/kstat/zfs` path when the standard path is missing. Synthetic tests cover ARC size, pool counters and standard-path precedence. Go 1.27.4 `go test -p 2 -tags='testing no_ui' ./agent/...` and corresponding vet passed. No dependency, protocol or iOS collector changes. The required full P017 audit remains **BLOCKED** by the active advisory. The prior failed P017 run 44225928437 remains evidence; no threshold or advisory disposition changed. PocketBase remains 0.40.6 pending security clearance.

Hosted qualification for source `1f1ced78b18078b5ee8c2229b9a0e2c11ed963f3` completed as P013 `44265680064`, P017 `44265684612` and iOS `44265689094`. P013 and native are PASS (job and artifact details in “Terminal qualification for retained QNAP source”); the original P017 failed on the active advisory and remains preserved. Next action: obtain an official verified fix for both affected major lines, update both locks, inspect every installed copy, run unchanged strict audits and full frontend/native qualification, then recover the PocketBase candidate and rerun its full migration matrix.

### Qualification update for QNAP source

- **PASS** — local Go 1.27.4 full `testing no_ui` suite, full vet, module verify and tidy on `1f1ced78b18078b5ee8c2229b9a0e2c11ed963f3`.
- **FAIL** — P017 run `44265684612`, job `134669507863`, at the unchanged npm lock OSV gate, exit 1 for GHSA-fc5q-39xc-vjpf. Frozen install, frontend tests and production build, workflow scan, Go scan, vet and module checks passed before that step. JavaScript audit artifact `14410421922`, archive SHA-256 `d905f08aa7a012a87a3501918bffac039031ff9f4e726342f0e8d982ed78f69c`; Go scan artifact `14410162612`, SHA-256 `c12b6a66db588eadcb8e2cd54ade4602fead5d52a0877411fb509ee2f2171f7a`; workflow scan artifact `14410531693`, SHA-256 `98483c8fd5f387121483a60c5edeed04f4c30e89a07e88b6d5109af419fd2a74`.
- **PASS** — P013 direct job `134669497426`, artifact `14410891615`, archive SHA-256 `3d1b9acf24fcc48ad9b2cf0d198cfe60cbed065ce42c8eaaf9060b0b08e04c24`.
- **FAIL** — P013 via-P027 job `134669497185` failed before opening the fixture when `proxy.golang.org` returned HTTP/2 INTERNAL_ERROR downloading `modernc.org/sqlite@v1.60.1`. Job retry `134676221022` was requested without changing the source.
- **PASS** — Procursus job `134669522604` in native run `44265689094`. Native build and independent consumer require their own terminal results.

### November 19 advisory revision

At 2026-11-19 01:58 UTC GitHub and OSV reported GHSA-fc5q-39xc-vjpf withdrawn at 2026-11-18T15:38:56Z. The [maintainer explanation](https://github.com/micromatch/picomatch/pull/235#issuecomment-6315749381) recommends against accepting untrusted glob patterns. No patched package was released. NVD continues to list CVE-2026-112366 as awaiting analysis, with a last modification timestamp of 2026-10-19T21:13:39.973Z. The advisory's withdrawal does not establish that the reported worst-case behavior was corrected.

**PASS** — unchanged `npm audit --audit-level=low --json --package-lock-only` and `bun audit --json --audit-level low` returned exit 0 locally on the existing locks. Hosted P017 run `44440922693`, job `135281267501`, on documentation checkpoint `93d82c6ed6c6e0ff29a18b6062d62e176a5e3028` completed PASS. Its application and workflow tree matches source `1f1ced78b18078b5ee8c2229b9a0e2c11ed963f3` apart from the three P030 documentation files. Frontend audit artifact `14490952465`, SHA-256 `64ddd0823dd5daf20414afda8ed1f0e8c1f6f979090783b5285be3835bd9229b`; Go scan `14490398265`, SHA-256 `a0c339e5019faec358fc3ae52af9433b397ced56c09e2ce5072723985891dd5c`; workflow scan `14490004411`, SHA-256 `3cc2e00be79f5801953858017bda98cd52c8e7d3a923bf1ce99ba28b70705949`.

**PASS** — P013 via-P027 retry job `134676221022` on source `1f1ced78b18078b5ee8c2229b9a0e2c11ed963f3`, run `44265680064`, artifact `14411975348`, SHA-256 `050bd958f0c450be3821f91073f74aff8826e9de332860a8d76e124526258083`. The initial download failure remains recorded above. **PASS** — native run `44265689094`, build `134669522891`, Procursus `134669522604`, independent consumer `134684368294`. Agent/Hub artifact `14411254465`, SHA-256 `f57769617496f1125fc645636184d13dc7f428302f2273044038e0d15ba405c4`; consumer report `14411199970`, SHA-256 `deb9fb7e9d4b6201fd5b2d03d622ecb1d254c042c4c22a5333d3abf0f78a3725`.

The user required a verified security remediation before recovering PocketBase. The package behavior has no verified fix and NVD remains open, so PocketBase recovery is **BLOCKED** despite the audit gate passing after withdrawal. Continue to restrict untrusted glob input and reassess a verified maintained fix or formal security disposition before applying the candidate. Physical-device work remains **DEFERRED — OWNER APPROVED**.


## November 19 P030 security and PocketBase recovery

### Starting state and retained source

At recovery the P030 branch and origin were both `b10bd565d5bd5a653bf07062e9b25fd51ee0edc4`, with a clean worktree. Retained application source is `1f1ced78b18078b5ee8c2229b9a0e2c11ed963f3`; this update changes documentation only. Draft PR [#34](https://github.com/nghianguyen150612/Beszel-iOS/pull/34) remains open against `ios/p029-upstream-integration`.

The most recent P017 security qualification is **PASS** in run [44440922693](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/44440922693), job `135281267501`, on documentation-only source `93d82c6ed6c6e0ff29a18b6062d62e176a5e3028`. Its application, workflow and lockfile tree matches retained source `1f1ced78...`; no application-source changes separate them. The unchanged strict npm and Bun audit commands returned exit 0. Frozen install, tests, production build, asset/security checks and Go checks completed. This does not establish that picomatch code was corrected.

### Terminal qualification for retained QNAP source

The terminal results were checked against GitHub on 2026-11-19. Application source `1f1ced78b18078b5ee8c2229b9a0e2c11ed963f3` has:

- **PASS** — P013 run [44265680064](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/44265680064): direct job `134676224454`, artifact `14410891615`, SHA-256 `3d1b9acf24fcc48ad9b2cf0d198cfe60cbed065ce42c8eaaf9060b0b08e04c24`; via-P027 job `134676221022`, artifact `14411975348`, SHA-256 `050bd958f0c450be3821f91073f74aff8826e9de332860a8d76e124526258083`.
- **PASS** — native run [44265689094](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/44265689094): build `134669522891`, Procursus `134669522604`, independent consumer `134684368294`. Agent/Hub artifact `14411254465`, SHA-256 `f57769617496f1125fc645636184d13dc7f428302f2273044038e0d15ba405c4`; consumer report `14411199970`, SHA-256 `deb9fb7e9d4b6201fd5b2d03d622ecb1d254c042c4c22a5333d3abf0f78a3725`; native validation artifact `14411453937`, SHA-256 `1179d204bc4c8c389e45b7493a494b25d22d8efed1eb3b78a23d942390b74f78`; Wi-Fi diagnostic artifact `14411648268`, SHA-256 `0ba13cf0be5c610004e3133b506c8bdae36409e234fabbd3941bd3b4b06f9851`.
- **PASS, application-tree equivalent** — P017 run [44440922693](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/44440922693), job `135281267501`, on documentation-only `93d82c6ed6c6e0ff29a18b6062d62e176a5e3028`; frontend/config/workflow/lockfile and application tree match `1f1ced78...`. The earlier P017 failure on exact `1f1ced78...` remains preserved because it occurred before advisory withdrawal.

### P030-A — picomatch disposition

Evidence was refreshed at **2026-11-19 08:22 UTC**:

- GitHub marks [GHSA-fc5q-39xc-vjpf](https://github.com/advisories/GHSA-fc5q-39xc-vjpf) withdrawn at `2026-11-18T15:38:56Z`; its record still names CVE-2026-112366, severity High, affected versions through 4.0.7 and no first patched release.
- [OSV](https://osv.dev/vulnerability/GHSA-fc5q-39xc-vjpf) was modified at `2026-11-18T15:56:12Z`, records the withdrawal and affected versions from 0 through 4.0.7.
- [NVD](https://nvd.nist.gov/vuln/detail/CVE-2026-112366) still reports “Awaiting Analysis”; its last-modified field is `2026-10-19T21:13:39.973Z`.
- npm registry latest remains picomatch `4.0.7`, published `2026-08-24T14:36:23.690Z`. The upstream `master` and `v2` references had open, unmerged fix proposals (#270 and #246); proposal #235 is closed without merge. No official fixed release or maintained drop-in replacement was verified. Withdrawal is not a code fix.
- A resource-bounded check compiled `+(*a)` and matched 28 `a` characters followed by `!` in isolated Node subprocesses. Installed picomatch 2.3.2 and 4.0.4 each consumed the 2-second CPU limit and were terminated (exit -9). A first attempt with a 512 MiB address-space cap failed before execution because Node/V8 could not reserve its segmented table; it is not counted as behavior evidence. No unbounded experiment was run.

Exact Beszel lock hashes: `internal/site/bun.lock` SHA-256 `bb6e22344f87c9e34463f78e2b5985585bae036272adaee670f27f77b23d07df`; `internal/site/package-lock.json` SHA-256 `306d9e0df497ab7582a1abde4ff1bfe8428b520727202e6941ea569ddfc0c1c2`. Installed copies are picomatch 2.3.2 under micromatch, anymatch and readdirp, and 4.0.4 at the Vite dependency root; the paths are frontend development/build dependencies. The app source has no picomatch/micromatch import, built frontend assets contain no picomatch code, and no public app endpoint was found that accepts a glob. Lingui locale extraction and Vite/content scans use repository-controlled patterns. A hostile change to checked-out build configuration or source could still consume CPU on a CI runner.

- **AUDIT STATUS — PASS:** unchanged strict checks pass after advisory withdrawal; previous failing audit evidence remains preserved.
- **PATCH STATUS — NOT PATCHED:** no verified release or merged fix; installed copies remain within the withdrawn range.
- **RUNTIME EXPOSURE — NOT APPLICABLE:** picomatch is absent from deployed frontend bundles and the runtime is the Go Hub/Agent.
- **BUILD-TIME EXPOSURE — PRESENT, CONSTRAINED:** glob parsing is used by repository-controlled build tooling; no untrusted runtime glob source was identified.
- **RESIDUAL RISK — PRESENT:** the bounded test reproduced CPU exhaustion on installed 2.3.2 and 4.0.4; latest 4.0.7 has no verified correction.
- **ACCEPTANCE STATUS — BLOCKED:** no authorized project acceptance of the residual behavior is recorded. Preserve this gate; do not label the component patched or accepted-risk.

Sources: GitHub Advisory API, OSV API, NVD CVE API, npm registry metadata, [picomatch upstream](https://github.com/micromatch/picomatch), [upstream PR #270](https://github.com/micromatch/picomatch/pull/270), [upstream PR #246](https://github.com/micromatch/picomatch/pull/246) and [maintainer explanation on #235](https://github.com/micromatch/picomatch/pull/235). API and package-registry facts above were retrieved 2026-11-19 08:22 UTC.

### P030-C — PocketBase 0.42.2 candidate

The main P030 branch remains on PocketBase 0.40.6. Candidate `a13030714043006a8ee65c69377094d6a19bec14` remains isolated in `/workspace/scratch/beszel-pocketbase-candidate`; the ordinary revert `474458f7e6c3780568a5d1ec3c3ccfd09b9cca5f` remains part of branch history. The candidate graph is the relocated PocketBase 0.42.2 module, dbx 1.13.0, fexpr 0.7.0, ozzo validation 4.4.1, modernc SQLite 1.62.0/libc 1.80.1 and its recorded transitive upgrades. Candidate tag commit and checksums are recorded above in the PocketBase gate section.

**Migration engineering: PASS on the exact candidate, with the hosted P029 transport failure retained.** Prior exact-candidate P013 evidence covers direct, via-P026, via-P027 and via-P028 hosted jobs in run [44224594028](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/44224594028); the synthetic local via-P029 retry also passed. It verified session refresh, schema and migration-history preservation, records/relationships, main and auxiliary database integrity and foreign keys, idempotence/restarts and five cold recovery cases. Candidate Go 1.27.4 tests/vet, module verification/tidy, Hub auth/API/WebSocket/CBOR/hooks and native/A7 consumer qualification are recorded above.

**Dependency transport diagnosis: PASS.** The old hosted via-P029 job `134534546715` failed before opening the fixture when proxy.golang.org returned HTTP/2 `INTERNAL_ERROR` fetching modernc.org/sqlite v1.60.1. A bounded retry using the normal verified Go proxy and checksum database downloaded that exact module successfully into an isolated cache; Go verified module sum `h1:/blz53O951KWFOso4QQvEs/Fq6cDBKLtMVrYNSeJVKw=` and go.mod sum `h1:1dIoEagfDE72QytD5scH1lxARtaUgKgHC/NuApA27r0=`, from upstream v1.60.1 tag `b122d0417c01508beb55158faedeb64a0c5bfd8a`. This classifies the prior failure as transient proxy transport, not a missing artifact, checksum mismatch or database incompatibility.

A new exact-candidate via-P029 harness attempt on 2026-11-19 rebuilt the historical and candidate frontends, then stopped compiling the candidate Hub with `no space left on device` before opening the fixture. **NOT RUN** for that retry; it is an environment failure, not a migration failure. Candidate `go mod verify` and `go mod tidy -diff` passed afterward. No source or production database was changed.

The supported recovery boundary remains a cold, coordinated checkpoint of the stopped Hub including both databases and related state. PocketBase's online backup takes separate `VACUUM INTO` snapshots of `data.db` and `auxiliary.db`; those are not an atomic cross-database point. Online restore remains **NOT RUN**. No production credentials or data were used.

**Integration decision: BLOCKED.** The independent Go migration evidence does not clear the picomatch owner-acceptance gate. Keep 0.40.6 on the main branch until an official corrective release or authorized security disposition exists; then reproduce the hosted via-P029 path and run the full exact-source P017/native/consumer qualification on the integrated tree.

### Current action and status

- **PASS** — retained-source P017 on the unchanged app/config/lock tree; strict audit, Go scan, workflow scan and frontend qualification.
- **PASS** — isolated candidate migration engineering on the five P013 paths using prior exact-candidate results; hosted P029 proxy failure is before fixture, local retry passed.
- **FAIL (environment, before fixture)** — latest local P029 candidate rebuild stopped for disk exhaustion; no migration assertion ran.
- **BLOCKED** — PocketBase candidate integration pending the unresolved picomatch acceptance decision.
- **DEFERRED** — physical-device testing remains OWNER APPROVED; no device operations performed.
- **P030: PARTIAL — RESUMABLE.**

Exact P017 evidence: JS artifact `14490952465`, SHA-256 `64ddd0823dd5daf20414afda8ed1f0e8c1f6f979090783b5285be3835bd9229b`; Go scan `14490398265`, SHA-256 `a0c339e5019faec358fc3ae52af9433b397ced56c09e2ce5072723985891dd5c`; workflow scan `14490004411`, SHA-256 `3cc2e00be79f5801953858017bda98cd52c8e7d3a923bf1ce99ba28b70705949`. Bun 1.4.0 and Go 1.27.4 were used. No new P013 or native workflow was started because the main source did not change and disk space was insufficient.

## Resume checkpoint — 2026-11-20

Starting HEAD `3cc0c25b187b24d4015108e84d4114dc1fc2836c` matched origin and Draft PR #34, base `ios/p029-upstream-integration`. Diff from qualified retained source `1f1ced78b18078b5ee8c2229b9a0e2c11ed963f3` comprises only the three P030 documentation files; application and workflow evidence above remains applicable. Fresh workspace had 7.5 GiB available initially and 7.2 GiB after both checkouts; no evidence or caches were deleted. Hosted isolated execution avoids duplicating local module and frontend caches.

Exact candidate `a13030714043006a8ee65c69377094d6a19bec14` P013 run [44224594028](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/44224594028) was rerun for the previously failed job, attempt 2, via-P029 job `136066005105`. **IN PROGRESS at checkpoint**; inspect the terminal result, logs and synthetic fixture artifact before asserting hosted PASS. Prior local via-P029 PASS and hosted direct/via-P026/via-P027/via-P028 PASS remain documented above. Earlier hosted transport and subsequent local disk failures occurred before fixture execution. Main branch retains PocketBase 0.40.6; candidate integration **BLOCKED — OWNER SECURITY ACCEPTANCE**. Do not integrate solely on migration success.

Picomatch refresh: GitHub advisory remains withdrawn; npm latest and GitHub latest release remain 4.0.7; proposed fixes #270 and #246 remain open. **AUDIT: PASS** (prior strict hosted result); **PACKAGE FIX: NOT VERIFIED; RUNTIME EXPOSURE: NOT IDENTIFIED; BUILD-TIME EXPOSURE: PRESENT, CONSTRAINED; RESIDUAL RISK: PRESENT; OWNER ACCEPTANCE: NOT GRANTED.** Repository-owned patterns and deployed-runtime boundary are documented above. CI job timeouts and permissions constrain execution, but do not eliminate ReDoS risk or establish safety of every untrusted build input. Owner decision requested: accept the documented residual build-time risk or require a verified compatible remediation. This request is not acceptance or release clearance.

**P030: PARTIAL — RESUMABLE.** Next: inspect attempt 2 via-P029 and its fixture evidence, record exact outcome and artifact hash, and update PR #34. Preserve the candidate separately; no merge, release, deployment or physical-device claim. Device qualification remains **DEFERRED — OWNER APPROVED**.
