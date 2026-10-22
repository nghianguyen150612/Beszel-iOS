# P022 cloud release readiness

**Candidate status: NOT APPROVED FOR PRODUCTION.** This document separates observed P021 evidence from P022 implementation and pending exact-head qualification. P022 does not publish or deploy.

## Candidate identity and version

- Integration baseline: `228b50644a8f1a28c61c5f6944698bc1011d0926`.
- Pinned upstream source: `b5357343550c8e2493d0e65472a06c1007396384`; current upstream `main` was separately observed at `b97abf7662a7c7a2dd7542ea2062405abd22d4dc` (five commits ahead). P022 does not integrate that drift.
- `beszel.Version` is `0.21.0`; the candidate includes post-v0.21 upstream-main commits. This is not evidence that upstream has released a later stable version.
- Proposed Beszel-iOS tag convention: `v0.21.0-ios.1`, matching the repository's existing `v0.19.0-ios.1` format. The strict payload validator checks this format against `beszel.go`. No tag has been created.
- Latest public iOS release observed during P022 inventory: `v0.19.0-ios.1`. The installer continues to resolve that published Latest release and pin its resolved tag for the whole transaction. It does not select an unpublished P022 artifact.

## Release boundary

P022 hardens `.github/workflows/ios-build.yml`. The release job now requires a `push` event on a tag ref with the iOS tag prefix/suffix; `workflow_dispatch`, pull requests, ordinary branch pushes, and P022 branch pushes cannot enter it. A Python validator checks the exact candidate version, payload layout, Mach-O device metadata, SHA-256 manifest, same-run native metadata, Go 1.27.2 build settings, clean VCS state, and source commit before release creation. The publisher stages a draft with `--verify-tag`, downloads and revalidates the draft assets, and only then marks it published. The `contents: write` permission and `GITHUB_TOKEN` are confined to the release job and individual release steps. Missing artifacts, failed builds, wrong source provenance, checksum mismatches, or unexpected assets fail before publication.

The release workflow's generic notes no longer claim that a future candidate was tested on an iPad. The iOS workflow has no mutable `latest` build-artifact input: both binary and native validation artifacts are downloaded from the same workflow run. The public installer's GitHub Latest URL refers to the latest published release; an explicit candidate is selected by immutable source SHA, same-run workflow artifacts, and artifact ID for review. It is not installed or executed automatically. A later public release requires a separately authorized correctly formatted tag push.

### Controlled candidate staging

For review, select the exact PR head SHA and its successful native workflow run, then select the `beszel-ios-arm64` and `ios-native-validation` artifacts by their GitHub artifact IDs from that run. Download them through the authorized GitHub artifact API; do not resolve an asset through Latest. Extract the binary ZIP and native metadata into disposable staging and run:

```sh
python3 .github/scripts/verify-ios-release-payload.py <payload-directory> \
  --metadata <native-validation-directory>/native-artifacts.json \
  --tag v0.21.0-ios.1 \
  --source-sha <exact-40-character-source-sha>
```

The validator checks the proposed tag against the checked-out `beszel.go`, the exact three-file payload, checksum-to-byte correspondence, ARM64 iOS 12 Mach-O metadata, Go settings, clean VCS revision and both binaries' provenance. This stages bytes for review only. No installer command consumes them, and the public installer default remains the currently published Latest release. A future publication requires an explicitly authorized tag push whose commit is on `iOS`; CI then builds and validates that tag's same-run artifacts and verifies draft assets before publishing.

The synthetic validator tests exercise a valid candidate, the proposed `v0.21.0-ios.1` convention against the repository version, malformed tags, mismatched source revision, payload checksum failure, missing/extra assets, and metadata for different bytes. They never create a Git tag, GitHub Release, or package. Local test result: 7 passed. Workflow isolation tests cover branch pushes, pull requests, `workflow_dispatch` on a tag, the P022 branch, a non-iOS stable tag, a correctly scoped iOS release tag, same-run build dependency, credentials, and the unchanged public Latest installer default; 36 checks passed. P022 checkpoint native and regression checks passed; qualification must rerun on the final evidence-document head.

