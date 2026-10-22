# P024 checkpoint log

Repository: `nghianguyen150612/Beszel-iOS`

Branch: `ios/p024-upstream-delta-low-memory`

PR base: `ios/upstream-v021-integration`

## Initial audit checkpoint

- Verified starting integration: `8e03afadb3a03480414fa47f87d40c13331e05b3`.
- P023 feature head: `998283532041d21a1b2dbf5ef673441cc0340cde`.
- P023 normal merge commit: `8e03afadb3a03480414fa47f87d40c13331e05b3`.
- Pinned upstream base: `b5357343550c8e2493d0e65472a06c1007396384`.
- Pinned upstream target: `b97abf7662a7c7a2dd7542ea2062405abd22d4dc`.
- Remote P024 branch was created from the exact integration SHA. No earlier P024 branch or PR existed.
- The local `work` checkout at production iOS SHA `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` was left untouched. P024 work is isolated in `.p024-worktree`.
- The five upstream commit objects, ordered ancestry, and 23 changed paths were verified locally. Source merge and implementation have not started.
- The five-commit audit and semantic conflict inventory are in `docs/ios-p024-upstream-audit.md`.
- Initial audit checkpoint: `adfdad88e315d2ec191e4ef7f18ea21a799d32b8`, parent `8e03afadb3a03480414fa47f87d40c13331e05b3`. `git ls-remote` confirmed the feature ref at the exact commit.
- Draft PR: [#28](https://github.com/nghianguyen150612/Beszel-iOS/pull/28), base `ios/upstream-v021-integration`; PR head matched the confirmed remote SHA at creation.

## Normal upstream merge checkpoint

- Merge commit: `e2b677769dd47ff97b5578fb4069372b0f2f79e4`.
- Parents, in order: P024 head `b399faf01065bc89452d2d903cc5edf7d3f3e1bf`; pinned upstream `b97abf7662a7c7a2dd7542ea2062405abd22d4dc`.
- The merge preserves the pinned five-commit chain from `b5357343550c8e2493d0e65472a06c1007396384`; `git rev-list` reported 5 commits in that range. The upstream target and prior base are ancestors of the merge commit.
- `go.mod` retains `go 1.27.3`; dependency downloads completed and `go mod verify` reported `all modules verified`.
- Focused Go tests passed with local Go 1.27.3: Agent memory/certificate cases, system serialization, Hub memory migration, and record averaging. Their logs are in the workspace-local `.p024-runtime/test-artifacts/` directory.
- Bun 1.4.0 `install --frozen-lockfile` installed 385 packages without lockfile changes. Lingui extraction, catalog compilation, TypeScript, and Vite production build passed. Vite reported a mixed dynamic-import warning and large-chunk warnings.
- Hub and records tests initially could not set up because embedded `internal/site/dist` was absent. After the frontend build generated that directory, the focused tests passed.
- `git ls-remote` confirmed remote P024 at `e2b677769dd47ff97b5578fb4069372b0f2f79e4` immediately after push.

## Qualification state

The normal merge checkpoint and focused tests are complete. Full compatibility fixes, history-preservation checks, full suites, hosted CI, Gate I, and PR merge remain pending. P023 Gate G and Gate H remain inherited as PASS from the merged P023 record. P018 Gate F remains PENDING.
