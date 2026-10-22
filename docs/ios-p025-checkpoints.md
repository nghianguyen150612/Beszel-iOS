# P025 Checkpoint Log

This file separates preliminary remote observations from final P025 outcomes. No result below is a P025 qualification pass.

## Checkpoint 0 — baseline verification

- GitHub repository: `nghianguyen150612/Beszel-iOS` (public fork; authenticated GitHub connection reports push permission).
- Integration branch fetched from GitHub: `07f6fd5f87c99348ec33549fe8dfed099464eb7f`; tree `e1cd43c81b131c05c4e4dfcde40d598dc962b330`.
- P024 PR #28: merged; merge commit `07f6fd5f87c99348ec33549fe8dfed099464eb7f`, parents `8e03afadb3a03480414fa47f87d40c13331e05b3` and `13c63387c588f65e288449ba611e5a83c6d6eef7`.
- P024 native, P017 regression, P013 migration, and installer runs: all completed successfully on exact P024 feature head `13c63387c588f65e288449ba611e5a83c6d6eef7`; IDs are in `ios-p025-validation.json`.
- Production `iOS`: `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; fork `main`: `e984ee12081f4e112601c3f718c9d81c112fe718`.
- Initial workspace checkout was the old `iOS`/P003 commit. It is preserved; P025 work is isolated in `.p025-worktrees/ios-p025-artifact-rc-freeze`.
- P025 branch checkpoint commit and remote verification: pending.

## Subsequent checkpoints

Append phase, exact source SHA/tree SHA, tests and run IDs, remote SHA verification, artifact IDs/digests, and blockers after each completed phase. Never convert pending or unexecuted work into PASS.
