# P017 release-candidate checkpoint ledger

## State

**Status: IN PROGRESS — NOT RELEASE-READY (2026-10-18).**

The previous P016 review hold was expressly waived **by the repository owner for PR #20 only**, without representing that an independent human review happened. See [P016 decision](https://github.com/nghianguyen150612/Beszel-iOS/pull/20#issuecomment-6245377904). GitHub merged PR #20 normally at `3ed79af25a03381f4c1b8800fd7cb93e3480403d`; P016 head `bd0be1efb1be75227db68f6f6865e8461eba26a5`.

P017 branch `ios/release-candidate-validation` was created from that exact merge commit. P017 must preserve the frozen upstream `main` SHA `bb23f99b3eeaced9608e11e280e7c986dba650be` and production `iOS` SHA `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`, the 178 upstream commits, and 13 logical iOS patches.

## Checkpoint A — CI safety and regression entrypoint

- Added a dedicated **nonpublishing** `P017 Release Candidate Regression` workflow: read-only repository permission, exact candidate checkout, Go 1.27.1/Bun 1.4.0, upstream ancestry, frozen matrix, audit and shell tests, Hub migration unit tests, genuine Bun build, complete Go regression, and a source-scoped frontend manifest artifact.
- Enabled a restricted integration-target pull-request trigger on the existing native iOS build workflow. The two verification jobs now check out the exact feature HEAD rather than the synthetic GitHub PR merge revision; native diagnostics and artifact checks use `git rev-parse HEAD` for revision identity.
- The native release job retains its existing tag-only guard and must be **SKIPPED** on this branch.
- These workflow changes are **unverified until the new P017 PR checks actually run**; no PASS is claimed here.

## Required evidence before P017 can pass

1. Canonical Go and focused race tests, frontend tests, source matrix/readiness, audit and installer suites.
2. P013 synthetic v0.19 → v0.21 migration, full-state backup, rollback, and interruption recovery preserved at the P017 candidate.
3. Actual native Agent/Hub iOS ARM64 artifacts at a candidate-relevant product source revision, minimum iOS 12.0 and A7 runtime safety.
4. Downloaded asset archive digests, internal `SHA256SUMS`, exact clean VCS provenance, genuine 95-ish embedded assets (count from actual manifest), and no private MobileWiFi linkage.
5. Nonpublishing workflow isolation, upstream and iOS patch accounting.
6. Security review: reachable high `GO-2026-6608` and build-only `GHSA-vfj7-8cjw-p6xm` remain **OPEN**. Previously mitigated HTTP/2/Range risks must not regress.
7. Explicit Gate E decision based on actual logs and reviewed final artifacts. P018 physical-device validation is not part of this cloud gate.

## Safety and owner boundaries

No release tag, GitHub Release, deployment, signed installation, physical A7 test, production database change, trust-cache change, entitlement change or MobileWiFi acquisition is authorized. P012 native RSSI remains **DEFERRED — REFERENCE DEVICE UNAVAILABLE**; P018 physical checks remain **PENDING**.

If CI or quota is interrupted, continue from this same P017 branch and its existing Draft PR. Keep the gate **IN PROGRESS/BLOCKED** until all required evidence exists. Do not copy historical P016 test results into P017 as though executed on this revision.
