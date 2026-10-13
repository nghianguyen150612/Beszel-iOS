# P004 — upstream Beszel v0.20.0 integration

P004 source integration is complete; its integration gate is **PASS**.
Gate B remains pending P005 and P006. This is not a release-ready iOS port.

## Verified foundation and Git history

| Reference | SHA |
| --- | --- |
| Starting `iOS` and new `ios/upstream-v021-integration` | `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` |
| Frozen `main` / roadmap target | `bb23f99b3eeaced9608e11e280e7c986dba650be` |
| Historical fork point | `f204dc17e64bba444478c60210d018e43a777193` |
| Verified `v0.20.0^{commit}` / pinned merge target | `a0dc19eacf6902d2f6934a3efd7ea7277198c40b` |
| Upstream merge commit | `65731f9a86f19b1668d260e49f6d5ee7318453be` |

The cloud checkout was explicitly cloned with `--branch iOS --single-branch`.
Both product refs matched the requested P003 baseline. The integration branch
was absent and was created and pushed from verified `origin/iOS`. The P004
working branch is `ios/p004-upstream-v020-merge`; its PR base is
`ios/upstream-v021-integration`. No product branch was reset or updated.

Executed:

```sh
git merge --no-ff --no-commit a0dc19eacf6902d2f6934a3efd7ea7277198c40b
git commit -m 'merge(ios): integrate upstream Beszel v0.20.0 (P004)'
```

The normal merge has first parent `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`
and real second parent `a0dc19eacf6902d2f6934a3efd7ea7277198c40b`.
All **25/25** full SHAs match the immutable P002 change matrix's P004 rows
**2–26** (header is row 1). Each SHA, canonical row, priority and source path
is also recorded in [the preservation record](upstream-v021-p004-preservation.json).
Functional owners come from P003's implementation crosswalk, not P002's
provisional prompt labels. P005 remains **116/116 pending** and P006
**37/37 pending**: none of their canonical SHAs is an ancestor of this result.
No v0.21-only or moving `upstream/main` commit was merged.

## Changed files and actual conflict review

The source merge changes **161 files**, 17,672 insertions and 1,156 deletions.
The exhaustive file list is `merge_changed_files` in the JSON record. It includes
Agent monitor tasks/probes/history/scheduling, CBOR models, Hub sync/storage,
alerts, collection snapshot, rollups, frontend monitor UI/locales, pinned Go
dependencies, Docker variant tags and the v0.20 source/package version.

**Actual conflicts: none.** `git diff --name-only --diff-filter=U` and
`git diff --cc` were empty. `git show --remerge-diff` shows no resolution hunks.
No manual Agent/Hub conflict replacement or preference strategy was used.

Every merge-changed path equals the pinned target's tree, except
`.github/workflows/docker-images.yml`, where the automatic merge combines the
upstream `latest=false` variant behavior with the existing `!v*-ios.*` exclusion.
Shared backend, security, models, records and migration paths equal upstream
exactly. `agent/system.go` did not change in this range; its iOS CPU bypass and
`adjustPlatformSystemDetails()` hook remain intact.

A separate validation repair makes the preexisting systemd CPU test fixture
scale with `runtime.NumCPU()` instead of assuming eight cores. The cloud runner
has three CPUs; production calculations and assertions remain unchanged. This
is test-only work, not a merge conflict or a new upstream history batch.

## Thirteen logical iOS patches

The pre-merge inventory was captured before the merge. JSON records current
files, before/after SHA-256 hashes, owners and historical absent paths. The old
Agent/Hub probe workflow files were already consolidated into `ios-build.yml`
at P003; their absence is not a P004 deletion. Lowercase `readme.md` was likewise
already replaced by the product's EN/VI README files.

| Logical patch | Canonical owner | P004 disposition |
| --- | --- | --- |
| `IOS-A7-RUNTIME` | P007 | PRESERVED; existing implementation unchanged |
| `IOS-AGENT-BATTERY` | P008 | PRESERVED; existing implementation unchanged |
| `IOS-AGENT-BUILD-PROBE` | P007 | PRESERVED; existing implementation unchanged |
| `IOS-AGENT-SYSTEM-HOOK` | P008 | PRESERVED; existing implementation unchanged |
| `IOS-FRONTEND-BUILD` | P015 | PRESERVED; existing implementation unchanged |
| `IOS-HUB-BUILD-PIPELINE` | P015 | PRESERVED; existing implementation unchanged |
| `IOS-INSTALLER-ARCH-DETECTION` | P016 | PRESERVED; existing implementation unchanged |
| `IOS-INSTALLER-BOOTSTRAP-INTEGRITY` | P016 | PRESERVED; existing implementation unchanged |
| `IOS-INSTALLER-LIFECYCLE` | P016 | PRESERVED; existing implementation unchanged |
| `IOS-INSTALLER-VALIDATION` | P017 | PRESERVED; existing implementation unchanged |
| `IOS-PORT-DOCUMENTATION` | P017 | PRESERVED; existing implementation unchanged |
| `IOS-RELEASE-ISOLATION` | P016 | PRESERVED; Docker variant fix + iOS tag exclusion retained |
| `IOS-UPSTREAM-SYNC-CI` | P017 | PRESERVED; existing implementation unchanged |

