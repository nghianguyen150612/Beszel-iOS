# P022 durable checkpoint ledger

## Initial verified baseline

Execution started 2026-10-21 in `/workspace/Beszel-iOS`. The repository is `nghianguyen150612/Beszel-iOS`; the P022 feature branch is `ios/p022-release-readiness`, based on `ios/upstream-v021-integration`.

GitHub branch refs observed before implementation:

| Ref | SHA | Verification |
|---|---|---|
| `ios/upstream-v021-integration` | `228b50644a8f1a28c61c5f6944698bc1011d0926` | GitHub branch API and fetched Git ref |
| `iOS` | `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` | GitHub branch API and fetched Git ref |
| `main` | `e984ee12081f4e112601c3f718c9d81c112fe718` | GitHub branch API and fetched Git ref |

The observed integration ancestry records P019 PR #23 merged normally as `c8bd3dba8656c8195ffacf56ff743dd58b01da25`, P020 PR #24 merged normally as `42556e62c23f3595a276d2eb4a55125a96f5d31f`, and P021 PR #25 merged normally as `228b50644a8f1a28c61c5f6944698bc1011d0926`. P021 pins upstream `b5357343550c8e2493d0e65472a06c1007396384`. Its recorded disposition keeps P018 physical Gate F `PENDING`.

No remote P022 branch or existing PR was returned by the initial search. The branch was created from the verified integration SHA, and the first remote commit `9f26ffd583ed3fa6a88e6f26f4dc71a382d3067c` records this initial ledger. Draft PR #26 is open at <https://github.com/nghianguyen150612/Beszel-iOS/pull/26> against `ios/upstream-v021-integration`. The local worktree's checkpoint tree matches the remote commit tree; Git CLI push authentication is unavailable in this environment, so subsequent durable commits are being made through the connected GitHub repository API and verified against the PR head.

## Qualification plan and current status

| Phase | Required evidence | Initial status |
|---|---|---|
| A — production difference | Complete integration-to-`iOS` change and compatibility matrix; 637 paths inventoried; 178/178, 35/35 and 13/13 accounting validators pass | `PASS` |
| B — promotion dry-run | Disposable merge from exact production SHA, result and focused regression evidence | `PENDING` |
| C — release workflow | Trigger/permission/artifact isolation audit and synthetic nonpublishing dry-run | `PENDING` |
| D — artifacts and signing | Exact hosted artifact provenance plus independent archive verification where available | `PENDING` |
| E — installer | Rootful lifecycle, integrity, staged candidate, and failure recovery evidence | `PENDING` |
| F — migration and rollback | Synthetic historical fixture, backup/recovery, interruption, compatibility matrix | `PENDING` |
| G — security | Current scans and advisory dispositions with reachability and release decisions | `PENDING` |
| H — versioning | Validated unpublished candidate convention and English/Vietnamese release notes | `PENDING` |
| I — cloud CI | Required suites on the exact final P022 PR source SHA | `PENDING` |
| Gate G — production promotion preparation | All cloud-only promotion criteria in PROMPT022 | `PENDING` |

Planned additive evidence files:

- `docs/ios-p022-production-diff.md`
- `docs/ios-p022-promotion-dry-run.md`
- `docs/ios-p022-release-readiness.md`
- `docs/ios-p022-release-security.md`
- `docs/ios-p022-upgrade-rollback.md`
- `docs/ios-p022-artifact-provenance.json`
- `docs/ios-p022-validation.json`

## Boundaries

P022 is cloud-only. No physical iPad is to be contacted. The production `iOS` branch and fork `main` are read-only. No release tag, GitHub Release, package publication, deployment, or production database operation is authorized. P018 physical Gate F remains `PENDING` regardless of the P022 result.

## Checkpoint history

| Checkpoint | Local source SHA | Remote source SHA | Evidence |
|---|---|---|---|
| Initial verified baseline | `228b50644a8f1a28c61c5f6944698bc1011d0926` | `9f26ffd583ed3fa6a88e6f26f4dc71a382d3067c` | Draft PR #26; baseline ledger |
| Phase A production difference audit | Pending remote phase commit | Pending remote phase commit | `ios-p022-production-diff.md` and `ios-p022-production-matrix.csv` |
