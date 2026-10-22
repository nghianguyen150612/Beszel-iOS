# P023 release security disposition

**Reviewed candidate code commit:** `f7a4525aeb7513163439b56b330ddfbd527c4da8`\
**PR:** [#27](https://github.com/nghianguyen150612/Beszel-iOS/pull/27)  
**State:** Draft; Gate H pending exact-head cloud qualification

P023 is a cloud security milestone. Its PASS does not authorize a public
release, close P018 Gate F, or replace release-owner security decisions.

## Current disposition

| Advisory | P023 disposition | Open release decision |
|---|---|---|
| GO-2026-6608 | `REMEDIATED_AND_VERIFIED` on Go 1.27.3; absent from the exact-version source scan | None identified |
| GO-2026-6809 | `REMEDIATED_AND_VERIFIED`; native Go 1.27.3 x509 fix verified, custom backport no longer applied | None identified for this advisory |
| GO-2026-5932 | `NOT_REACHABLE_WITH_EVIDENCE`; module-only warning remains, no OpenPGP package is imported or reachable in reviewed profiles | Release owner must accept or require removal of the unmaintained module-level residual |
| GHSA-vfj7-8cjw-p6xm | `MITIGATED_WITH_RESIDUAL_RISK`; high-severity braces package remains in build tooling and is not established as a production runtime dependency | Release owner must accept the build-time DoS residual or require a verified upstream fix |

Raw audit state and test evidence are in
[`ios-p023-advisory-matrix.json`](ios-p023-advisory-matrix.json),
[`ios-p023-go1273-migration.md`](ios-p023-go1273-migration.md), and
[`ios-p023-javascript-dependency-review.md`](ios-p023-javascript-dependency-review.md).
The latest npm audit completed and reports five high-severity affected package
entries for one braces GHSA. Its proposed major Lingui 6.10.0 migration was
not treated as a vulnerability fix. A prior npm registry timeout remains
recorded with its raw error; only that exact error enables the full-lock OSV
fallback. The current source-bound OSV cross-check found the same single GHSA
and no fixed version.

## Release pipeline controls

- Release and Docker publish workflows run only on release tag pushes. P017
  workflow-isolation regressions confirm PR and ordinary-branch events do not
  publish.
- PR qualification workflows have read-only repository permissions and do
  not persist checkout credentials. They upload expiring review artifacts.
- Installer artifact selection, checksum verification, source/run provenance,
  and rollback controls remain covered by P017 and installer lifecycle tests.
- P023 changes no tag, published release, production `iOS` branch, fork
  `main`, database, or physical device.

## GitHub Actions supply chain

The initial workflow scan found 72 static findings. P023 pinned all 54 action
use sites to 19 commit SHAs resolved from upstream release tags, narrowed
permissions, removed reusable release/native build caches, and stopped
persisting checkout credentials. The final local `actionlint v1.7.12` run passed and `zizmor v1.31.1`
reported zero findings. The first hosted scan on 304cc failed actionlint on two
shell lint findings, which were fixed and pushed at f7a4525. P017 is rerunning
both scans on that exact source and retains raw output. Zizmor's workflow scan is not
a complete CVE scan of bundled action runtimes; action-source advisory coverage
remains bounded by its online rules.

## Gate H and authorization

Gate H remains pending until the exact PR head passes P017 Go/security and
frontend checks, P013 migration, installer lifecycle, native macOS iOS
qualification, artifact integrity/provenance, ancestry preservation, and PR
release isolation. Hosted run IDs, artifacts, and final SHA are tracked in
[P023 validation](ios-p023-validation.json).

If Gate H passes, it qualifies only the cloud integration candidate. The two
release-owner decisions above remain open. P018 Gate F remains pending. This
document does not approve Ready-for-Review or merge.