The guarded A7 `runtime.procyieldAsm` patch is unchanged. Known Go 1.27.1 source
patches to YIELD; missing/duplicate start markers, unknown instruction shape and
missing end marker fail without writing. Native build settings remain
`GOOS=ios GOARCH=arm64 CGO_ENABLED=1`, iPhoneOS clang SDK wrapper and iOS 12.0.
Battery still uses AppleARMPMUCharger / `ioreg -l` with Darwin `!ios` exclusion;
`go list` selects `battery.go` + `battery_ios.go` and the iOS system platform hook.

Installer engine and bootstrap are byte-identical. Installer version stays
**1.0.0**, state format **1**, engine pin
`d4b4e381a76c75ac79cb7c99558853b62c7c2163879a72c0333acc5c40092d0c`.
Independent Agent/Hub LaunchDaemons, existing CLI, SHA-256 validation, ldid
signing, release asset names, binary backups and ordinary data preservation
remain unchanged. Existing binary rollback is not proof of migrated DB rollback;
P003's complete-data restore requirement still blocks production upgrades.
Generic release and Docker iOS tag exclusions are preserved. No release tag,
release publication, installer update or production deployment was performed.

## Security and migration review

The P004 P0 rows 6, 9, 14, 19 and 21 retain their intended changes:

- `50f6fc…`: first-user checks, user/admin creation and temporary-admin deletion
  occur atomically inside one transaction; concurrent/failed bootstrap tests remain.
- `4bf707…`: trusted-header identity uses the actual peer `RemoteAddr` and
  configured IP/CIDR allowlist. Unset/empty allowlist preserves upstream's opt-in
  trusted-header behavior; a configured invalid-only list trusts nobody.
- `f7fd3e…`: OAuth-created users start with `user`; existing roles remain intact.
- `b0bc72…`: final v0.20 collection snapshot replaces the 0.19/development
  snapshot. Upstream intentionally folds and removes
  `1788955200_pool_updates.go` and `1789079746_container_update_available.go`.
  No migration name, identifier, content or registration order was changed locally.
- `187dc8…`: SSH reconnect and failed monitor sync converge, including empty
  configurations, without taking successful stats collection down.

Network monitor/stat collection rules remain authenticated and system-member
scoped; readonly users cannot write. Explicit `SHARE_ALL_SYSTEMS=true` semantics
remain upstream's configuration contract. Alert CRUD derives the current user
from authentication and checks system membership. No authentication or TLS
verification was disabled. HTTP(S), TCP, DNS, ICMP, retries, deadlines,
cancellation, sync and rollups retain shared upstream implementations.

The existing authenticated systemd detail response still has
`Cache-Control: public, max-age=60`; its later audited hardening remains
P005/P006 history and P013/P017 Gate D work. No existing protection was removed,
and no later commit was cherry-picked early to disguise this obligation.

All database tests use PocketBase's disposable synthetic test applications.
Neither `/var/lib/beszel-hub` nor a physical iPad DB was opened. Unit/integration
collection, ownership, bootstrap and rollup checks do not demonstrate the full
0.19 upgrade/backup/restore/interruption plan. That harness is not yet implemented;
DB-BACKUP-01, DB-MIG-01, DB-RESTORE-01 and DB-INTERRUPT-01 remain **DEFERRED**, owned
by P013/P017 at Gate D (and installer reconciliation at Gate E). Never start the
old Hub against migrated state or delete data to make a migration pass.

## Executed validation and captured evidence

Toolchains installed in cloud scratch: **Go 1.27.1**, **Bun 1.4.0**. Dependencies
remain at the merged target and the existing iOS frozen lock; none was upgraded
to suppress warnings. Both baseline and merged frontend installations used
`bun install --frozen-lockfile`, then genuine `bun run build`; generated `dist`
contains real assets and is embedded by the Hub. No fake dist or embed bypass.
Vite's large-chunk warning is informational; builds succeeded.

