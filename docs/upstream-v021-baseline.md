# Beszel upstream v0.21.0 baseline (P001)

This document freezes the Beszel-iOS comparison baseline for the P001 milestone.
It records the source state and compatibility contracts for later review; it is
not an upstream integration or a classification of all pending commits.

## Audit snapshot

- Audit date: **2026-10-13**, timezone **Asia/Ho_Chi_Minh**.
- Product repository: <https://github.com/nghianguyen150612/Beszel-iOS>.
- Canonical upstream: <https://github.com/henrygd/beszel>.
- Product branch: `iOS` (case-sensitive).
- Frozen references:

  | Reference | Full SHA | Source version |
  | --- | --- | --- |
  | `origin/iOS` | `a6b73365147fc8c3a820552a145d370418e5b680` | `0.19.0` |
  | `origin/main` | `bb23f99b3eeaced9608e11e280e7c986dba650be` | `0.21.0` |
  | `upstream/main` | `bb23f99b3eeaced9608e11e280e7c986dba650be` | `0.21.0` |
  | Merge-base of `origin/iOS` and `origin/main` | `f204dc17e64bba444478c60210d018e43a777193` | — |

  The source versions above were read from `beszel.go` at each ref. The
  canonical `upstream/main` ref matched the frozen `origin/main` snapshot when
  fetched; upstream had not advanced beyond this target.

## Reproduction commands and commit counts

The cloud checkout was explicitly cloned from the case-sensitive product
branch, then the product and canonical refs were fetched separately:

```sh
git clone --branch iOS --single-branch \
  https://github.com/nghianguyen150612/Beszel-iOS.git Beszel-iOS
cd Beszel-iOS
git fetch origin refs/heads/main:refs/remotes/origin/main
git remote add upstream https://github.com/henrygd/beszel.git
git fetch upstream refs/heads/main:refs/remotes/upstream/main
git branch --show-current
git status --short
git rev-parse HEAD
git rev-parse origin/iOS
git rev-parse origin/main
```

The history and source declarations were checked with:

```sh
git merge-base origin/iOS origin/main
git rev-list --left-right --count origin/iOS...origin/main
git log -1 --format=fuller origin/iOS
git log -1 --format=fuller origin/main
git diff --stat origin/iOS...origin/main
git status --porcelain
git show origin/iOS:beszel.go
git show origin/main:beszel.go
git show upstream/main:beszel.go
git rev-parse upstream/main
git merge-base --is-ancestor origin/main upstream/main
```

The verified merge-base is `f204dc17e64bba444478c60210d018e43a777193`.
`git rev-list --left-right --count origin/iOS...origin/main` returned
`35 178`: 35 commits are unique to the iOS branch and **178 upstream commits
are absent from iOS**. The comparison diff contains 333 files (42,863
insertions and 4,614 deletions). No claim is made here that those 178 commits
have been individually audited; that review and its change-classification
matrix belong to P002.

The **118-commit** figure is a separate `main`-branch synchronization delta,
not the iOS integration delta. Its September point is the `origin/main`
ancestor `d80a2f49f9d00abe4c7853f0b041dcb65846814b` (2026-09-24); the October
snapshot is `bb23f99b3eeaced9608e11e280e7c986dba650be`:

```sh
git rev-list --left-right --count \
  d80a2f49f9d00abe4c7853f0b041dcb65846814b...origin/main
```

This returned `0 118`. The earlier September audit preserved in
[`upstream-sync.md`](upstream-sync.md) remains historical evidence and is not
rewritten as the October snapshot. The `118` main synchronization count and
the `178` upstream commits absent from iOS describe different branch ranges.

## Version baseline

- Current Beszel source version on `origin/iOS`: **`0.19.0`**.
- Frozen upstream target source version on `origin/main` and `upstream/main`:
  **`0.21.0`**, including commits through `bb23f99b3eeaced9608e11e280e7c986dba650be`.
- The version declarations were read directly using `git show
  origin/iOS:beszel.go`, `git show origin/main:beszel.go`, and
  `git show upstream/main:beszel.go`.
- P001 does not move the target if upstream advances later. It creates no
  release tag, binary revision, or deployment.

## Compatibility contracts frozen for integration work

These are required invariants for future port work. Their presence in source or
tests does not imply that every item was retested during P001.

