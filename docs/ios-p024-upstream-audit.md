# P024 upstream delta audit

**Status:** all five pinned commits inspected and integrated by the normal merge below; compatibility implementation and local qualification are complete; exact-final-head hosted qualification remains pending.

**P024 branch base:** `8e03afadb3a03480414fa47f87d40c13331e05b3` (P023 integration)

**Previously integrated upstream:** `b5357343550c8e2493d0e65472a06c1007396384`

**Pinned upstream target:** `b97abf7662a7c7a2dd7542ea2062405abd22d4dc` in `henrygd/beszel`

The target is a descendant of the previous upstream commit by exactly five commits, in the order below. The complete range changes 23 paths. `henrygd/beszel` main currently points to the pinned target; no later upstream commits were observed. P024 will not include commits after this SHA.

| Order | Commit | Patch reviewed | Risk | iOS and compatibility risk |
|---:|---|---|---|---|
| 1 | `ab5d40b84c3aecc1e624f3ea9641d65c9a0322d9` — `fix(agent): re-check HTTPS monitor certificates hourly (#2673)` | `agent/network_monitor_cert.go`, `agent/network_monitor_cert_test.go`, `agent/network_monitor_task.go` | Medium | Cadence changes to one hour for success and failure. Existing behavior retains the last known certificate on failure and skips concurrent calls without holding a lock over I/O. P024 still needs cancellation, replacement, retry-boundary, and bounded-lifecycle tests. The upstream cadence test uses `testing/synctest`; it does not sleep for an actual hour. |
| 2 | `bb8815a008b8010807c9371bdc65659b6b0de7d1` — `New Crowdin updates (#2677)` | Catalan, Brazilian Portuguese, Portuguese, and Simplified Chinese `.po` catalogs | Low | Catalog-only edits. Message identifiers and placeholders remain the source contract. Security, account, token, certificate, and alert wording needs a final catalog check and Lingui compilation. The later Brazilian Portuguese patch cancels this commit's net file changes. |
| 3 | `aa431c5b21c2541455121ef742d3c8fc5a7da071` — `i18n: add catalan and remove untranslated languages` | Lingui locale list, language selector, Romanian/Thai/Uyghur catalogs, and Brazilian Portuguese catalog | Low | Adds Catalan metadata and removes three catalogs. The deleted Romanian file had 117 translated entries among 501; Thai and Uyghur each had 3. None of these locales was offered in `languages.ts` or configured in Lingui at the P023 base, and the files have no P023/iOS-specific modifications. They are stale, unsupported catalogs; keep the upstream removals and do not list those locales. Vietnamese remains present. |
| 4 | `37c0547b02ee44487618cf58d0ec6e8db9a0d22f` — `feat: collect memory and swap in bytes for low memory systems (#2359, #2360)` | Agent collection/tests, system stats contract, Hub migration/tests, record averaging/tests, memory charts, frontend types | High | Adds `MemBytes [4]uint64`, `SwapBytes [2]uint64`, and `MaxMemBytes`; JSON names are `mem`, `swap`, and `memm`, with CBOR integer keys 41 and 42 for the arrays. Deprecated GB fields remain for older Hubs and integrations. The iOS implementation must keep Apple VM page accounting, bound used bytes by total, avoid Linux cache formulas, leave unsupported swap unavailable, and avoid inventing ZFS ARC. Upstream's legacy-float conversion and uint64 averaging need overflow and invalid-input coverage. |
| 5 | `b97abf7662a7c7a2dd7542ea2062405abd22d4dc` — `chore(deps): upgrade go dependencies` | `go.mod`, `go.sum` | High | Updates purego, shoutrrr, x/exp, mimetype, dbus, pprof, Ginkgo, and objx checksums/versions. The upstream tree declares Go 1.27.2; P023's Go 1.27.3 requirement and native x509 fix remain authoritative. Review Apple A7/native build compatibility and notification/networking behavior before qualification. |

## Semantic conflict inventory

