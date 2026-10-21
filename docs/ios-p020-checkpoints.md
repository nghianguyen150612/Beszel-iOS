# P020 durable checkpoint ledger

## Base and scope

P020 starts from the verified P019 integration merge
`c8bd3dba8656c8195ffacf56ff743dd58b01da25` on
`ios/upstream-v021-integration`. P019 PR #23 was merged with a normal merge
commit. The P019 feature source was `81e6a5e779efd61ac464b9835eb1b1658d4263d8`;
the merge has parents `f9d6500c7a68572dc8e464dd305d79bf6c1e79a9` and that feature
head. Pinned upstream `b5357343550c8e2493d0e65472a06c1007396384` is an ancestor.

Post-merge structural validation reported all 41 pinned upstream commits,
178/178 upstream audit rows, 35/35 frozen iOS commits, 13 logical iOS
patchsets, zero unresolved audit rows, 45 passing iOS patch sentinels, and 10
passing upstream-sync sentinels. P019 details are in
`docs/ios-p019-checkpoints.md` and `docs/ios-p019-security-review.md`.

P020 branch: `ios/p020-post-upstream-qualification`.
Target: `ios/upstream-v021-integration`.
PR #24: https://github.com/nghianguyen150612/Beszel-iOS/pull/24.

## Durable checkpoints

| Source SHA | Remote verified | Purpose |
|---|---|---|
| `6a0d5904fe7eb3cd6ab8a9ea35bad10bbd331f65` | Yes | Initial review checkpoint and Draft PR. |
| `2ad963066aa918e86158e31c0b60d65bacb2218e` | Yes | Reject universal-token GET URL parameters; add deterministic tests. |
| `5819a82b8d628b66411335643c767e1e147de93d` | Yes | Completed review, matrix, security disposition, and initial exact-head qualification. |
| `e7d3d36041d4bb49eeeb15ee4dcae2fe4d7d1391` | Yes | Add deterministic CPU breakdown field/order coverage to close a reviewed test gap. |

## Confirmed regression and correction

The post-upstream review found that `GET /api/beszel/universal-token` still
accepted `token`, `enable`, and `permanent` query parameters. The frontend was
already using POST, but the legacy GET path allowed a token in the request URL
and allowed GET state changes. P020 makes GET read-only and returns HTTP 400
when any of those query keys is present. Token issuance, mutation and token
comparison continue through the existing POST JSON body. `Cache-Control:
no-store` remains set.

Tests independently reject each query key, assert no token echo and no
permanent-record side effect, and verify that a later GET remains inactive.
Existing POST endpoint tests cover activation, expiry/persistence, disable,
role authorization, ownership and Agent authentication.

Local Go 1.27.2 checks on code source `2ad963066aa918e86158e31c0b60d65bacb2218e`
passed:

- `go test -tags 'testing no_ui' ./internal/hub -run '^(TestApiRoutesAuthentication|TestUniversalTokenEndpoint)$' -count=1 -timeout=5m`.
- `go test -race -tags 'testing no_ui' ./internal/hub -run '^TestUniversalTokenEndpoint$' -count=3 -timeout=5m`.
- `gofmt` and `git diff --check`.

## Initial hosted qualification on exact source `5819a82b8d628b66411335643c767e1e147de93d`

| Workflow | Run | Result | Evidence |
|---|---:|---|---|
| P017 Release Candidate Regression | [39605920035](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39605920035) | PASS | Full canonical Go tests and builds, Go and Bun/npm audits, installer/bootstrap/isolation regression, ancestry and patch accounting, genuine frontend build. Manifest: 97 assets, digest `3b80dbea2957210afc99b4f4e2322d682015f53eb6034d163f55d3a9fc2a92e1`; source `5819a82`. |
| P013 Hub migration fixture | [39605920033](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39605920033) | PASS | Genuine v0.19 Hub-created synthetic database, candidate migration, restart, full backup/restore and interruption recovery. Evidence artifact `12348148448`, SHA-256 `066897290e41baa418d1e26e283e86a3977b0752e3ea78741801be1011ef6ae3`. |
| iOS Build | [39605920505](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39605920505) | PASS | Native transport/race and monitor suites, Go A7 patch, ARM64 Mach-O builds, iOS 12 metadata, provenance, checksums, no private MobileWiFi linkage, and 97 embedded assets. The release job was skipped. |
| Procursus network-disabled isolation | [39605920505](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39605920505) | PASS | Hosted container uses `--network none` and `--read-only`; Procursus parser/read-only tests passed. |
| iOS installer lifecycle (manual dispatch at P020 source) | [39606486222](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39606486222) | PASS | Bootstrap checksum, bootstrap tests, Mach-O/workflow isolation fixtures, installer lifecycle, 45 iOS patch sentinels and 10 upstream-sync cases. Source `5819a82`. |

