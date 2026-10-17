# P016 — Installer, privilege, and service lifecycle checkpoints

## Starting point

- Requested starting SHA: `50546d7f7468f7b07af347eb157139208b04fcbc` (P015 merge).
- Verified integration tip when P016 began: `d69a602045454ff91febd847fd3d49179cb94c7f`. The requested P015 SHA is an ancestor of this newer tip, so P016 preserves the newer P015 follow-up work.
- Feature branch: `ios/installer-privilege-lifecycle`.
- Base: `ios/upstream-v021-integration`.
- Protected refs at start: production `iOS` = `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; frozen `main` = `bb23f99b3eeaced9608e11e280e7c986dba650be`.
- No P016 branch or pull request existed at start.
- The P015 database fixture, backup/restore, current-source security, and dependency-advisory evidence is present in the integration tree.

## Checkpoint A — Lifecycle serialization and path cleanup

The lifecycle engine now serializes mutating CLI operations with a root-owned atomic directory lock. A live owner blocks a second installer; a dead PID can be isolated and recovered; malformed or symlinked locks fail closed. Service start, stop, and restart use the same lock, while read-only diagnostics remain lock-free. The layout check rejects a managed directory that is itself a symlink, and temporary cleanup only removes a `beszel-ios.*` directory directly under the configured temporary root.

Initial regression results before P016 edits:

- `sh tests/bootstrap-test.sh`: 54 passed, 0 failed.
- `sh tests/install-sh-test.sh`: 814 passed, 0 failed.
- These results are baseline evidence only; final-head results will be recorded after the checkpoint tests and later phases stabilize.

Checkpoint A validation:

- `sh tests/bootstrap-test.sh`: 54 passed, 0 failed.
- `sh tests/install-sh-test.sh`: 835 passed, 0 failed (21 new lock, mode, symlink-layout, and cleanup assertions).
- `sh -n install.sh`, `sh -n scripts/ios/install-beszel.sh`, and `sh -n tests/install-sh-test.sh`: passed.
- The bootstrap engine SHA-256 matches the edited lifecycle engine.
- `git diff --check`: passed.
- ShellCheck is unavailable in this environment.

Checkpoint status: A implementation validated; commit and remote verification are next.

## Remaining checkpoints

| Checkpoint | Scope | Status |
| --- | --- | --- |
| A | Lifecycle and filesystem safety | In progress |
| B | Full Hub data backup and transactional rollback | Pending |
| C | Bootstrap and native artifact verification | Pending |
| D | Rootful permissions and path safety | Pending |
| E | Privilege and signing review | Pending |
| F | Read-only diagnostics and service ordering | Pending |
| G | Release workflow isolation | Pending |

Device validation `IOS-INSTALL-05` remains pending P018. P012 native Wi-Fi RSSI remains deferred. P016 does not authorize release, tagging, deployment, device installation, or production database changes.
