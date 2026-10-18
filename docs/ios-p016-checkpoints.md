# P016 — Installer, privilege, and service lifecycle checkpoints

## Starting point

- Requested starting SHA: `50546d7f7468f7b07af347eb157139208b04fcbc` (P015 merge).
- Verified integration tip when P016 began: `d69a602045454ff91febd847fd3d49179cb94c7f`. The requested P015 SHA is an ancestor of this newer tip, so P016 preserves the newer P015 follow-up work.
- Feature branch: `ios/installer-privilege-lifecycle`.
- Base: `ios/upstream-v021-integration`.
- Protected refs at start: production `iOS` = `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; frozen `main` = `bb23f99b3eeaced9608e11e280e7c986dba650be`.
- No P016 branch or pull request existed at start.
- The P015 database fixture, backup/restore, current-source security, and dependency-advisory evidence is present in the integration tree.
- Draft PR [#20](https://github.com/nghianguyen150612/Beszel-iOS/pull/20) targets `ios/upstream-v021-integration` from `ios/installer-privilege-lifecycle`.

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

Checkpoint status: A implementation validated, committed, pushed, and verified on the remote branch.

## Checkpoint B — Complete Hub backup and rollback

The Hub update now uses one transaction for the Hub and any installed Agent writer. It quiesces Agent before Hub, waits for prior processes to exit, verifies a complete snapshot and its extracted inventory, journals the transaction phase and prior service state, then stages and applies the candidate. Hub health and Agent connectivity must both pass before installation state is committed. Recovery restores the verified data, binary, plist, state, manager, logs, and metadata before any prior service is started. Uncertain recovery keeps both services stopped and retains the journal for retry.

The fault-injection suite mutates synthetic database and SSH identity sentinels during candidate Hub launch. Failed migration/health, state commit, journal commit, archive restore, and prior-service health cases verify complete rollback or safe stopped/pending behavior. The installer fault-injection database is a mock and does not claim SQLite execution; genuine PocketBase migration and rollback evidence remains in the P013 hosted fixture record.

Checkpoint B focused validation:

- `sh tests/install-sh-test.sh`: 873 passed, 0 failed.
- `sh tests/bootstrap-test.sh`: 54 passed, 0 failed; the pin is `3ba6c89f43da5b43666c2a2f3f53c2f7cf67cc63c80ae9e7f7ea363b2fb7a7cb`.
- `sh tests/ios-macho-validation-test.sh` with genuine P015 Agent/Hub binaries: 11 passed, 0 failed.
- `python3 tests/ios-workflow-isolation-test.py`: 18 passed, 0 failed; all workflow YAML files parse with PyYAML.
- Go 1.27.1 focused tests passed for `./agent/health ./agent ./internal/cmd/agent`, including authenticated connection state and disconnect cleanup.
- Full Go 1.27.1 regression passed at source `04747f6cb94faa0160328256f437e4bdf07bae9b`: `env -u NETWORK go test -tags 'testing no_ui' ./... -count=1 -timeout=20m`. The first attempt used an old ignored `internal/site/dist` from the workspace and correctly failed the embedded asset check; rebuilding with the workflow-pinned Bun 1.4.0 restored the expected 95-asset manifest, after which the complete command passed.
- The pinned frontend workflow commands also pass locally: Bun 1.4.0 frozen install, catalog sync, UI tests, and production build. Generated locale edits were restored; `dist` is ignored and not committed.
- Upstream matrix/readiness validators and 67 audit unit tests pass; patch sentinels pass 45 checks; upstream sync passes 10 checks.
- P015 run `39014286778`, source `b83e179bd84b2177cd7fc8b5b0f5460d5f2411ba`, artifact `12093514420`; manifest SHA-256 validates. Agent SHA-256 `7a0c5c17891fc7298f3f3b326338a7b4f7a1c4df5dcff670662fc41d42386a0e`; Hub SHA-256 `fde89ace4a39c3204c3cc07f30d7f548f59ca4aaa22145a8047a3179fdb9ec2b`.
- `docs/ios-installer-rollback-evidence.md` distinguishes installer fault injection from the genuine P013 PocketBase fixture and from P018 device validation.
- Final hosted checks passed at source `ea8dd4881964bd4527550a0c412a9ddc38c51013`: iOS Build run `39027357939`, installer run `39027349785`, and genuine P013 migration fixture run `39027349778`. The iOS Build release job was skipped for the branch dispatch. The P013 artifact `12099315917` SHA-256 matched GitHub's artifact digest and its manifest records migration, full rollback, and interrupted-migration recovery.
- The verified P016 native payload is artifact `12099471655` (artifact SHA-256 `7e319c6c79cc29fd905b4e0f621da66463799de917b6d31737395239e62bab86`); validation metadata is artifact `12099755587`. Agent SHA-256 is `fb7d961e4774dd8265150e1982f4e2df409ac8b1d73804c308379b972183615c`; Hub SHA-256 is `ab97d19781bc2056e147a5e005010c32509c2ad892791a5469cbe3fc1da1811f`. Both are clean-revision Go 1.27.1 `GOOS=ios GOARCH=arm64 GOARM64=v8.0 CGO_ENABLED=1` builds, iOS platform 2, minimum iOS 12.0; Hub embeds 95 assets. The uploaded ZIP digest and internal `SHA256SUMS` were checked locally.

## Remaining checkpoints

| Checkpoint | Scope | Status |
| --- | --- | --- |
| A | Lifecycle and filesystem safety | Complete; remote commit `3885cd71027abd9b17380000aa88f941171d371c` |
| B | Full Hub data backup and transactional rollback | Implemented; 873 installer tests pass; final installer and genuine P013 fixture passed at `57dbc0a` |
| C | Bootstrap and native artifact verification | Bootstrap tests and genuine P015 fixtures pass; P016 hosted Agent/Hub artifacts at `57dbc0a` pass checksum, Mach-O, platform, and minOS checks |
| D | Rootful permissions and path safety | Owner, mode, link, parent, traversal, state-write, and concurrency fixtures pass at the tested source |
| E | Privilege and signing review | Audit written in `docs/ios-privilege-and-signing-review.md`; mocked signing only; runtime/trust behavior remains P018 |
| F | Read-only diagnostics and service ordering | CLI/service fixtures and hosted installer pass; Agent update gate uses authenticated transport evidence; reboot/reactivation remains P018 |
| G | Release workflow isolation | 18 static event and credential-scope checks pass; the P016 branch build's release job was skipped |

`IOS-INSTALL-01` through `IOS-INSTALL-04` pass their non-device criteria. `IOS-INSTALL-05` remains pending P018. P012 native Wi-Fi RSSI remains deferred. Gate E still requires P017. P016 does not authorize release, tagging, deployment, device installation, or production database changes. PR #20 has no submitted reviews yet and remains Draft.
