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

## Resumed P030 batch — implementation and exact-source migration/security PASS; native build in progress

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
- **QUEUED — independent artifact consumer:** run `44013430667`, consumer job `133816919553`. It must verify uploaded bytes, payload SHA-256, source provenance and ARM64/iOS compatibility before native artifact provenance is marked PASS. Release jobs remain skipped as intended.

Review decisions: PocketBase relocation commits `d21e60b6afb4ee5a944fbef343f56facf772d275` and `0b51c7262f2d77fd68bc8f9532539bdbcca23415` remain **DEFERRED** after the module audit. The new vanity import path resolves to `https://ganigeorgiev.com/git/pocketbase/pocketbase`, tag `v0.42.2` at `9873ed8ffdf2e7a99bd57e269a33f7cea51620d9`; the module checksum matches upstream `go.sum`. The first commit changes PocketBase/dbx/fexpr/ozzo module paths and updates PocketBase 0.40.6 to 0.42.2 across 86 files; the follow-up corrects one remaining test import. There is no `vendor` tree, so Go module downloads and checksums govern release builds. Both versions contain the same eight bundled migration files; after normalizing import paths and the moved discussion URL, migration logic is unchanged. The new module requires Go 1.27 and upgrades pure-Go modernc SQLite 1.60.1 to 1.62.0 plus related dependencies. Keep 0.40.6 until direct/via-P013/P027 upgrade fixtures qualify those dependency and schema effects on the iOS/A7 build. Existing Lingui chart time labels are already locale-aware across all 34 catalogs; the `Intl.NumberFormat` unit-style refactor is not needed and would add iOS 12 WebKit compatibility risk. Locale refresh batches remain deferred. API-key/MCP work remains a separate security-sensitive feature group and was not partially imported.

Draft PR [#34](https://github.com/nghianguyen150612/Beszel-iOS/pull/34) remains open and targets `ios/p029-upstream-integration`. Production `iOS` remains `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; no tag, release, merge, production change, or watcher change occurred. Device validation is **DEFERRED — OWNER APPROVED**.

**P030: PARTIAL — RESUMABLE.** Next exact action: wait for consumer job `133816919553` in run `44013430667`; capture its terminal result and binary SHA-256 values, then commit and push this evidence and update PR #34. Do not count run `44010182491` or historical artifact `14210402117` as qualification for `ac22b6be`.
