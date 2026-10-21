# P021 durable checkpoint ledger

## Base and scope

P021 starts from P020 normal merge `42556e62c23f3595a276d2eb4a55125a96f5d31f` on `ios/upstream-v021-integration`. P019 merged normally as `c8bd3dba8656c8195ffacf56ff743dd58b01da25`; P020's merged feature head was `c716d3c9ce8de170cade25b3921bccef8c5b97a7`. The pinned upstream is `b5357343550c8e2493d0e65472a06c1007396384`.

The old unpushed P019 commits were unrecoverable and P019 was independently rebuilt; the original identities are not claimed. P021 reviews P001–P020 without another upstream merge or unrelated features. No product implementation change was needed. P021 added `docs/ios-p021-*` to P013 and installer workflow pull-request filters, so every final evidence checkpoint runs the genuine migration fixture and installer lifecycle suite.

Branch: `ios/p021-integration-closeout`
PR: https://github.com/nghianguyen150612/Beszel-iOS/pull/25
Base: `ios/upstream-v021-integration` at `42556e62c23f3595a276d2eb4a55125a96f5d31f`.

## Upstream and historical accounting

The pinned upstream is an ancestor. The range after frozen baseline `bb23f99b3eeaced9608e11e280e7c986dba650be` contains 41 commits. Read-only P002/P003 validators passed on P021 candidate `b20ed2b636d95bca26c87161de06d8e1e7f03862`: 178/178 upstream-only commits accounted, 35/35 historical iOS commits mapped, 13 logical iOS patchsets, zero unresolved audit rows, 24 acceptance cases, and 33 compatibility scenarios. Native CI on `b20ed2b` passed the upstream-sync and iOS-patch sentinels.

## Test correction and exact source history

Initial P021 qualification on `037ffa8457cbd33b7444a0bc9035232c0b7502c6` passed P013 `39614985429`, P017 `39614985449`, native iOS/Procursus `39614985052`, and installer `39614984961`.

A later P017 run on `9ef6d859923b4fe832a4bcf3f54f2edf6e0f38be` failed in `TestClientTimeoutCancellationAndRepeatedCleanup`. The test used a 100 ms helper timeout, assumed each child had started before checking a PID file, and could also observe a stale PID marker from the previous attempt. The application code did not fail. The test now waits for a distinct appended start marker for each of 12 acquisitions, verifies the actual child is reaped, allows the helper to initialize, and bounds completion. Focused Go 1.27.2 checks passed locally: non-race `-count=2` and race `-count=1`. The failed P017 run was `39618985621`; the old native run `39618985568` was cancelled when the branch advanced and is not counted as a result.

## Final implementation/test source `b20ed2b`

All completed checks below ran on exact candidate source `b20ed2b636d95bca26c87161de06d8e1e7f03862`, not a PR merge pseudo-commit:

| Workflow | Run | Result | Evidence |
|---|---:|---|---|
| P013 Hub migration fixture | `39620427023` | PASS | Genuine v0.19 fixture and restore checks; artifact `12354696647`, SHA-256 `2c86217d7436954a0629769f0629708a2ea9d90ae33d4b9d3adc3ce32ef74243` |
| P017 Release Candidate Regression | `39620427058` | PASS | Full Go regressions, security scans, frontend build, and provenance; artifacts listed in validation JSON |
| Native iOS Build | `39620426859` | PASS | Procursus and native build jobs succeeded; release job skipped |
| Installer lifecycle | `39620426909` | PASS | Bootstrap checksum, lifecycle, rollback, workflow isolation, and patch sentinels |

P017 records source SHA `b20ed2b636d95bca26c87161de06d8e1e7f03862`, 97 built frontend assets, and `device_tested=false`, `release_published=false`, `tag_created=false`. Native CI used Go 1.27.2, `GOOS=ios`, `GOARCH=arm64`, `GOARM64=v8.0`, `CGO_ENABLED=1`, clean VCS metadata, and the iOS 12.0 minimum. It produced ARM64 device Mach-O binaries with real embedded frontend assets, matching SHA256SUMS, and no private MobileWiFi linkage. Exact binary hashes, runtime and x509 patch hashes, and artifact digests are in `ios-p021-validation.json`.

The Go 1.27.2 scan omits `GO-2026-6608`; three raw `GO-2026-6809` traces remain and the official CL 851047 x509 backport/tests passed; one module-level `GO-2026-5932` finding remains open and is not claimed fixed. Bun/npm reported zero critical findings; `GHSA-vfj7-8cjw-p6xm` remains OPEN. See `ios-p021-security-disposition.md`.

## Final evidence checkpoint

The current evidence update also changes no product behavior. It records the `b20ed2b` test correction and exact run results. Because `docs/ios-p021-*` is routed through P013, P017, native iOS/Procursus, and installer workflows, all four must pass again on the final PR head before merge. Their exact head and run IDs will be recorded in the PR check suite and final closeout comment; the committed evidence remains pinned to `b20ed2b` to avoid a self-referential run-ID commit cycle.

GitHub's branch ref may briefly advance before the PR's internal head ref. Do not merge during that interval: require `refs/pull/25/head` and PR metadata to match the live feature branch, then verify the complete final-head check suite.

## Review and merge record

No independent review is claimed. If repository policy requires an owner exception, PR #25 records the repository owner's explicit PROMPT021 authorization to merge after all checks pass. No reviewer is impersonated and no check is bypassed. Merge only after the final PR-head checks succeed, then use a normal merge commit and verify the PR and integration ref SHA.

## Boundaries

Cloud results do not constitute device validation. P018 physical Gate F remains PENDING; native Wi-Fi RSSI remains DEFERRED. No physical device is accessed. Production `iOS`, fork `main`, release tags, releases, and deployment remain unchanged and out of scope. Earlier Work incidents involving `/dev/shm/beszel_health` and `bwrap` are recorded in the validation JSON and were not repeated in this continuation.
