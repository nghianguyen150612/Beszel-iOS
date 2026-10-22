# P022 production promotion dry-run

**Status:** the disposable Git merge simulation and focused static regressions pass. The simulated commit is local-only evidence and is not a production commit.

## Inputs and result

| Item | SHA / result |
|---|---|
| Production starting SHA | `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` |
| Integration source SHA | `228b50644a8f1a28c61c5f6944698bc1011d0926` |
| Merge base | `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` |
| Simulated merge tree | `53d0f5abc11c4973ec27db84952a46e0611175ac` |
| Local-only simulation merge SHA | `9ee85fa27fea0b5f5e3dc75407575e00dce5afcd` |
| Result tree equals integration tree | PASS (`53d0f5…`) |
| Automatic merge conflicts | None |
| `git diff --cached --check` | PASS |
| Case-insensitive path collisions | None observed in production, integration, or result tree |

The simulation used a detached worktree at `.tmp/p022-production-dry-run`, created from the exact production SHA, then ran the equivalent `git merge --no-commit --no-ff 228b50644a8f1a28c61c5f6944698bc1011d0926`. The merge completed without conflict. `git commit-tree` created a two-parent commit object solely to record the simulated result. It has no branch or tag reference and was not pushed.

## Tree delta

The candidate result changes 637 paths relative to production: 368 added, 266 modified, two detected renames, and one deletion. The deleted `internal/migrations/1788955200_pool_updates.go` is accounted for by the frozen upstream v0.21 snapshot audit: its pool updates were folded into the snapshot. The renames are the v0.20 collection snapshot and the network-monitor certificate migration. The complete path list and per-path risk profiles are in [the production matrix](ios-p022-production-matrix.csv), [part two](ios-p022-production-matrix-part-2.csv), and [the joined profiles](ios-p022-promotion-profiles.json).

No production bootstrap, rootful installer default, version metadata, Agent/Hub name, or launch service is removed by merge resolution: the resulting tree is exactly the integration tree. The integration source does change these systems relative to production, and their compatibility and rollback risks remain documented in the matrix and [upgrade review](ios-p022-upgrade-rollback.md).

## Focused checks on the simulated result

All checks below ran from the detached simulated merge worktree, whose tree is byte-for-byte the integration tree:

- `python3 scripts/audit/validate-upstream-v021-matrix.py --repo .` — PASS, 178/178 frozen upstream rows, 35/35 historical iOS commits, 13 logical iOS patches, no unresolved rows.
- `python3 scripts/audit/validate-upstream-v021-readiness.py --root .` — PASS, 24 acceptance cases and 33 compatibility scenarios.
- `sh tests/ios-patch-sentinels.sh` — PASS, 45 checks.
- `TMPDIR=/workspace/Beszel-iOS/.tmp/test-tmp sh tests/upstream-sync-test.sh` — PASS, 10 checks.
- `python3 tests/ios-workflow-isolation-test.py` — PASS, 18 checks for the baseline workflow at integration SHA. P022 adds separate tests for its hardened release path.

No Go, frontend, database, installer, native build, or device behavior is inferred from a clean Git merge. Those qualification suites are separately scheduled on the exact P022 PR head. No merge conflict resolution was needed. No production ref, release tag, GitHub Release, deployment, database, package, or device was changed.

## Promotion disposition

The merge simulation is a PASS. The future operator must still re-simulate from the then-current production SHA before any separately authorized production promotion; this P022 local-only commit is not suitable for cherry-picking or deployment.
