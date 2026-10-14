# P006 — Complete frozen Beszel upstream integration

**P006 source integration gate: PASS. Gate B: PASS.** The frozen upstream history is fully integrated; this is not release acceptance. Gates C–F and P007–P018 remain required. No release or deployment occurred, and P007 was not started.

## Git provenance and exact coverage

| Reference | Full SHA |
| --- | --- |
| Starting integration | `26a6447801fe66f59eddec376f15cb6d0b94aed9` |
| Frozen upstream target / main | `bb23f99b3eeaced9608e11e280e7c986dba650be` |
| Upstream merge | `211bdf5c131328f1a6bc691ecb1547de7e9f68c9` |
| Merge parent 1 | `26a6447801fe66f59eddec376f15cb6d0b94aed9` |
| Merge parent 2 | `bb23f99b3eeaced9608e11e280e7c986dba650be` |
| Tested source candidate | `15bc4226eac9f1dd8a9c5d0dd6de5a4ccd2966f5` |
| Unchanged product iOS | `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` |

The cloud clone used `git clone --branch iOS --single-branch` into an isolated `/workspace/p006-cloud/Beszel-iOS` checkout, preserving the preexisting workspace checkout. Starting refs matched the requested SHAs. P004/P005 records and the P005 merge/follow-up PRs #6/#7 were inspected before source changes. `TARGET..INTEGRATION` contained 57 integration-side commits; these are not the upstream-only count. `INTEGRATION..TARGET` and the exact P002 P006 SHA set both contained 37 commits.

