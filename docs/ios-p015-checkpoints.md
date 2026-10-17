# P015 checkpoints

## Repository state

- Repository: nghianguyen150612/Beszel-iOS.
- Feature branch: ios/frontend-assets-localization-security.
- Base and starting integration SHA: ios/upstream-v021-integration at 8cd928e3cbb3e4f22556ad82146e109685ed7261.
- P014 PR #17 is merged normally at that SHA. Abandoned PR #16 is not part of integration.
- P014 evidence records 178/178 frozen upstream commits and 13/13 logical iOS patches. Production iOS, frozen main, and P012 native RSSI deferral remain unchanged.

## Checkpoint A — frontend build and asset manifest

The first pushed checkpoint is a70a1303f0ee80acf8f0f94b71a3944c91e59e09. It established clean Bun install/build and manifest checks. The final manifest was regenerated after focused frontend fixes and semver-compatible security updates.

The implementation checkpoint is d855182a513084fcef67a17da296bc49c38fe333. It is pushed and contains the frontend, Hub, lockfile, and browser/security changes. The remaining evidence commit contains documentation and risk-record updates.

- Go 1.27.1; Bun 1.4.0.
- Two clean bun install --frozen-lockfile and bun run build cycles passed.
- Final manifest: 95 files, 3,357,427 bytes, SHA-256 4bd19c22f587053da5bbf0275ad602bad0ba00bd243898260a65756c59b421c6.
- bun.lock SHA-256 e2d090fa8b5bcffd8e2de003262cfe6c506c297a676279fbb08d1c2db3bc6dbf; frozen installs did not rewrite it.
- package-lock.json SHA-256 6fbd1027efc8368462a78a0ff87dae23b758eee759ba56172eb862f70df1b3d5 and remained unchanged.
- Lingui, TypeScript, Vite, embedded asset validation, and 55 Bun tests/267 assertions passed.
- Existing Vite dynamic/static import and large WASM/vendor warnings remain.

## Checkpoints B and C — actual Hub, routes, and localization

- Actual Hub served 94 non-index asset hashes matching the final manifest; index.html was dynamically modified for base path and version.
- Nine supported SPA routes returned 200 at root and /beszel. MIME/cache checks passed; unknown and filesystem-like paths returned 404.
- Fresh unauthenticated collection requests returned empty lists. Seventeen ranges returned 400; maximum is sixteen.
- Browser harness passed login/logout, account isolation, expired session, safe failed-request fallback, routes, deep links, history/refresh, monitor details, and synthetic system rendering.
- All 32 locale modules compiled. Browser checks passed vi-VN detection, invalid locale fallback, Vietnamese Unicode and persistence, Arabic/Persian/Hebrew RTL layout and seven charts, then English restore.
- Existing API/metric tests preserve Linux/Windows/macOS/iOS snapshots, valid zero, null/absent optional data, and chart gaps. Vietnamese is 529/537 translated.

## Checkpoint D — advisory review and regressions

- Bun audit found 29 records in the original graph. Eight semver-compatible transitive updates resolved 28; one build-only GHSA-vfj7-8cjw-p6xm remains without a compatible fix.
- govulncheck 1.9.0 reviewed Go 1.27.1 using the 2026-10-08 database snapshot. The critical HTTP/2 server finding is mitigated and tested; Range parsing is capped. High multipart MIME finding GO-2026-6608 remains documented for the pinned toolchain.
- Python audit suite: 67 passed. P013 backup fixture unit suite: 7 passed.
- Shell suites: upstream sync 10, patch sentinels 45, bootstrap 54, installer lifecycle 814 passed.
- Canonical Go tests across ./... passed with Go 1.27.1. Targeted Hub race tests passed. Linux Agent and Hub builds succeeded.
- Source matrix/readiness passed: 178 upstream, 35 historical iOS, 13 logical patches, zero unresolved audit rows; readiness reports 24 acceptance cases and 33 compatibility scenarios.
- The first P015 workflow run was a P013 fixture pass on the initial checkpoint. Final-head evidence is recorded below.

## Hosted final-head CI and Gate D

Status: **PASS** at tested source SHA `0f649d4cf861439d136845bd101cda9a6569ef69`.

- iOS Build [39011382151](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39011382151) passed all jobs on that exact SHA. The release job was skipped because this was a branch run, not an iOS release tag.
- Installer lifecycle [39011411689](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39011411689) passed on the same SHA.
- P013 Hub migration fixture [39011373934](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39011373934) passed on the same SHA.
- Security review is complete: govulncheck and Bun/npm audit findings have evidence-backed dispositions; focused HTTP/2 ALPN, Range, authorization, and asset-serving regressions pass. The high multipart MIME finding and build-only `braces` advisory remain documented; no unmitigated critical release-blocking finding remains.

### Artifact verification

The artifacts were downloaded from GitHub and their archive SHA-256 values matched the GitHub artifact API digests. The payload `SHA256SUMS` also verified after extraction.

| Artifact | ID | Archive bytes | Archive SHA-256 | Verified contents |
| --- | ---: | ---: | --- | --- |
| `beszel-ios-arm64` | 12093266412 | 16,284,834 | `42720d752e99953fc2f1ffd68689eb97c4cec85b58c002ca6d20f32cb0e3e38e` | Agent and Hub payload hashes below |
| `ios-native-validation` | 12092379338 | 7,838 | `6e8bca17f33a8956505791faeec7c9a2b9c622cb599e77f5781b7dc22711d476` | Native metadata, Go source selection, runtime and Wi-Fi linkage evidence |
| `ios-wifi-diagnostic-disabled` | 12093505693 | 6,050 | `9e70fa160fa9ac2855114bb2073ea96ea657e3d69f3fe9adc0e692942709c657` | Separate diagnostic artifact; validation says it is not in the Agent/Hub payload |

| Payload | Bytes | SHA-256 | Platform metadata |
| --- | ---: | --- | --- |
| `beszel-agent-ios-arm64` | 9,542,984 | `d50998333beeaf2f97f98a6e5ee46fc144077d28028b8637b89fb332ef5602f2` | Mach-O arm64, iOS platform 2, minimum 12.0 |
| `beszel-hub-ios-arm64` | 31,977,208 | `f0464d716a5fa397214809a115cb46f4bd499c4bb0046616a86bda20b405157f` | Mach-O arm64, iOS platform 2, minimum 12.0; 95 embedded frontend assets |

Both binaries report Go 1.27.1, `GOOS=ios`, `GOARCH=arm64`, `CGO_ENABLED=1`, `vcs.revision=0f649d4cf861439d136845bd101cda9a6569ef69`, and `vcs.modified=false`. Agent and Hub runtime instruction checks pass with no counter-register access; the isolated Go runtime patch uses a zero-guarded YIELD loop and no `CNTFRQ_EL0`/`CNTVCT_EL0` reads. The separate Wi-Fi diagnostic reports `not_in_agent_hub_payload: true`; native linkage found no private MobileWiFi/CoreWLAN/Apple80211 symbols.

Gate D passes: P013 and P014 acceptance remains valid; clean frontend builds, embedded assets, routes, authentication boundaries, localization/RTL, missing-metric handling, security review, regression suites, final-head hosted CI, and artifact provenance are verified. The high dependency residuals remain explicit and are not described as fixed. P012 native RSSI remains `DEFERRED — REFERENCE DEVICE UNAVAILABLE`; P018 physical acceptance is still pending. This decision does not authorize release or deployment.

The artifact-producing source tree is the tested `0f649d4` tree. This final evidence update changes documentation only; no product source changed after the hosted run. The merged product source must remain identical to the tested source tree.