The native run checked out exact source `5819a82`. The patched Go runtime
`runtime/asm_arm64.s` changed from SHA-256
`47f9313d4f956b4159dcf8b292bf340a12adb6b58ee9a94360fe948045e2c5db` to
`c861b98d76ac487269c7f98c987b94ea9978c5f01740a4596eecbfdb9c8bb91f`; the
replacement remains a zero-guarded `YIELD` loop without `CNTFRQ_EL0` or
`CNTVCT_EL0` reads. The official Go CL 851047 x509 backport also passed
`go test crypto/x509 -count=1`.

Agent SHA-256:
`46295c407078811e9fd44e5bfdecb1957cee09380de88bd1e94fddd893492ea6`.
Hub SHA-256:
`c142ed153d35970558a4e27e2bafb5e73964600f39a90aa8e240fa4672cb0a02`.
Both report Go 1.27.2, `GOOS=ios`, `GOARCH=arm64`, `GOARM64=v8.0`,
`CGO_ENABLED=1`, VCS revision `5819a82b8d628b66411335643c767e1e147de93d`,
and `vcs.modified=false`. Both are ARM64 Mach-O iOS device binaries with
minimum iOS 12.0. The Hub embeds 97 frontend assets. Native artifacts:

- Payload artifact `12349287095`, SHA-256
  `f29c639a630d25c2956fc96819e1adf1bde4445180d473f481ca3e39e05cf063`.
- Native validation artifact `12348826761`, SHA-256
  `8a32720cb42171a3c16e9b44dc93218357f4607c7d9a4f1fde0ca0228fbe9305`.
- Separate disabled Wi-Fi diagnostic artifact `12348662683`, SHA-256
  `d33be5a3dbe1509f6498bd555798914dc2b5ba48b79c6a476607868409fb2830`.

GitHub artifact metadata matched all native, migration, Go and JavaScript
artifact digests. The P017 candidate manifest explicitly records
`device_tested=false`, `release_published=false`, and `tag_created=false`.
No production secrets or production data were included in review artifacts.

## Security disposition

The exact-source Go scan excludes `GO-2026-6608`. It retains three raw
`GO-2026-6809` traces while recording the official Go CL 851047 backport and
passing x509 test. The module-level `GO-2026-5932` finding remains recorded and
is not claimed fixed. Bun and npm report zero critical findings and keep
`GHSA-vfj7-8cjw-p6xm` OPEN. Details and exact artifact digests are in
`docs/ios-p020-security-review.md` and `docs/ios-p020-validation.json`.

## Merge state and remaining boundary

The initial full qualification passed on `5819a82`. Review then found no direct assertion for the exported CPU breakdown vector order. P020 added `TestCpuBreakdownContract` for User, System, Iowait, Steal, Idle, IRQ, SoftIRQ, and Nice. The code checkpoint `e7d3d36041d4bb49eeeb15ee4dcae2fe4d7d1391` passed the complete P017, P013, native iOS, Procursus, and installer workflows listed below. Those exact-source results supersede the earlier checkpoint for code qualification. After this evidence update is committed, require every check on the final documentation-only PR head to pass before merge.