1. `agent/memory.go` has an iOS-specific collector in `agent/memory_platform_ios.go` and `agent/ios_memory.go`. It derives `Total` from `hw.memsize`, uses Darwin free plus inactive pages for available memory, bounds used memory, and does not infer swap or cache from those counters. The byte port must preserve those choices and retain the existing legacy GB fields.
2. The upstream protocol uses fixed arrays. A zeroed optional swap array is omitted by the existing `omitzero` tags; P024 must not turn an unsupported iOS source into a visible `[0,0]` measurement. Fixed memory arrays cannot mark individual cache/ARC elements unavailable, so iOS zero values for those slots must be documented and hidden as unsupported in the chart where platform context is available.
3. The upstream Hub and records fallback treats an all-zero byte array as legacy data and converts GB floats. The P024 conversion rejects negative, NaN, and infinite values and saturates out-of-range legacy values. Mixed historical/new records keep units separate, use independent valid sample counts, and avoid uint64 sum overflow with 128-bit accumulation.
4. The upstream frontend reads exact byte arrays first and legacy GB fields second. P024 must preserve that fallback, handle absent/partial data, and avoid rendering unsupported iOS swap or cache/ARC as measured values.
5. `go.mod` conflicts with P023 only on the Go directive and any already-remediated dependency handling. Keep `go 1.27.3`, retain native Go x509 remediation, and adopt pinned dependency updates only after qualification.

## Inherited preservation baseline

P023's merged evidence records the existing 41 upstream commits, all 178 frozen audit rows, 35 historical iOS commits, and 13 logical iOS patchsets as preserved. P024 adds the five commits above, for 46 cumulative post-v0.21 upstream commits. The local matrix and readiness validators pass after the P024 merge: 178/178 frozen rows, 35/35 historical iOS commits, 13/13 logical patchsets, and no unresolved preservation entries. P017 ancestry validation is running on source 01865f223d5c4f744b47b42b4268f4b3ced11d27.

## Normal merge and conflict resolution

Normal merge commit: `e2b677769dd47ff97b5578fb4069372b0f2f79e4`.

Parents are P024 checkpoint `b399faf01065bc89452d2d903cc5edf7d3f3e1bf` and pinned upstream `b97abf7662a7c7a2dd7542ea2062405abd22d4dc`. The upstream parent is genuine and retains all five original commits. The remote feature ref was verified at the merge commit and after each durable checkpoint; its current code head is `01865f223d5c4f744b47b42b4268f4b3ced11d27`.

Merge conflicts were resolved semantically in `agent/memory.go`, `go.sum`, and Catalan `ca.po`; `go.mod` auto-merged. The Agent still emits legacy GB fields, avoids iOS swap and ARC inference, and uses the upstream exact byte arrays. P023's `go 1.27.3` and native x509 disposition were retained. The Catalan translation supplied by upstream replaced one previously blank entry. These statements describe the merge resolution; full qualification remains pending.

Post-merge compatibility work bounds memory values, averages byte counters without uint64 overflow, excludes missing memory/swap samples from record averages, and sanitizes chart values. The full local Go suite, affected Go race suites, frontend tests/build, module verification, asset-manifest tests, and the new all-tier mixed-format database fixture pass. P017, P013, and iOS/Procursus runs on 01865f2 are in progress; final evidence documentation and exact-head reruns are still pending.

## Current remote checkpoint

The P024 feature ref and open Draft PR #28 were last confirmed at `01865f223d5c4f744b47b42b4268f4b3ced11d27`. P017 run `39755571369` and P013 run `39755571381` were in progress at the last status check. iOS Build run `39755571531` passed Procursus validation and had its macOS/Xcode job queued. Previous code head b6c0b5af passed P017 run `39754342581` and P013 run `39754342524`; its native macOS job was canceled by the next feature push and is not a native pass. The required system memory rollup fixture was added at 01865f2 and passes locally under normal and race tests.

The required installer workflow was not yet dispatched for 01865f2. An earlier pass on source edd7e9e is historical only. GitHub's branch-protection endpoint returned HTTP 403 to the connected account; actual review and merge enforcement must not be bypassed.

No production `iOS` branch or fork `main` mutation is part of this audit. No device access, tag, release, or deployment was made.

## Net changed paths (23)

```text
agent/memory.go
agent/memory_linux_test.go
agent/network_monitor_cert.go
agent/network_monitor_cert_test.go
agent/network_monitor_task.go
go.mod
go.sum
internal/entities/system/system.go
internal/hub/systems/system.go
internal/hub/systems/system_test.go
internal/records/records.go
internal/records/records_averaging_test.go
internal/site/lingui.config.ts
internal/site/src/components/routes/system/charts/memory-charts.tsx
internal/site/src/lib/languages.ts
internal/site/src/locales/ca/ca.po
internal/site/src/locales/pt/pt.po
internal/site/src/locales/ro/ro.po (deleted)
internal/site/src/locales/th/th.po (deleted)
internal/site/src/locales/ug/ug.po (deleted)
internal/site/src/locales/zh-CN/zh-CN.po
internal/site/src/types.d.ts
supplemental/CHANGELOG.md
```