| Area | Mandatory invariant | Authoritative references |
| --- | --- | --- |
| Runtime and platform | `GOOS=ios`, `GOARCH=arm64`, `CGO_ENABLED=1`; iOS 12.0 minimum deployment target; Apple A7 support; patch Go `runtime.procyieldAsm` for A7; refuse to patch an unknown runtime source shape | `.github/workflows/ios-build.yml`; `.github/scripts/patch-go-ios-arm64-runtime.py`; `tests/ios-patch-sentinels.sh` |
| Agent | Native battery telemetry through `AppleARMPMUCharger`; iOS system metadata from `sysctl` and `SystemVersion.plist`; retain shared Beszel protocol and models; preserve existing Agent functionality | `agent/battery/battery_ios.go`; `agent/battery/battery_darwin.go`; `agent/system.go`; `agent/system_platform_ios.go`; `agent/system_platform_other.go`; `internal/common/`; `docs/architecture.md` |
| Hub | Standard upstream-compatible Hub, including PocketBase behavior and migrations; persistent data path `/var/lib/beszel-hub`; no destructive database operations during port or installer work | `internal/cmd/hub/`; `internal/hub/`; `internal/migrations/`; `scripts/ios/install-beszel.sh`; `docs/architecture.md` |
| Installer and services | Independent Agent and Hub LaunchDaemons (`dev.beszel.agent` and `dev.beszel.hub`); preserve the existing `beszel-ios` CLI contract; SHA-256 verification; `ldid` signing; backups and rollback; release assets `beszel-agent-ios-arm64`, `beszel-hub-ios-arm64`, and `SHA256SUMS` | `install.sh`; `scripts/ios/install-beszel.sh`; `.github/workflows/ios-build.yml`; `tests/install-sh-test.sh`; `tests/bootstrap-test.sh`; `docs/architecture.md` |
| Hardware reference | iPad mini 2, model A1489 / `iPad4,4`, Apple A7, iOS 12.5.7, Amethyst / Procursus jailbreak environment | `docs/device-validation.md`; `docs/ios-port-status.md` |

The Hub data directory contains persistent accounts, configuration, and history.
Normal installer updates and uninstall preserve it. Data removal is a separate
explicit purge flow and is outside ordinary maintenance.

## Implementation and validation status

### Implemented and present at the frozen iOS ref

- The iOS build workflow builds Agent and Hub for iOS arm64 with CGO, targets
  iOS 12.0, applies the guarded A7 runtime patch, verifies Mach-O metadata, and
  prepares the two named binaries plus `SHA256SUMS`.
- The Agent battery and system metadata platform files are present. The Hub,
  protocol, models, and migration code remain shared Beszel code; this baseline
  contains no replacement Agent or Hub implementation.
- The installer has separate Agent and Hub services, checksum-verified release
  downloads, `ldid` signing, staged updates, binary backups, rollback, and
  Hub-data preservation.

### Historical validation, not run again during P001

[`device-validation.md`](device-validation.md) records the prior physical-device
matrix dated 2026-09-17/18: iPad mini 2 / A1489 / Apple A7 / iOS 12.5.7 in an
Amethyst / Procursus semi-untethered jailbreak. It records battery and system
metadata, Agent and Hub operation, installer lifecycle behavior, database and
history preservation, and reboot persistence after manual jailbreak
reactivation. Those claims remain historical evidence for the tested 0.19.0
binary and prior installer version. P001 performed no physical-device test and
does not claim that 0.21.0 has been validated on hardware.

### P001 local baseline tests

The first four commands were run before documentation edits on the clean
`origin/iOS` checkout and repeated after the P001 edits. Both runs were local in
the cloud workspace, not GitHub Actions or physical hardware.

| Exact command | Result | Evidence |
| --- | --- | --- |
| `sh tests/upstream-sync-test.sh` | Pass | 10 passed, 0 failed |
| `sh tests/ios-patch-sentinels.sh` | Pass | 44 passed, 0 failed |
| `sh tests/bootstrap-test.sh` | Pass | 54 passed, 0 failed |
| `sh tests/install-sh-test.sh` | Pass | 814 passed, 0 failed |
| `git diff --check` | Pass after edits | No whitespace errors |

The complete Go test suite was blocked in this workspace. The checkout has no
generated `internal/site/dist` directory, which `internal/site/embed.go`
requires through `//go:embed all:dist`; Bun is unavailable, and the available
`/usr/bin/go` command is not a Go toolchain (`go test ./...` returned
`Go: Unknown option: test`). No empty frontend directory was created. The iOS
CGO/Mach-O build also requires the macOS iPhoneOS SDK and was not run locally.
The iOS build workflow's Go build is not triggered by these documentation-only
changes; the separate installer workflow matches the changed maintenance-doc
paths and is checked on the PR. A skipped or absent iOS build is not counted as
a successful build.

### Untested or blocked items

- The 0.21.0 source has not been built for iOS or exercised on the reference
  device in P001.
- The new iOS behavior of the full upstream delta remains unaudited until P002.
- No P001 CI result or physical-device result is claimed in this baseline.
- Other iOS devices, SoCs, iOS versions, and jailbreak environments remain
  untested as recorded in the port-status and device-validation documents.

## Scope frozen for P002 and later work

P001 freezes the refs, version declarations, compatibility contracts, and test
evidence only. Upstream integration has **not** begun: no Beszel 0.20.0 or
0.21.0 code was merged, and no production code, dependency, generated asset,
installer behavior, runtime patch, database schema, or release logic changed.

P002 is responsible for auditing all 178 upstream-only commits and producing a
change-classification matrix with source evidence and iOS-specific review
requirements. P001 does not pre-classify that range. Later integration work is
owned by the roadmap's P004–P006 milestones and must preserve the contracts
above. The target remains the frozen `bb23f99b3eeaced9608e11e280e7c986dba650be`
snapshot unless a later milestone explicitly records a new target.
