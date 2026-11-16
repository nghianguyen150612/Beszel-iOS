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

**P030: PARTIAL — RESUMABLE.** The implemented two-fix batch is host/native qualified; remaining ledger work is not qualified or claimed ported. Next source action: inspect `9a02a92a0582382bad1557cb0f088230fc84d873` and `9afebd3f0285401a2ea7f88ed2db9b0d6fb2e927`, port container refresh isolation/order with concurrency regressions while preserving pagination/virtualization; then dispatch P013, P017 and iOS Build on the new source SHA. Health-marker `ee043e72`, dependency relocation, locale and optional feature reviews remain explicit ledger follow-ups. No scheduled watcher changes. Device validation remains **DEFERRED — OWNER APPROVED**.
