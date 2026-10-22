# P023 checkpoints

## Checkpoint 0 — verified baseline and durable start

- Starting integration was fd334076f53d8354a4924a4c734f29cf65a43879. PR #26 was merged and Gate G passed.
- Production iOS, fork main, pinned upstream, and live upstream refs were checked. Live upstream main b97abf7662a7c7a2dd7542ea2062405abd22d4dc remained five commits ahead of pinned snapshot b5357343550c8e2493d0e65472a06c1007396384; those commits are excluded.
- Created ios/p023-security-hardening, pushed the initial security baseline, and opened Draft PR [#27](https://github.com/nghianguyen150612/Beszel-iOS/pull/27) targeting ios/upstream-v021-integration.

## Checkpoint 1 — Go 1.27.3, x509, and A7 implementation

Implementation source fd6b1595ff2e8b4d7037229ce6d71af87f907b4e is pushed and remote-verified. Go 1.27.3 is pinned across the toolchain contract and release/build validators. The A7 runtime patch is guarded by exact Go 1.27.3 source hashes and source shape. The official Go x509 fix is native and the old backport is not applied.

The six-profile OpenPGP import/dependency guard retains the single raw GO-2026-5932 module finding and fails closed on changed findings. Bun/npm/OSV scans retain the real braces advisory and no fake package fix is claimed.

## Checkpoint 2 — GitHub Actions supply-chain hardening

All 19 unique action release tags across 54 use sites are pinned to upstream-resolved full commit SHAs. P023 narrowed write permissions, removed reusable release/native build caches, disabled checkout credential persistence, moved shell outputs through environment variables, and uses short-lived GIT_ASKPASS helpers for the required push workflows.

Initial zizmor findings were 72: 51 mutable action refs, 9 credential-persistence findings, 6 cache exposure findings, 4 excessive permission findings, and 2 shell interpolation findings. After remediation, zizmor reports zero findings. Hosted actionlint revealed shell warnings missed by the initial local setup. P023 fixed SC2129 and heredoc indentation, then SC2016, and finally pinned ShellCheck 0.11.0 explicitly in the workflow scanner. The first hosted cycle with the pinned analyzers passed on fd6b1595.

## Checkpoint 3 — exact-source hosted qualification

All required workflows passed on source fd6b1595ff2e8b4d7037229ce6d71af87f907b4e:

| Workflow | Run | Result |
|---|---:|---|
| P017 regression/security | 39718772365 | PASS |
| Native iOS and Procursus | 39718772276 | PASS; release validation/publish jobs skipped |
| P013 Hub migration/recovery | 39718772288 | PASS |
| Installer lifecycle | 39718772319 | PASS |
| Helm workflow | 39718772278 | PASS; no chart published |

P017 source ancestry reports 41 pinned upstream commits, 178 frozen historical rows, 35 historical iOS commits, and 13 logical iOS patchsets. The local frontend asset manifest matches the hosted manifest at 97 files.

The native payload, native validation, and disabled Wi-Fi diagnostic archives were downloaded independently and all matched GitHub artifact SHA-256 digests. The Agent/Hub checksum manifest passed. Independent Go build metadata and Mach-O validation passed for source fd6b1595, Go 1.27.3, iOS ARM64, GOARM64 v8.0, CGO_ENABLED=1, and minimum iOS 12.0. Hub embeds 97 assets. Both runtime reports pass with no counter-register access, and Agent/Hub have no private Wi-Fi framework linkage. The disabled diagnostic is separate and makes no private calls. No physical device was tested.

The docs checkpoint containing these reports must be pushed, then P017, native iOS, P013, installer, and Helm checks must be verified again on that final PR head before marking the PR ready or merging.

## Security disposition and boundaries

GO-2026-6608 and GO-2026-6809 are REMEDIATED_AND_VERIFIED on the official Go 1.27.3 toolchain. GO-2026-5932 remains a module-only finding and is NOT_REACHABLE_WITH_EVIDENCE. GHSA-vfj7-8cjw-p6xm remains open and HIGH with build-only isolation mitigation. The release-owner decisions for GO-2026-5932 and GHSA-vfj7-8cjw-p6xm remain OPEN.

No production iOS branch, fork main, production database, physical device, tag, or public release was changed or accessed. No sudo was used. One early local audit test invocation created short-lived Python temporary fixtures outside the allowed workspace; Python cleaned them. This boundary incident is recorded in the validation JSON. No other workspace violation was recorded. P018 Gate F remains pending; public release authorization is not granted.