Working branch: `ios/merge-upstream-snapshot`; PR [#8](https://github.com/nghianguyen150612/Beszel-iOS/pull/8), base `ios/upstream-v021-integration`. Executed:

```sh
git merge --no-ff --no-commit bb23f99b3eeaced9608e11e280e7c986dba650be
git commit -m "merge(ios): integrate remaining frozen upstream changes"
```

Both merge parents are verified above. Separate regression commits `fd6eb7f69bf6240f36d20c0a3ad7f397d45035fd` and `15bc4226eac9f1dd8a9c5d0dd6de5a4ccd2966f5` add P006 tests and fixture license/attribution; production adaptations were unnecessary. The evidence is committed separately with `docs(ios): record completed upstream snapshot integration`.

**P004 25/25; P005 116/116; P006 37/37; total 178/178; P006 P0 4/4; iOS patches 13/13.**

Every one of the canonical CSV's 178 full SHAs is programmatically reachable. The fork-to-v0.20, v0.20-to-v0.21 and v0.21-to-target ranges contain 25, 116 and 37 commits. All frozen endpoints and P004/P005 merges/evidence remain ancestors. The JSON contains every SHA, canonical CSV row, changed file, priority, P003 owner and reachability result. Source-version strings are not used as history proof.

## Conflict and source-tree review

**Textual conflicts: none.** No blanket strategy or manual source replacement was used. The automatic merges in Docker workflow, `agent/system.go`, Bun lock and both Helm chart metadata files were reviewed semantically:

- Docker keeps upstream's six build-once image variants and both registries, with the iOS `!v*-ios.*` exclusion retained. Generic release isolation and the dedicated iOS workflow remain intact.
- Cgroup CPU/memory opt-in, host counter rollback and memory refactoring coexist with the iOS CPU-probe bypass and `adjustPlatformSystemDetails()`. Actual `GOOS=ios GOARCH=arm64 CGO_ENABLED=1 go list` selects non-Linux CPU/memory fallbacks, iOS metadata/battery, and safe GPU/Wi-Fi fallbacks. Linux cgroups do not become an iOS dependency.
- Bun 1.4.0 frozen installation leaves lock SHA256 `16643e3228f9b0fac1e5134d89782c096f49e8071489cc2a4f977f3565f02afa` unchanged. The inherited jiti hoisting differs from upstream; all 501 package/version entries match the frozen target. `package.json`, npm lock and Go dependencies match the target exactly.
- Helm chart package versions remain **0.1.8**; appVersion advances to **0.21.0**. Lint/render/package and unpublished-version checks passed. Publishing was skipped; the P005 0.1.7 collision did not recur.

The complete target-tree difference inventory is in JSON, with a reason and Git blob IDs for every path. Shared auth/ownership/history/transport production code equals upstream. Differences in shared production code are limited to the established iOS battery, platform metadata, macOS collector exclusions and absent fallbacks; build/release/installer differences implement the established iOS contracts. Other inherited differences are status-lock tests, Bun test setup, whitespace cleanup, product docs and audit fixtures. No upstream production feature was discarded.

## Thirteen logical patches and four P0 invariants

The pre-merge snapshot records all 13 patch surfaces, source SHA256 values, build settings, workflow text and installer pin at the starting integration SHA. Post-merge hashes and individual reasoning are in [the preservation record](upstream-v021-p006-preservation.json). P006 reassesses the dispositions: ten PRESERVED, three RECONCILED WITH TEST EVIDENCE, zero BLOCKED.

| Logical patch | Canonical owner | P006 disposition |
| --- | --- | --- |
| `IOS-A7-RUNTIME` | P007 | PRESERVED |
| `IOS-AGENT-BATTERY` | P008 | PRESERVED |
| `IOS-AGENT-BUILD-PROBE` | P007 | PRESERVED |
| `IOS-AGENT-SYSTEM-HOOK` | P008 | RECONCILED WITH TEST EVIDENCE |
| `IOS-FRONTEND-BUILD` | P015 | RECONCILED WITH TEST EVIDENCE |
| `IOS-HUB-BUILD-PIPELINE` | P015 | PRESERVED |
| `IOS-INSTALLER-ARCH-DETECTION` | P016 | PRESERVED |
| `IOS-INSTALLER-BOOTSTRAP-INTEGRITY` | P016 | PRESERVED |
| `IOS-INSTALLER-LIFECYCLE` | P016 | PRESERVED |
| `IOS-INSTALLER-VALIDATION` | P017 | PRESERVED |
| `IOS-PORT-DOCUMENTATION` | P017 | PRESERVED |
| `IOS-RELEASE-ISOLATION` | P016 | RECONCILED WITH TEST EVIDENCE |
| `IOS-UPSTREAM-SYNC-CI` | P017 | PRESERVED |

The runtime script and native workflow are unchanged: ios/arm64, CGO=1, iPhoneOS clang wrapper and minimum iOS 12.0. The Go 1.27.1 positive fixture and five fail-closed cases pass, including repeat application. AppleARMPMUCharger, Darwin exclusion, sysctl/plist metadata and A7 identification remain intact. Installer version 1.0.0, state format 1 and engine pin `d4b4e381a76c75ac79cb7c99558853b62c7c2163879a72c0333acc5c40092d0c` are unchanged.

| Full upstream SHA | Executed focused tests | Outcome |
| --- | --- | --- |
| `36c14e903d73b226a2df5bbf7ab4fbf1943c2906` | `TestLegacyHubGetsSinglePayload`; `TestModernHubRequestResponse`; `TestModernHubMalformedFallbackSinglePayload` | PASS |
| `7305eda972cb423bd51ac18f4966312b346360d0` | `TestUpdatePingsAgentAfterFailedFetch`; `TestSlowAgentKeepsConnectionAcrossPolls`; `TestFetchDataTimeoutKeepsWebSocketOpen` | PASS |
| `d91f06f3c533b231346dc1d0ca5bf0bca265e462` | `TestSlowAgentKeepsConnectionAcrossPolls` | PASS |
| `f8d3fe1817b2ac977c051243d6ffc9077a000ba3` | `TestZeroValueJSONMatchesV1`; `TestSerializationPresenceContracts`; `TestSerializationCBORKeysAndOptionalZero`; `TestPublicAndAlertJSONZeroContracts`; `TestFetchDataViaWebSocketDoesNotRetainOmittedFields` | PASS |

The legacy fix exits immediately after the single version-selected payload; malformed modern requests and repeated fallback sessions also send once. Slow collection survives failed polls because the 100s read deadline outlasts 60s polling plus the 30s data timeout. Atomic last-message tracking limits failed-poll keepalive to three minutes of Agent silence; the stale-peer test proves eventual disconnection instead of an infinite keepalive loop. Late responses are discarded and subsequent polls recover.

Nil non-omitempty inventories intentionally retain the existing JSON defaults: v1 null versus v2 []; explicit empty inventories are [] in both. JSON v1/v2 tests cover absent/zero/false, empty strings/maps/slices, explicit empty inventories, optional zero pointers, nested structures, required zero values and scalar omission. CBOR numeric keys and battery/zero-pointer presence remain compatible; old-Agent monitor guards and fresh decode tests pass. `Details` contains Go durations and travels over CBOR; direct default JSON v2 marshaling of non-nil `Details` has no duration format and is not the stored-stats path. Tests follow the actual contracts without altering wire tags.

## Functional, security and database review

Per-monitor HTTPS TLS verification remains enabled by default. Explicit opt-out selects a separate cloned transport; a secure–insecure–secure sequence on the same manager proves isolation. The bool migration, persistence/config query, CBOR key 6, sync, API copy and frontend switch/badge remain present. Non-HTTP monitor records clear the option; no global TLS bypass was introduced.

HUB_HEADERS keeps validated name/value parsing and errors without secret values. Canonicalized X-Token/X-Beszel override attempts are ignored; configured authentication/version remain authoritative. Malformed input fails startup. Signature/fingerprint/token auth, custom CA and hostname verification, reconnect, SSH fallback/deadlines/cancellation tests pass. Hub monitor ownership, alert access/delivery, collection permissions and private systemd cache policy remain upstream and pass regression tests.

The remaining collector fixes are retained: CPU rollback; opt-in Linux cgroups; interface/container counter reset handling; apt-wrapper detection/xbps; SMART/NVMe wide counters, empty/error responses and namespace rescans; Intel JSON streams/buffer reuse; NVML power fallback; ZFS status/root-dataset capacity/refquota; and systemd availability/user journals. Unsupported local iOS capabilities stay absent/no-op. Shared Hub remote Linux/Windows/macOS telemetry models remain intact; native package and RSSI adapters remain P011/P012.

Dashboard changes retain empty/isolated charts, latency tolerance, cancellation of delayed realtime subscriptions, safe textContent label measurement, last update counts while down and stable dialog layout. Frontend tests/build pass; full browser/device/remote-OS acceptance remains later work.

P006 adds `1790924727_zfs_pool_status.go` then `1790964708_network_monitor_skip_tls_verify.go`, before the existing lexically ordered `initial-settings.go`. Upstream IDs/bodies/order and existing ownership/retention/history rules are unchanged. Fresh disposable databases and targeted migration down/up/idempotence tests pass. Down migrations remove new-field values. The negative test demonstrates that generic down/up can replay one-way initial-settings and collide with an existing superuser. **A full 0.19 backup/upgrade/restore/interruption demonstration remains BLOCKED/PENDING P013/P017 at Gates D/E; no production upgrade is authorized.** No `/var/lib/beszel-hub` or physical iPad database was opened.

## Executed validation and CI

Toolchains: Go **1.27.1**, Bun **1.4.0**. The baseline canonical Go suite passed. Clean Bun tests initially lacked generated Lingui catalogs; genuine build/catalog preparation resolved that prerequisite. Post-merge runs used frozen install, `bun run sync`, tests and the genuine production build. No empty embed directory was used. Build-generated locale additions were restored to the frozen tracked sources after build.

| Command | Result |
| --- | --- |
| `python3 scripts/audit/validate-upstream-v021-matrix.py` | PASS (exit 0) |
| `python3 scripts/audit/validate-upstream-v021-readiness.py` | PASS (exit 0) |
| `python3 -m unittest discover -s tests/audit -p 'test_*.py' -v` | PASS (exit 0) |
| `sh tests/upstream-sync-test.sh` | PASS (exit 0) |
| `sh tests/ios-patch-sentinels.sh` | PASS (exit 0) |
| `sh tests/bootstrap-test.sh` | PASS (exit 0) |
| `sh tests/install-sh-test.sh` | PASS (exit 0) |
| `git diff --check` | PASS (exit 0) |
| `cd internal/site && sha256sum bun.lock && bun install --frozen-lockfile && sha256sum bun.lock` | PASS (exit 0) |
| `cd internal/site && bun run sync` | PASS (exit 0) |
| `cd internal/site && bun test` | PASS (exit 0) |
| `cd internal/site && bun run build` | PASS (exit 0) |
| `go test -tags "testing no_ui" ./...` | PASS (exit 0) |
| `env -u NETWORK go test -race -tags 'testing no_ui glibc' ./agent -run 'Test(NvmlSampleToFloat\|PowerFromSamples.*)' -count=1 -v` | PASS (exit 0) |

Audit unit tests: **20**, including **6 A7 fixture cases**. Sync/sentinels/bootstrap/installer: **10/44/54/814** checks. Frontend: **44 tests, 149 expectations**; genuine dist: **95 files**, 88 JavaScript assets. Linux Agent and Hub builds pass. iOS/macOS/Linux Go file selection passes. Optional Windows file inspection was blocked by the upstream generated LHM embed directory being absent; no Windows build is claimed.

Focused suite: **936 passing test/subtest events**, zero failures. Race suite across Agent, Hub systems/WebSocket/SSH, alerts, migrations and entities: **1775 passing test/subtest events**, zero failures. Four local ICMP socket subtests are **SKIPPED**, not passes, due to cloud socket permissions. Packages with no test files are also not counted as tests passed. P010/P017/P018 own native ICMP acceptance. An initial race compile overlapped Vite dist regeneration and failed on disappearing embed assets. The stable-assets run completed 11 packages, but alerts reached the default 10m timeout while progressing through password-heavy fixtures. The complete unchanged alerts package passed a rerun with `-timeout 30m`. Final race totals combine only completed package results; timed-out partial events are excluded.

- [iOS Build — 38486554442](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38486554442): **success**, SHA `fd6eb7f69bf6240f36d20c0a3ad7f397d45035fd`.
- [iOS installer lifecycle — 38486558092](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38486558092): **success**, SHA `fd6eb7f69bf6240f36d20c0a3ad7f397d45035fd`.
- [Helm charts — 38486605239](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38486605239): **success**, SHA `fd6eb7f69bf6240f36d20c0a3ad7f397d45035fd`.
- [iOS Build — 38488654850](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38488654850): **success**, SHA `15bc4226eac9f1dd8a9c5d0dd6de5a4ccd2966f5`.
- [iOS installer lifecycle — 38488657303](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38488657303): **success**, SHA `15bc4226eac9f1dd8a9c5d0dd6de5a4ccd2966f5`.
- [Helm charts — 38488659904](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38488659904): **success**, SHA `15bc4226eac9f1dd8a9c5d0dd6de5a4ccd2966f5`.

Native logs show Go 1.27.1, Xcode 15.4/iPhoneOS SDK 17.5, actual A7 source patch, both Agent/Hub CGO builds, Mach-O ARM64/platform 2/minos 12.0, asset filenames and verified SHA256SUMS. Agent SHA256 `30c34192d45cf82ceb1894d783aa98878256ecd1fc4673845452cb1d3bc6dd08`; Hub SHA256 `4436b39de019e027599fc4dbfc206f2d51f5cc532b915a4bfe6c761921931704`. `release-ios` and Helm publishing are intentionally skipped. Normal Actions build artifacts were uploaded; no release/registry publication occurred. Optional local artifact re-download returned cloud blob HTTP 403; no local Mach-O artifact inspection is claimed beyond the successful hosted checks.

After this evidence commit, final-head iOS Build/installer and applicable PR checks are rerun before merge. Their exact final feature SHA/run URLs and post-merge refs are recorded in PR #8 and the completion report; this file records the tested source candidate and initial exact runs to avoid a self-referential commit hash. Any changed build input requires affected CI reruns.

The frozen dependency audit returned **29 advisory entries across 9 packages** (1 critical, 19 high, 8 moderate, 1 low). Details/URLs are in JSON. The critical tar advisory is in `tar@7.5.13 <- @tailwindcss/oxide@4.1.12 <- @tailwindcss/vite@4.1.12`, a dev build path. P015/P017 own full exposure triage, Go inventory and remediation at Gates D/E; no dependency was changed to suppress findings. Build warnings about mixed imports/large chunks remain informational.

## Deferred obligations and Gate B decision

- P007/P008: platform/A7 and native metrics/battery acceptance, with P018 device execution (C/F).
- P009/P010: exact old/new transport pairs, native monitors/ICMP and unsupported-capability fixtures with P017/P018 (C/F).
- P011/P012: read-only jailbreak package updates and native Wi-Fi RSSI adaptation (C).
- P013/P017: complete consistent 0.19 backup, seeded upgrade, full restore, interruption recovery and account/config/history invariants (D/E; production upgrade blocked).
- P014/P015/P017: browser/API/security/remote-OS matrix, dependency triage and reproducible candidate evidence (D/E).
- P016/P017: installer data-aware rollback and service/CLI acceptance (E).
- P018: physical A7/no-SIGILL, service/reboot/reactivation and release authorization (F).

Gate B **PASS** means the frozen 178-commit source history is integrated with preserved iOS contracts and passing integration checks. Gates C–F remain pending. `iOS` and `main` are unchanged by P006. No release, tag push, image/chart publication, production-data mutation or deployment occurred. Next is P007; it was not begun.

## Exact P006 inventory

Every entry below is reachable; functional owners follow P003, separate from P006's history ownership.

| Full upstream SHA | Priority | Canonical functional owner |
| --- | --- | --- |
| `4a40d1772c2c0fcb1aec1fefecbbc56bb6622cbe` | P2 | P016 |
| `7748651f5cb3ee0ccd3d3e6141f315ee5d20454e` | P2 | P016 |
| `dcae7682cddf5e52ba3bd7b3aebfa0f472d8c890` | P2 | P017 |
| `36c14e903d73b226a2df5bbf7ab4fbf1943c2906` | P0 | P009 |
| `74c952b87907a36d84d404ad17a0695be4ac55df` | P1 | P011 |
| `d89f844a2a0abc7474555f0635344245456f82d6` | P1 | P010 |
| `ca40b5b38ffac6336872cdb0d5f4f053916f394a` | P2 | P014 |
| `965518e63c92cbf72e498ab6d10f14f0a78f4196` | P2 | P014 |
| `ef840e0e6d8f066419a351c25312acb2d5323af7` | P1 | P008 |
| `4fe1a0a7a42f655564e0b1577a2bc0dd74af8f58` | P2 | P008 |
| `bc2278e7e9820825ce990fb2fe7fca4b9fe9a1c3` | P1 | P011 |
| `7305eda972cb423bd51ac18f4966312b346360d0` | P0 | P009 |
| `d91f06f3c533b231346dc1d0ca5bf0bca265e462` | P0 | P009 |
| `195afc10b8924ef4d6afc5d409eb8a8e1c43b96c` | P2 | P015 |
| `e30e980854408d46b2e8837a14a0c0d771a18545` | P1 | P015 |
| `5529d2e15e142dcec0ee5a239ffadb33e3295b01` | P1 | P010 |
| `5714643ff9001e70e0fbf8ecb4f55369dcf7ae61` | P2 | P014 |
| `b11e3617990bee170122c4af1913ed79ddb4a632` | P1 | P008 |
| `017ebb091aa64596f9545c89e67f6c880f8bb121` | P1 | P010 |
| `d8b41919751da63f63745b4a9d2ce894dd61342d` | P1 | P008 |
| `13bdd05934a7363016aa9a92c0d5bef13c0e81a4` | P1 | P010 |
| `383e9c707249e88075044d2170a0f1e68411ee76` | P1 | P010 |
| `f8d3fe1817b2ac977c051243d6ffc9077a000ba3` | P0 | P009 |
| `7f63e65f22bff5d4bf5962e3555bc4107c31de6e` | P2 | P010 |
| `7f2eb90f24df180a7c0dd6c1b888aaba65b3dbb6` | P2 | P010 |
| `d324ccc0977d0347e4f94529490fefbc72e54900` | P1 | P009 |
| `4da7d867f237542aadac148eebbcff60f82d9ef9` | P1 | P010 |
| `596f96442e5b60706018e47d63c56bd5abad2a37` | P1 | P016 |
| `da51dfd5986528d7341ba488b96046603298ae0b` | P1 | P014 |
| `3d33d5eabb2bed36b5ff252057a96647dbd8c8c7` | P1 | P010 |
| `665bcb3e53d80a4d4c01e43e0b0bea0ecc85383c` | P1 | P010 |
| `a1cbdc980e05f0a03616d89e0b3d6986e0e59aa8` | P2 | P017 |
| `a1d86d120fa046eabdfc0fc1a674e34aebdc81f1` | P2 | P014 |
| `02f24ed0af859dd9a3e21ba009ed3714733a50f8` | P1 | P010 |
| `9cf75cca7af6a198c418d99a2bd6d78efb2e5d9b` | P2 | P014 |
| `32d469285382521bae4ad3f368a7045f32ff2f5e` | P1 | P010 |
| `bb23f99b3eeaced9608e11e280e7c986dba650be` | P1 | P010 |