## Release candidate inventory

| Phase | Current status | Evidence boundary / remaining work |
|---|---|---|
| A — production delta | PASS | 637 paths inventoried and risk-profiled; full accounting and ancestry validators pass. |
| B — merge simulation | PASS | Clean local-only merge from exact production SHA; see [dry-run report](ios-p022-promotion-dry-run.md). |
| C — release workflow | PASS at source `3b7f14e`; final head pending | P017 release-isolation tests and the 7 synthetic payload tests passed; native release validation/publisher jobs must remain skipped on the final PR head. |
| D — artifacts and signing | PASS at source `3b7f14e`; final evidence-head rerun pending | Native iOS Build and Procursus passed. The binary and native-validation ZIPs were downloaded through the authorized GitHub artifact API; both archive digests matched GitHub metadata. The exact binary bytes match checksums and same-run metadata, Go 1.27.2, clean source revision, iOS 12 ARM64 Mach-O, and 97 embedded Hub assets. Runtime instruction scans and no-private-MobileWiFi checks passed. No Apple signing identity was used; hosted validation does not establish physical launchability or Apple trust. |
| E — installer lifecycle | PASS at source `3b7f14e`; final head pending | Exact P022 installer workflow passed bootstrap checksum, syntax, Mach-O, workflow isolation, lifecycle, patch sentinel and sync tests. Rerun on final evidence-document head. |
| F — migration and recovery | PASS at source `3b7f14e`; final head pending | P013's genuine synthetic v0.19-era migration, integrity checks, backup restore, post-migration rollback and injected interruption recoveries passed. The sanitized manifest/failure-log ZIP matched GitHub's digest. Rerun P013 on the final evidence-document head. |
| G — advisory disposition | PENDING final exact head and release decisions | P022 P017 on source `3b7f14e` found GO-2026-6608 absent, three GO-2026-6809 raw occurrences with the matching official x509 backport and passing test, one open module-level GO-2026-5932 finding, and an open high build-only GHSA-vfj7-8cjw-p6xm; zero critical JavaScript findings. Reconfirm on the final doc head and preserve explicit residual decisions. |
| H — release notes/version | PASS as proposal | EN/VI draft notes and `v0.21.0-ios.1` format prepared; neither is a published release or approval. |
| I — exact-head cloud qualification | PENDING | Required P013, P017, native iOS/Procursus and installer runs must pass on the exact final P022 PR head. The release job must remain skipped. |

## Required cloud qualifications

The final PR head must pass the full P017 Go, race, audit, installer, frontend, and portable build suite; P013 synthetic historical migration and recovery fixture; native macOS/Xcode iOS Agent and Hub build with Go 1.27.2, Apple A7 runtime inspection, iOS 12.0 metadata, clean provenance, checksums, embedded frontend and no private MobileWiFi linkage; Procursus checks; and installer lifecycle checks. The already-qualified checkpoint `3b7f14e` passed all these categories and its artifacts were independently inspected. On checkpoint source `3b7f14e`, P017, P013, installer, native iOS, and Procursus passed; both release jobs were skipped on the PR event. The binary and native validation ZIPs were independently downloaded and verified. The final evidence-document commit must trigger those workflows again on its exact PR head. The changed release simulation tests passed in P017. Final exact-head statuses are recorded after the documentation commit; no skipped required workflow counts as a pass.

P022's merge remains conditional on exact-head cloud checks, the applicable PR review/check rules, and a skipped release job. GitHub reports integration unprotected and repository rulesets empty, with no required review gate; PR #26 currently has no submitted reviews. No review approval is fabricated. Gate G remains **PENDING** until the final evidence-head checks pass. Even if Gate G later passes, it means cloud production-promotion preparation only. P018 physical Gate F remains PENDING; production rollout and public release require separate decisions.
