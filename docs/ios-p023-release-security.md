# P023 release security disposition

**Qualified implementation source:** fd6b1595ff2e8b4d7037229ce6d71af87f907b4e
**PR:** [#27](https://github.com/nghianguyen150612/Beszel-iOS/pull/27)
**Cloud qualification:** implementation source passed; the final documentation-only PR head must pass exact-head checks before merge

P023 is a cloud security milestone. A PASS qualifies the integration candidate only. It does not authorize a public release, close P018 Gate F, or replace release-owner security decisions.

## Advisory dispositions

| Advisory | Disposition | Open release decision |
|---|---|---|
| GO-2026-6608 | REMEDIATED_AND_VERIFIED on Go 1.27.3; absent from the exact-version source scan | None identified |
| GO-2026-6809 | REMEDIATED_AND_VERIFIED; native Go 1.27.3 x509 fix verified, custom backport no longer applied | None identified for this advisory |
| GO-2026-5932 | NOT_REACHABLE_WITH_EVIDENCE; module-only warning remains and no OpenPGP package is imported or reachable in reviewed profiles | Release owner must accept or require removal of the unmaintained module-level residual |
| GHSA-vfj7-8cjw-p6xm | MITIGATED_WITH_RESIDUAL_RISK; high-severity braces package remains in build tooling and is not established as a production runtime dependency | Release owner must accept the build-time denial-of-service residual or require a verified upstream fix |

Raw scanner state and evidence are in the [advisory matrix](ios-p023-advisory-matrix.json), [Go migration report](ios-p023-go1273-migration.md), and [JavaScript dependency review](ios-p023-javascript-dependency-review.md).

## Exact-source hosted gates

The implementation-source runs are:

- [P017 regression/security 39718772365](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39718772365): PASS, full Go suite and portable Agent/Hub builds, raw Go and JavaScript scans, six-profile OpenPGP check, actionlint, ShellCheck, zizmor, release isolation, and ancestry.
- [Native iOS/Procursus 39718772276](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39718772276): PASS, Go 1.27.3 A7/x509 validation, native Agent/Hub builds, Mach-O, VCS provenance, asset embedding, checksums, no private MobileWiFi linkage, Procursus; release validation and publish jobs skipped for the PR.
- [P013 genuine migration 39718772288](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39718772288): PASS.
- [Installer lifecycle 39718772319](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39718772319): PASS.
- [Helm workflow 39718772278](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39718772278): PASS; no changed chart was published.

All five runs are bound to source fd6b1595ff2e8b4d7037229ce6d71af87f907b4e. The documentation-only checkpoint is required to rerun the exact-head workflows before merge.

## Artifact integrity and provenance

The three native artifacts were independently downloaded and matched the GitHub API digests:

| Artifact | ID | SHA-256 |
|---|---:|---|
| Agent/Hub iOS payload | 12400817906 | 73b8080826c337b70588c023cea93ccc6b1ce222a0a375807095d560eb963d30 |
| Native validation evidence | 12400214755 | 03abd713ee11bafddb76174683d9a5badc652f7976b8474312b105e500cd63ef |
| Disabled Wi-Fi diagnostic | 12400947269 | 5a3dd663a211ca6afd5b6c497167010abb8d820f0616dfd72421be0a65c17cb6 |

The independent validator verified Agent SHA-256 5676c4b49a0312339c0697cdf490cbda16177bebda7d311cc47a99a8fdd8719f and Hub SHA-256 2d63fa87486f0c455122886f7c9b262a0c465586d27940a8d2f3025e5592b850 against their payload manifest and actual bytes. Both Go build records identify clean source fd6b1595ff2e8b4d7037229ce6d71af87f907b4e. The Hub embeds all 97 frontend assets. Artifact records do not claim physical device launchability.

The P017 Go, JavaScript, workflow, and regression manifest artifacts also matched their GitHub API SHA-256 values. Raw scanner reports remain available from the linked P017 run. They retain the open GO-2026-5932 and GHSA findings.

## Release pipeline controls

- Release and Docker publishing workflows are restricted to release tag pushes. P017 workflow-isolation regressions verify PR, branch, ordinary tag, iOS tag, and dispatch behavior.
- PR qualification workflows use read-only repository permissions and checkout with persisted credentials disabled.
- Installer selection, checksum verification, same-run provenance, controlled rollback, and production-data preservation remain covered by installer lifecycle tests.
- The PR native release validation and release jobs were skipped. No release was created.
- P023 changes no production iOS branch, fork main, database, physical device, tag, or public release.

## Gate H and authorization

| Gate H condition | Result on implementation source fd6b1595 | Evidence |
|---|---|---|
| Go 1.27.3, A7 runtime, native x509 fix | PASS | Native run 39718772276 and P017 run 39718772365 |
| Current advisory dispositions and safe remediations | PASS with bounded residuals | P017 raw Go/npm/Bun/OSV evidence; GO-2026-5932 and GHSA remain visibly OPEN |
| No introduced critical security regression | PASS | P017 exact-source scan and regression suite; no critical finding reported |
| Full applicable Go suite and frontend suite | PASS | P017; frontend build, localization, and embedded asset checks |
| Genuine Hub migration and installer lifecycle | PASS | P013 39718772288; installer 39718772319 |
| Native iOS ARM64 Agent and Hub | PASS | Native 39718772276; Go 1.27.3, iOS 12.0, A7 instruction and provenance checks |
| Artifact integrity and provenance | PASS | Three native archives matched GitHub digests; independent payload manifest/build-info validation passed |
| Release publishing remains isolated on PRs | PASS | Native release validate/publish jobs skipped; P017 isolation regression passed |
| Historical upstream and iOS changes preserved | PASS | 41 upstream commits, 178 rows, 35 historical iOS commits, 13 logical patchsets |
| P022 Gate G protections and open release-owner decisions | PASS | Existing auth, transport, installer, artifact, and release-isolation regressions pass; both decisions remain OPEN |
| Production branches, data, and devices unchanged | PASS | Remote refs unchanged; no production data or physical device accessed |

Gate H is PASS for the Go 1.27.3 implementation at fd6b1595. The final documentation-only PR head must also pass exact-head workflows before merge, and PR publishing jobs must remain skipped.

The two release-owner decisions remain OPEN. P018 Gate F remains PENDING. Release authorization is NOT_GRANTED.
