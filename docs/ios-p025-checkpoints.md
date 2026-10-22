# P025 Checkpoint Log

This file separates preliminary remote observations from final P025 outcomes. No result below is a P025 qualification pass.

## Checkpoint 0 — baseline verification

- GitHub repository: `nghianguyen150612/Beszel-iOS` (public fork; authenticated GitHub connection reports push permission).
- Integration branch fetched from GitHub: `07f6fd5f87c99348ec33549fe8dfed099464eb7f`; tree `e1cd43c81b131c05c4e4dfcde40d598dc962b330`.
- P024 PR #28: merged; merge commit `07f6fd5f87c99348ec33549fe8dfed099464eb7f`, parents `8e03afadb3a03480414fa47f87d40c13331e05b3` and `13c63387c588f65e288449ba611e5a83c6d6eef7`.
- P024 native, P017 regression, P013 migration, and installer runs: all completed successfully on exact P024 feature head `13c63387c588f65e288449ba611e5a83c6d6eef7`; IDs are in `ios-p025-validation.json`.
- P024 native run `39759219855` artifacts: payload ID `12418500671`, GitHub-reported digest `sha256:c74e9916186998df60462846cf1445fb701ca3b94849802127e8fe1e5e28228e`; native validation ID `12417553816`, reported digest `sha256:445971fe64d41798bcacbafddd6ef8742ed4559629d87ca7ed70f2d0e667bc66`. Authenticated local download of both returned HTTP 403. GitHub metadata digests are not independent archive verification.
- Production `iOS`: `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; fork `main`: `e984ee12081f4e112601c3f718c9d81c112fe718`.
- Current published iOS release verified from GitHub Releases: `v0.19.0-ios.1`.
- Initial workspace checkout was the old `iOS`/P003 commit. It is preserved; P025 work is isolated in `.p025-worktrees/ios-p025-artifact-rc-freeze`.
- Initial P025 commit `e90427e6a61553b3d7ca1db5aba5dd18473c92db`, tree `de2e7cdda3906c8f372602e50675fb972d0975f8`; remote `git ls-remote` matched. Draft PR #29 targets integration.
- P025 artifact consumer and reproducibility comparison are implemented in the current local worktree; local focused tests pass. The implementation checkpoint push and hosted CI are pending.

## P025 implementation checkpoint (local, not yet pushed)

The workflow change adds a read-only, separate-host artifact consumer, downloads the payload and native-validation artifacts by explicit same-run ID, computes the raw ZIP SHA-256 through the GitHub Actions artifact API when available, validates extracted bytes, checks Go/Mach-O/iOS/frontend/A7/x509/source-selection provenance, and emits a sanitized manifest. If direct ZIP access returns HTTP 403 while `actions/download-artifact` succeeds, the consumer records the archive hash as unverified and still validates the actual hosted-downloaded payload; it never substitutes the GitHub-reported digest as an independently computed archive hash.

Two native builds with separate output directories and Go build caches are recorded with input digests, binary hashes, build IDs, differing-byte offsets/values/sections, and frontend asset fingerprints. The builds share one hosted runner; this limitation will be stated in the final reproducibility report.

The P026 read-only handoff plan now specifies the reference device, immutable candidate identity requirements, artifact transfer verification, rootful install layout, LaunchDaemon labels/configuration, signing prerequisites, controlled rollback, operator warnings, and test cases. It explicitly notes that normal installer install/update commands still resolve current GitHub Releases `Latest` and do not select an unpublished P025 candidate.

Local focused checks on the current worktree: 11 artifact-consumer unit tests, 3 reproducibility-manifest unit tests, 24 native build-contract tests, 41 release workflow-isolation checks, YAML parsing for the three changed workflows, JSON parsing for the P025 records, and `git diff --check` all pass. Exact P025 hosted workflows have not yet run.

## Subsequent checkpoints

Append phase, exact source SHA/tree SHA, tests and run IDs, remote SHA verification, artifact IDs/digests, and blockers after each completed phase. Never convert pending or unexecuted work into PASS.
