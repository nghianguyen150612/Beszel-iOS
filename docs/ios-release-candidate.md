# P017 — Beszel-iOS v0.21.0 release candidate (cloud validation)

## Decision
**Gate E: PASS (cloud-only).** P016 ordinary merge: `3ed79af25a03381f4c1b8800fd7cb93e3480403d`. Candidate product source is unchanged from P016: P017 commits modify only CI workflows and documentation. The candidate binary-tested source is `5a7351f752376e206a3e1809b3698bb03cc87b98`. P017 after this source adds evidence-only documentation; no release/tag/deployment is authorized.

## Proven candidate and hosted regressions
- [P017 candidate regression](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39095687534): SUCCESS. Genuine Bun 1.4.0 frontend build, 95 assets, complete canonical Go 1.27.1 regression, 873 installer assertions, 54 bootstrap, 45 patch sentinels, 10 upstream-sync, 18 workflow isolation tests, audit and migration unit tests.
- [P017 native iOS build](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39095687398): SUCCESS, release job SKIPPED. Platform iOS 2, ARM64, deployment minimum iOS 12.0. Runtime A7 counter-read avoidance check PASS.
- [P017 installer](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39095687704): SUCCESS.
- [P016 genuine migration fixture](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39028505022): SUCCESS on P016 product source `bd0be1efb1be75227db68f6f6865e8461eba26a5`. P017 has not changed any product source or installer. **The real-version P013 fixture was not rerun on the P017 documentation/workflow-only commit.**

## Independent artifact checks
Downloaded the P017 GitHub Actions ZIP and native-validation ZIP, compared their archive digests to GitHub artifact metadata; both match. Extracted binaries were checked against uploaded SHA256SUMS. Independently inspected both binaries using llvm-objdump: Mach-O ARM64, MH_EXECUTE, LC_BUILD_VERSION iOS, minOS 12.0, SDK 17.5. The native validation artifact agrees on clean Go 1.27.1 VCS source `5a7351f`; A7 runtime instructions have no counter access, and binary linkage manifest has no MobileWiFi/CoreWLAN/Apple80211 private imports. Embedded Hub assets: 95. Detailed immutable digest evidence: `docs/ios-release-candidate-artifacts.json`.

## Security and release policy
Open HIGH advisories: `GO-2026-6608` (reachable multipart MIME Go 1.27.1; fix requires Go 1.27.2) and `GHSA-vfj7-8cjw-p6xm` (build-only braces). These are **not fixed**. Keep Hub/Agent on trusted networks and retain HTTP/2 disabling and Range limits; require explicit risk acceptance before any actual public release. No unmitigated critical advisory identified under documented P015 mitigations.

## Gate boundary
Gate E validates a reproducible cloud **candidate**, not on-device readiness. P018 Gate F must validate the reference iPad mini 2 A1489, Apple A7, iOS 12.5.7, Amethyst/Procursus rootful, binary launch (no SIGILL), services, reboot/jailbreak reactivation, current telemetry, Hub health, transport, backup/restore, and operator approval. Native iOS Wi-Fi RSSI remains DEFERRED. No tag, GitHub Release, production branch change or device deployment authorized.

## PR review transparency
Repository owner requested merge and advancement to P018 in chat; this grants an owner-directed exception to the previously requested independent review process for this PR only. It is **not an independent reviewer approval**. Platform branch protections still apply. Preserve normal merge commit history.
