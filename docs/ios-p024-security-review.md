# P024 security review

**Repository:** nghianguyen150612/Beszel-iOS  
**PR:** #28, base ios/upstream-v021-integration  
**Pinned upstream:** b97abf7662a7c7a2dd7542ea2062405abd22d4dc  
**Review state:** source reviewed; hosted Go/frontend/security checks passed on b6c0b5af and 01865f2; final evidence-doc checkpoint and exact-head checks pending

P024 keeps the P023 Go 1.27.3 security posture while integrating the pinned upstream dependency updates. This review records reachable findings and controls; it does not close the two existing release-owner decisions or authorize publication.

## Toolchain and x509

- go.mod remains at Go 1.27.3. The upstream Go 1.27.2 directive was not accepted.
- The P023 native Go x509 remediation remains in place. P024 does not restore the obsolete custom backport.
- P017 runs 39754342581 and 39755571369 passed on b6c0b5af and 01865f223d5c4f744b47b42b4268f4b3ced11d27 respectively. The pinned Go 1.27.3 check, native x509 source validator, full regression, and govulncheck scan passed on both. Final PR-head rerun remains required after the evidence checkpoint.
- GO-2026-6608 and GO-2026-6809 remain absent/remediated under the P023 exact-toolchain evidence. The final P024 scan must retain that result.

## Dependency changes

The P024 module graph verifies with go mod verify. The reviewed direct changes include purego v0.11.2 and shoutrrr v0.22.0; updated indirect modules include x/exp, mimetype, and godbus.

- purego is imported only by NVML files with amd64 plus Windows or glibc Linux build constraints. Those files are excluded from iOS arm64 source selection. Local iOS source-selection validation passes, and a tagged host compile covers the Linux/glibc implementation. The reviewed purego release is Apache-2.0.
- shoutrrr remains in notification delivery. The upstream update improves context-aware sending, close lifecycle, error reporting/redaction, HTTP client injection, and proxy environment handling. Proxy environment variables can affect outbound notification routing; the behavior was reviewed as an upstream change. P017 full Go regression and security CI must pass on the final head. The reviewed release is MIT.
- Updated x/exp, mimetype, and godbus module metadata and checksums are consistent under go mod verify. Their applicable BSD/MIT licenses were reviewed.
- Frontend dependency versions were not broadly upgraded. The frozen Bun lock install, tests, Lingui extraction/compile, TypeScript, and production Vite build pass locally; P017 audits both lock graphs on the candidate.

## Advisory disposition

| Advisory | P024 disposition |
|---|---|
| GO-2026-6608 | Remediated in the official Go toolchain; must remain absent from the P024 Go 1.27.3 scan. |
| GO-2026-6809 | Native Go 1.27.3 x509 fix retained and custom backport remains absent; must remain absent from the P024 scan. |
| GO-2026-5932 | Existing module-level x/crypto finding remains open. The P023 source and six-profile evidence found no OpenPGP imports/package reachability. This is not a fix. |
| GHSA-vfj7-8cjw-p6xm | Existing HIGH braces 3.0.3 build-tool finding remains open. The affected path is Lingui/chokidar/micromatch build tooling; generated assets are embedded and the Hub does not run Node tooling. P023 isolated hosted build mitigation remains. No fixed release is claimed. |

P017 run 39755571369 passed on 01865f2. The Go scan retained exactly one module-only GO-2026-5932 occurrence for golang.org/x/crypto v0.58.0; the six-profile OpenPGP check found no imported or reachable OpenPGP package. GO-2026-6608 and GO-2026-6809 remained absent. The JavaScript report retained only GHSA-vfj7-8cjw-p6xm for braces 3.0.3, with no fixed version claimed. These are raw findings with the existing P023 mitigations, not findings erased or marked fixed.

GitHub API metadata for P017 run 39755571369 records these artifact digests: Go vulnerability evidence 12415268183, SHA-256 e7dacadf682a89b3cd9a026b4ee99e3822f6a1b44a2b72c7aa1da0d480b98338; JavaScript audit evidence 12415632254, SHA-256 8a80601db1513d01dbd0d398ea62efeae52c0ef0f428e44bf7d952c1f712ce3d; workflow scan 12414869469, SHA-256 429578ec9f4ed0fb69a90e8891324862fb914672a85d37b88b13004e47cd7bf4; regression manifest 12414759710, SHA-256 b1b630e1e783eb6b5da9482614744c7711558649a2d7be1915c18ef20f84880c. The source-bound P017 manifest reports 98 assets with digest 48f9877b1f173bc7496b7302538420790248378b10dd05348b09cef50cb55180. These GitHub API digests do not establish independent archive verification. The raw reports are bound to candidate SHA 01865f223d5c4f744b47b42b4268f4b3ced11d27. Repeat the scan on the final PR head.

## Existing security controls

P022/P023 authentication, authorization, token handling, TLS validation, secret redaction, SSH/WebSocket recovery, installer checksum and rollback, artifact provenance, workflow action pinning, and release isolation remain in the regression suite. P024 adds no endpoint or authorization change.

The iOS review build is PR-only and read-only. The publishing jobs in ios-build.yml require an iOS release-tag push. P024 creates no tag or release. On the final PR run, release validation and publishing jobs must be skipped.

## Gate disposition

| Review item | State |
|---|---|
| Go 1.27.3, native x509 remediation, and dependency integrity | P017 PASS on 01865f2; final evidence-doc-head rerun pending. |
| P023 residual findings remain visible and bounded | P017 PASS on 01865f2 with only GO-2026-5932 and GHSA-vfj7-8cjw-p6xm retained; final rerun pending. |
| New reachable critical security regression | None reported by P017 on 01865f2; final rerun pending. |
| Native iOS and Apple A7 verification | Procursus passed on 01865f2; macOS/Xcode job remains queued. |
| Release isolation | Workflow conditions reviewed; final PR run must show publishing jobs skipped. |

P018 Gate F remains PENDING. No physical device was accessed. This document does not authorize public release or deployment.
