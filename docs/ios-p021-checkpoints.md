# P021 durable checkpoint ledger

## Base and scope

P021 starts from P020 normal merge `42556e62c23f3595a276d2eb4a55125a96f5d31f` on `ios/upstream-v021-integration`. P019 merged normally as `c8bd3dba8656c8195ffacf56ff743dd58b01da25`; P020's merged feature head was `c716d3c9ce8de170cade25b3921bccef8c5b97a7`. The pinned upstream is `b5357343550c8e2493d0e65472a06c1007396384`.

The old unpushed P019 commits were unrecoverable and P019 was independently rebuilt; the original identities are not claimed. P021 reviews the unified P001–P020 source without another upstream merge or unrelated features. The review found no further justified product-code change. It adds `docs/ios-p021-*` to the P013 and installer workflow pull-request path filters so documentation-only P021 checkpoints still run the genuine migration fixture and installer lifecycle suite.

Branch: `ios/p021-integration-closeout`
PR: https://github.com/nghianguyen150612/Beszel-iOS/pull/25
Base: `ios/upstream-v021-integration` at `42556e62c23f3595a276d2eb4a55125a96f5d31f`.

## Upstream and historical accounting

The pinned upstream is an ancestor. The pinned range after frozen baseline `bb23f99b3eeaced9608e11e280e7c986dba650be` contains 41 commits. On the P020 base, the read-only P002 matrix validator reported 178/178 upstream-only commits reviewed, 35/35 historical iOS commits mapped, 13 logical iOS patches, and zero unresolved rows. The P003 readiness validator reported 24 acceptance cases and 33 compatibility scenarios. Both validators were rerun on P021 source `037ffa8457cbd33b7444a0bc9035232c0b7502c6`; they passed with the same counts. Native P021 CI also passed the upstream sync and iOS patch sentinels on that exact source.

## Exact P021 source qualification

All checks below ran against feature-branch source `037ffa8457cbd33b7444a0bc9035232c0b7502c6`, not a PR merge pseudo-commit:

| Workflow | Run | Result | Evidence |
|---|---:|---|---|
| P013 Hub migration fixture | `39614985429` | PASS | Genuine v0.19 fixture, backup/restore failure cases, source matrix; artifact `12352203494`, SHA-256 `79654c72e1f2060368a30687bf9d95df2518e628bbe957e174c392763361229b` |
| P017 Release Candidate Regression | `39614985449` | PASS | Full Go regressions, security scans, frontend build, provenance; artifacts listed in `ios-p021-validation.json` |
| Native iOS Build | `39614985052` | PASS | `procursus-package-updates` and `build-ios` succeeded; `release-ios` skipped |
| Installer lifecycle | `39614984961` | PASS | Rootful/Procursus, bootstrap, transaction lifecycle, rollback, sentinels |

The P017 manifest reports source SHA `037ffa8457cbd33b7444a0bc9035232c0b7502c6`, 97 embedded assets, and `device_tested=false`, `release_published=false`, `tag_created=false`. The native build used Go 1.27.2, `GOOS=ios`, `GOARCH=arm64`, `GOARM64=v8.0`, `CGO_ENABLED=1`, clean VCS provenance, and an iOS 12.0 minimum. It produced ARM64 iOS device Mach-O binaries with genuine embedded frontend assets and no private MobileWiFi linkage. Native binary hashes and artifact digests are in the validation JSON.

The Go 1.27.2 scan does not report `GO-2026-6608`. Three raw `GO-2026-6809` traces remain visible; the official CL 851047 x509 backport and tests passed. One module-level `GO-2026-5932` finding remains open and is not claimed fixed. Bun/npm reported zero critical findings; `GHSA-vfj7-8cjw-p6xm` remains OPEN. See `ios-p021-security-disposition.md`.

These results qualify P021 candidate `037ffa8`. The final checkpoint adds P021 documentation and path-filter-only routing in the installer workflow, in addition to the P013 filter already present at `037ffa8`. No product source changes are introduced. The final PR head must pass P013, P017, native iOS/Procursus, and installer checks; exact head and run IDs are recorded on PR #25. Existing artifacts remain pinned to `037ffa8`.

## Review and merge record

No independent review is claimed. If repository policy requires an owner exception, the PR records the repository owner's explicit PROMPT021 authorization to merge after all required checks pass. No reviewer is impersonated and no check is bypassed. P021 is merged only with the exact final PR head, all required checks successful, and a normal merge commit; the resulting integration SHA is verified on PR #25 and the live branch.

## Boundaries

Cloud results do not constitute device validation. P018 physical Gate F remains PENDING; native Wi-Fi RSSI remains DEFERRED. No physical device is accessed. Production `iOS`, fork `main`, release tags, releases, and deployment remain unchanged and out of scope. The earlier Work incidents involving `/dev/shm/beszel_health` and `bwrap` are recorded in the validation JSON; they were not repeated in P021.
