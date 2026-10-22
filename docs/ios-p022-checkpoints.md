# P022 durable checkpoint ledger

## Verified baseline

Execution began in `/workspace/Beszel-iOS` on 2026-10-22. Repository: `nghianguyen150612/Beszel-iOS`. Feature branch: `ios/p022-release-readiness`. Target base: `ios/upstream-v021-integration`.

| Ref | Verified SHA | Evidence |
|---|---|---|
| `ios/upstream-v021-integration` | `228b50644a8f1a28c61c5f6944698bc1011d0926` | GitHub branch ref and PR #25 merge |
| production `iOS` | `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` | GitHub branch ref |
| fork `main` | `e984ee12081f4e112601c3f718c9d81c112fe718` | GitHub branch ref |
| pinned upstream | `b5357343550c8e2493d0e65472a06c1007396384` | P021 integration record |
| current upstream `main` observation | `b97abf7662a7c7a2dd7542ea2062405abd22d4dc` | Five commits ahead of pinned target; not integrated |

P019 PR #23 merged normally as `c8bd3dba8656c8195ffacf56ff743dd58b01da25`; P020 PR #24 as `42556e62c23f3595a276d2eb4a55125a96f5d31f`; P021 PR #25 as `228b50644a8f1a28c61c5f6944698bc1011d0926`. All three are verified merged. P018 physical Gate F remains PENDING.

P018 PR #22, [P018 device validation and release authorization](https://github.com/nghianguyen150612/Beszel-iOS/pull/22), was independently found open as Draft. Its hardware gate remains blocked/pending; P022 did not contact or test a device. GitHub reported the integration branch unprotected, repository rulesets empty, and no required review gate; PR #26 currently has no submitted reviews. No review approval is fabricated.

## GitHub checkpoints

| Checkpoint | Remote SHA | Result |
|---|---|---|
| Initial P022 baseline ledger and Draft PR #26 | `9f26ffd583ed3fa6a88e6f26f4dc71a382d3067c` | Branch created from integration; Draft PR targets integration |
| Phase A production difference audit | `87e700ef646bf89efddaa2197966cbb81ab8d791` | Complete 637-path inventory, split matrix and risk profiles; uploaded blob hashes matched local files |
| Phase A checkpoint ledger | `73d6ed346132fba73a402c376276b9148f238f99` | Remote branch and PR head verified |
| Release dry-run validator and workflow isolation | `a01c586460000435515203ac8b8e51cfd5aaf100` | Exact source code and workflow/test blobs verified; four hosted workflows started on exact PR head |
| Malformed provenance rejection | `3b7f14eef0971b5e0636e98bcd7246878f89c381` | Local synthetic tests pass after correction; remote branch ref verified |

Git CLI push authentication was unavailable (`gh auth git-credential` rejected the configured credential). Durable P022 commits are therefore being created with the connected GitHub repository API, with each blob SHA compared to the local file's Git blob SHA, then branch ref and PR head checked. No fabricated push is recorded.

## Phase status

| Phase | Status | Evidence / next requirement |
|---|---|---|
| A — production difference | PASS | 637 paths in two CSVs; 15 joined profiles; 178/178 frozen upstream rows, 35/35 historical iOS commits, 13/13 logical patches; no unresolved matrix rows |
| B — production merge simulation | PASS | Detached worktree from exact production; no conflicts; local-only merge SHA `9ee85fa27fea0b5f5e3dc75407575e00dce5afcd`; resulting tree equals integration; focused sentinels pass |
| C — release workflow | PASS at source `3b7f14e`; final head pending | Tag-push gate, separate read-only validator job, write-only publisher job, same-run artifacts, draft revalidation, 7 synthetic payload tests, and 36 workflow isolation tests passed. Final release jobs must be skipped on the final PR head |
| D — artifacts and signing | PASS at `3b7f14e`; final evidence-head rerun pending | Native iOS and Procursus passed; exact Agent/Hub ZIP and native evidence ZIP downloaded and GitHub digests matched; binary checksum/provenance validator passed; no physical trust/launch claim |
| E — installer lifecycle | PASS at source `3b7f14e`; final head pending | Exact P022 installer run `39699212345` passed; rerun on final evidence-document head |
| F — migration and rollback | PASS at source `3b7f14e`; final head pending | Exact P022 P013 run `39699212188` passed; synthetic evidence ZIP digest verified; rerun on final head |
| G — security | PASS at source `3b7f14e`; final head and release decisions pending | P017 run `39699212157` passed; GO-6608 absent, x509 backport evidence and open residuals recorded in JSON; rescan final head |
| H — versioning and notes | PASS as draft | `v0.21.0-ios.1` proposal and EN/VI notes prepared; no tag or release created |
| I — cloud qualification | PASS at `3b7f14e`; final evidence-document head pending | P013, P017, native iOS/Procursus, and installer runs all passed on `3b7f14e`; release validation and publisher jobs were skipped; exact binary/native ZIPs independently verified. Final docs commit must rerun all four workflows |
| Gate G | PENDING | Do not mark PASS until all listed cloud evidence and required review are complete |

## Local simulation evidence

Production started at `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; integration source was `228b50644a8f1a28c61c5f6944698bc1011d0926`; merge base was production. The simulated tree `53d0f5abc11c4973ec27db84952a46e0611175ac` equals the integration tree. The simulated merge object is local-only, unreferenced, and was never pushed. The complete report is `docs/ios-p022-promotion-dry-run.md`.

All controlled local files and scratch output are inside `/workspace/Beszel-iOS`; test temporary directories use `/workspace/Beszel-iOS/.tmp`. No `sudo` or equivalent was used. No physical device, production branch, fork `main`, release tag, GitHub Release, package, deployment, or production database was touched.

## P022 qualification checkpoint on `3b7f14e`

| Workflow | Run ID | Result |
|---|---:|---|
| P013 Hub migration/recovery | `39699212188` | PASS; synthetic data only; artifact `12392591121` independently digest-verified |
| P017 regression/security | `39699212157` | PASS; scan and manifest artifacts independently digest-verified |
| iOS Build/Procursus | `39699212169` | PASS; release validation and publisher both SKIPPED |
| Installer lifecycle | `39699212345` | PASS |

Native artifact evidence: binary ZIP artifact `12392662907`, GitHub/downloaded SHA-256 `c276756a55f5f8cb36e521adb4323e698b10227b36ba0231e88d4bcc5fd01ea0`; native-validation ZIP artifact `12392303371`, GitHub/downloaded SHA-256 `ea630b073b9b95ba0be118d9f7dfc654d9b0f75bce0b3f5329fd8c5396dbeb5d`. Agent SHA-256 `0606d36665d6c887f51034574fe3fd7a7ca289a8b96bb58bc9f58d0faafcd18d`; Hub SHA-256 `6618e2cd3b9129d553208096a413528c9a42bead5e401124badbd19959063d45`. The independent payload validator passed exact checksum bytes, clean source VCS metadata at `3b7f14e`, Go 1.27.2 iOS ARM64 settings, iOS 12 minimum, 97 embedded assets, A7 runtime checks, and no private MobileWiFi linkage. No device launchability or Apple trust is claimed.

## Next exact work

1. Commit and remotely verify the final P022 evidence documents.
2. Confirm the P022 PR head SHA and wait for P013, P017, native iOS/Procursus, and installer workflows on that exact source. Verify the release jobs are skipped.
3. Record exact-head results in PR #26 and evaluate Gate G without changing the final evidence branch head.
4. Review current PR state and applicable branch rules. Merge only if all mandatory conditions pass; otherwise keep the PR open with exact blockers.