| Check | P003 baseline | P004 result |
| --- | --- | --- |
| `python3 scripts/audit/validate-upstream-v021-matrix.py` | PASS | PASS; 178 frozen rows, no unresolved audit rows |
| `python3 scripts/audit/validate-upstream-v021-readiness.py` | PASS | PASS; 13 logical patches, 21 P0 risks |
| `python3 -m unittest discover -s tests/audit -p 'test_*.py' -v` | PASS; 14 | PASS; 14 |
| `sh tests/upstream-sync-test.sh` | PASS; 10 | PASS; 10 |
| `sh tests/ios-patch-sentinels.sh` | PASS; 44 | PASS; 44 |
| `sh tests/bootstrap-test.sh` | PASS; 54 | PASS; 54 |
| `sh tests/install-sh-test.sh` | PASS; 814 | PASS; 814 |
| `git diff --check` | PASS | PASS |
| Frozen Bun install and frontend build | PASS | PASS |
| `bun test` | No tests existed; exit 1 is not a pass | PASS; 4 monitor tests |
| Canonical `env -u NETWORK go test -tags 'testing no_ui' ./...` | CPU fixture failure on 3 CPUs | PASS; all packages after fixture repair |
| Linux Agent/Hub `go build` after real frontend generation | Not required for historical binaries | PASS; Agent and Hub |
| A7 runtime source and negative-shape fixtures | Existing patch reviewed | PASS; five cases |
| Native iOS Agent/Hub, frontend, Mach-O, checksums | Historical results not reused | PASS; Agent, Hub, frontend, Mach-O arm64 / iOS 12.0, SHA256SUMS |

Initial `go test ./...` failed on both snapshots because tests reference helpers
behind `testing`; `go test -tags testing ./...` additionally duplicated PocketBase
UI routes. These are not successful invocations. The authoritative Makefile
uses `testing no_ui`. The inherited cloud `NETWORK` value also polluted CLI tests;
it is removed for the test process only, preserving proxy, CA and credentials.
The host-size CPU fixture failure was investigated and repaired as described above.
Final canonical suite evidence includes first-user/OAuth/trusted-proxy auth,
collection permissions, alerts, network monitor task/probe/sync and history tests.

Hosted native source validation:
[iOS Build run 38390065511](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38390065511),
commit `65731f9a86f19b1668d260e49f6d5ee7318453be`.
[Installer lifecycle run 38390070057](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38390070057)
uses the same commit. These were explicit branch workflow dispatches. The tag-only
`release-ios` job is expected to be skipped and is never claimed as passed.
The later fixture/documentation commit does not alter build inputs; its diff is
reviewed separately. JSON records exact results and evidence hashes.

## Deferred compatibility and remaining gates

| Obligation | Result / reason | Owner | Blocking gate |
| --- | --- | --- | --- |
| P005/P006 history | 116 + 37 commits absent by ancestry; deliberately outside P004 | P005/P006 | B |
| ICMP on jailbroken iOS | Native socket permissions untested; shared exec fallback explicitly rejects `GOOS=ios`; failures become probe loss, not fabricated success | P010/P017; device P018 | C/F |
| A7 runtime execution | Hosted Mach-O build cannot prove absence of SIGILL on Apple A7 | P007/P018 | C/F |
| Metrics, transport, legacy pairs, unsupported collectors | Source hooks preserved; exact P003 version-pair/device matrix remains unexecuted | P008/P009/P010/P017 | C |
| Full DB upgrade, backup, restore, interruption | No completed seeded 0.19 migration/restore harness; unit tests are insufficient | P013/P017 | D/E |
| Auth-sensitive caching and final security matrix | Frozen later hardening commits still pending; complete P003 negative matrix required | P013/P017 | D |
| Browser routes and remote Linux/Windows/macOS telemetry | Built assets and local tests do not replace browser/remote-OS fixtures | P014/P015/P017 | D |
| Package updates / Wi-Fi RSSI | Later adapters/fallback validation required; no native capability claim | P011/P012 | C |
| Installer migrated-data rollback and release candidate | Existing installer regression passes; DB restore and candidate evidence remain | P016/P017 | E |
| Physical iPad mini 2 / A7 / iOS 12.5.7 | NOT EXECUTED; historical 0.19 device results remain historical | P018 | F |

P004's integration decision is **PASS**: canonical tests and applicable hosted
checks passed. PR merge remains conditional on diff review and GitHub requirements. Gates B–F remain incomplete.
No production deployment is authorized. P005 is the next milestone and was not started.