| Workflow | Exact-source run | Result | Evidence |
|---|---:|---|---|
| P017 Release Candidate Regression | [39609195215](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39609195215) | PASS | Full Go suite/builds, Go and Bun/npm audits, frontend tests/Lingui/TypeScript/Vite, installer/bootstrap/isolation regressions and provenance. 97-asset manifest artifact `12348869369`, SHA-256 `8f3e0678dc84809a076e416921b3c6aee54985f153bd5c3a707cf36ebe7fb66a`. |
| P013 Hub migration fixture | [39609195286](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39609195286) | PASS | Genuine v0.19 Hub-created synthetic database upgraded, restarted, backed up/restored, and recovered after interruption. Artifact `12349727839`, SHA-256 `75de188c4487c01e86508d512112b6ff526e6e09d5108e33c47124d3e3c236a1`. |
| iOS Build | [39609195413](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39609195413) | PASS | ARM64 iOS device Mach-O, Go 1.27.2 A7 runtime patch, x509 backport, native race/monitor suites, iOS 12 minimum, provenance, checksums, 97 embedded assets and no private MobileWiFi linkage. Release job skipped. |
| Procursus network-disabled isolation | [39609195413](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39609195413) | PASS | Hosted container ran with `--network none --read-only`; parser and read-only query tests passed. |
| iOS installer lifecycle | [39609283025](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39609283025) | PASS | Exact `e7d3d36` source; bootstrap checksum/tests, Mach-O and workflow isolation fixtures, installer lifecycle, 45 iOS patch sentinels and 10 upstream-sync cases. |

The native Go runtime source `runtime/asm_arm64.s` changed from SHA-256
`47f9313d4f956b4159dcf8b292bf340a12adb6b58ee9a94360fe948045e2c5db` to
`c861b98d76ac487269c7f98c987b94ea9978c5f01740a4596eecbfdb9c8bb91f`. The
replacement remains a zero-guarded `YIELD` loop without `CNTFRQ_EL0` or
`CNTVCT_EL0` reads. The official Go CL 851047 x509 backport passed its standard
library test.

Agent SHA-256:
`7956034cf1d0b0d1755cf082ccd021d3c0c40118a82fda723c4ee22c232ba1a5`.
Hub SHA-256:
`727356fdc73a4d3985924dd394ba6f84f5cf7b9010e8941c4b7aabb689196bb5`.
Both report Go 1.27.2, `GOOS=ios`, `GOARCH=arm64`, `GOARM64=v8.0`,
`CGO_ENABLED=1`, VCS revision `e7d3d36041d4bb49eeeb15ee4dcae2fe4d7d1391`,
`vcs.modified=false`, and the iOS 12.0 minimum. The Hub embeds 97 assets.
Artifacts: payload `12349709491` (SHA-256
`5b19ded34bbbfc94e9b3b68478541f93aa83a20e2310aa973fce920681df1492`), native
validation `12349404872` (SHA-256
`2c7f46ab1c405cae969e9fd88539c3a0437bbb37323202fde44353a9477cf896`), and
disabled Wi-Fi diagnostic `12349884248` (SHA-256
`9f4041b702977adca9e75f630634354eefea65ce3a089baf06502bf489582d00`).

P017 exact-source security results: `GO-2026-6608` absent;
`GO-2026-6809` retains three raw traces with the official backport and passing
x509 test; `GO-2026-5932` remains an open module-level finding and is not
claimed fixed. Bun/npm report zero critical findings and retain
`GHSA-vfj7-8cjw-p6xm` as OPEN. The P017 Go, JavaScript and manifest artifact
digests are recorded in `docs/ios-p020-validation.json`. P017 manifest source
is `e7d3d36`; it records 97 assets and `device_tested=false`,
`release_published=false`, and `tag_created=false`.

Artifact archive retrieval from the local session returned HTTP 403 from the
GitHub Actions signed blob URL. GitHub artifact metadata and completed workflow
logs were available and matched the exact source, workflow results, and
artifact digests above; the CI jobs themselves validated the generated files.

The first local vector assertion used exact float equality and failed because one computed value was `7.000000000000001`. The test now uses a `1e-10` tolerance. The corrected race test passed three consecutive runs.

PR #24 has no submitted reviews and no requested reviewers; no independent
review is claimed. The owner's PROMPT021 authorization expressly allows the
repository-level review-policy exception after mandatory technical and
security gates pass. Do not impersonate a reviewer or bypass failed checks.

An earlier Work execution opened and truncated `/dev/shm/beszel_health`, and
its Procursus isolation test invoked `bwrap`. This continuation did not repeat
either action: local health markers use `.task/health`, and the Procursus test
ran in the hosted network-disabled, read-only container. No local `bwrap`,
`/dev/shm` or other out-of-workspace write occurred in this continuation. A
GitHub Actions artifact download attempt was denied by its signed blob URL; no
artifact was extracted locally.

No physical device, production `iOS` branch, fork `main`, release tag, or
published release is in scope. P018 physical-device Gate F remains pending.
