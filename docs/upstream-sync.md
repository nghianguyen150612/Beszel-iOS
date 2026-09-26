# Upstream Sync

This is the maintenance contract for keeping the iOS port close to canonical
Beszel without silently dropping an iOS compatibility patch.

## Branch roles

- `upstream/main` — canonical Beszel source and behavior. It is an audit source,
  not the product branch.
- `main` — preserved upstream-aligned branch. iOS maintenance work does not
  advance or merge it.
- `iOS` — the Beszel-iOS development/default branch: upstream history plus
  the iOS-specific deltas listed below.
- `origin` — `https://github.com/nghianguyen150612/Beszel-iOS.git`;
  `upstream` — `https://github.com/henrygd/beszel.git`.

Never turn `upstream/main` into the product branch, and never update `ios` with
an unattended merge or rebase.

## Audit baseline

The maintenance audit started from the validated commit `8fed846c`, with
`HEAD == origin/iOS` and `beszel.Version == "0.19.0"`.

At the fetched audit snapshot:

- Upstream default branch: `main` (`upstream/HEAD -> upstream/main`).
- Upstream `main`: `4bf70700` (`feat(hub): add TRUSTED_PROXY_IPS allowlist for TRUSTED_AUTH_HEADER`).
- Newest upstream stable release/tag: `v0.19.0`, tag commit
  `ffcdb041`, dated 2026-09-03.
- `ios` fork point from upstream `main`: `f204dc17`
  (`feat(agent): Add docker image update available flag (#2211)`).
- `ios` includes 20 iOS-port commits after that fork point.
- `origin/main` was `6a7b2772`, six commits behind the fetched untagged
  `upstream/main`; it remains preserved and untouched.
- The Beszel-iOS base is therefore still `0.19.0`, and the stable-tag drift
  status is **IN SYNC**.
- The eight untagged upstream commits after `f204dc17` are pending review; they
  are not silently included in the validated iOS binary.
- Validated binary release: `v0.19.0-ios.1`.
- Installer version: `1.0.0`.
- Validated hardware: iPad mini 2 / `iPad4,4` / `A1489`, Apple A7, arm64,
  iOS 12.5.7.

The fork point matters: comparing only with the `v0.19.0` tag would mix
upstream's already-integrated post-release commits with the actual iOS delta.

## iOS-specific patch inventory

The inventory below was derived from the actual content of the 20 commits in
`git log upstream/main..ios` and `git diff f204dc17..ios`, not from commit
titles alone. It answers: “If upstream changes X, which iOS behavior must be
reviewed?”

| # | Area | iOS-specific files or behavior | Upstream change that can invalidate it |
| --- | --- | --- | --- |
| 1 | iOS build workflow | `.github/workflows/ios-build.yml`: macOS runner, Go setup, runtime patch, Bun/frontend build, clang wrapper, Agent + Hub builds, Mach-O/deployment validation, checksums, artifact, and tag-gated release job | Go module/toolchain layout, frontend build layout, command entrypoints, or workflow trigger conventions |
| 2 | `GOOS=ios` / `GOARCH=arm64` | Both build commands set `CGO_ENABLED=1`, `GOOS=ios`, and `GOARCH=arm64` | Go dropping or changing the iOS target, or a build package becoming unsupported on iOS |
| 3 | CGO iPhoneOS clang wrapper | The workflow resolves the iPhoneOS SDK with `xcrun` and passes `-arch arm64`, `-isysroot`, and `-mios-version-min=12.0` | Xcode/SDK/compiler changes; an upstream CGO dependency or build flag change |
| 4 | iOS deployment target | The wrapper targets iOS 12.0; `otool` checks the actual `LC_BUILD_VERSION` `minos` or legacy `LC_VERSION_MIN_IPHONEOS` `version` field | Build/linker changes that omit or raise the minimum deployment metadata |
| 5 | Apple A7 Go runtime workaround | `.github/scripts/patch-go-ios-arm64-runtime.py` replaces the known `CNTVCT_EL0` `runtime.procyieldAsm` body with a `YIELD` loop and refuses unknown source shapes | Go runtime `src/runtime/asm_arm64.s`; any Go version change requires source-shape review |
| 6 | iOS battery telemetry | `agent/battery/battery_ios.go` (`//go:build ios`) invokes `/usr/sbin/ioreg -r -c AppleARMPMUCharger -l`, handles legacy output, maps the system battery to `Primary`, and parses capacity/charge fields | Battery package API changes to `Battery`, state constants, normalization, or `Primary`; an upstream battery file split/build-tag change |
| 7 | Darwin battery exclusion | `agent/battery/battery_darwin.go` is gated `darwin && !ios`, so iOS does not select the macOS `AppleSmartBattery -a` implementation | Upstream changes to the Darwin battery build tag or platform file selection |
| 8 | Darwin disk handling / disk-I/O fallback | No iOS-specific disk fork: `agent/disk.go`, storage-pool code, and gopsutil disk counters remain shared upstream code. The port has no claimed Darwin-specific fallback | gopsutil disk APIs, shared `agent/storage_pool.go` / `agent/zfs/*`, or platform build tags; retest rather than assume iOS mount/I/O behavior |
| 9 | iOS network handling | No iOS-specific network fork; `agent/network.go` and the upstream network collectors are used as-is | gopsutil network APIs or interface/address assumptions; VPN and per-interface behavior is not broadly validated |
| 10 | Agent support / system metadata | `agent/system.go` skips the Darwin CPU probe on iOS and calls `adjustPlatformSystemDetails()`; `agent/system_platform_ios.go` supplies sysctl/plist metadata; `agent/system_platform_other.go` is the non-iOS no-op | `refreshSystemDetails`, `system.Details`, `system.Darwin`, or the gopsutil CPU/platform APIs |
| 11 | Hub support | No Hub source fork; `.github/workflows/ios-build.yml` builds `./internal/cmd/hub` for iOS with the standard Hub code and migrations | `internal/cmd/hub`, PocketBase APIs, Hub initialization, migrations, or a dependency that stops compiling for iOS |
| 12 | Frontend embed/build requirement | `internal/site/embed.go` embeds `all:dist`; `internal/site/dist` is ignored; the workflow runs `bun install --frozen-lockfile` (pinned Bun 1.4.0) and `bun run build` before either Go build | Embed path, frontend package/build output, Bun/Vite, or generated frontend API/types |
| 13 | GitHub release workflow | The iOS workflow uploads the three-file payload and only its tag job publishes; it validates source version, iOS history, checksums, asset count, and Latest/non-draft/non-prerelease state | `beszel.go` version declaration, artifact layout, GitHub action behavior, or release permissions |
| 14 | iOS release-tag format | `v<upstream>-ios.<positive-integer>`, currently `v0.19.0-ios.1`; enforced by `install.sh` and the iOS release job | Upstream version declaration or any change to the release asset/tag contract |
| 15 | `install.sh` + `scripts/ios/install-beszel.sh` | Two-layer installer: `install.sh` (v1.0.0) is a pipe-safe bootstrap that downloads the engine over HTTPS, verifies `engine_sha`, and executes the verified copy; the engine provides Installer 1.0.0 behavior: fresh Agent/Hub/Both install, checksum-pinned downloads, `ldid`, LaunchDaemon setup, health checks, and offline lifecycle actions | Release asset names, Latest-release URL shape, upstream version/tag policy, or any engine edit that is not re-pinned in `install.sh` |
| 16 | Installer tests | `tests/install-sh-test.sh` (814 baseline tests: parsing, checksum/state handling, transactions, rollback, diagnostics, repair, reconfiguration, uninstall, data retention, failure paths, bootstrap provenance) plus `tests/bootstrap-test.sh` (54 bootstrap contract/pin tests) | Any installer function contract, asset/state name, or bootstrap pin/provenance rule |
| 17 | LaunchDaemon handling | Plists are generated on-device by the engine (`scripts/ios/install-beszel.sh`, persisted as `/var/lib/beszel-ios/manager.sh`) as `dev.beszel.agent` and `dev.beszel.hub`; no repository plist is authoritative | Installer path/label changes or iOS launchd behavior |
| 18 | `ldid` signing | Installer signs staged binaries with `ldid -S`, installs under `/usr/local/bin`, and enforces `root:wheel` / mode 755 | Device signing tool or jailbreak environment changes; no upstream source dependency |
| 19 | State tracking | Installer-owned `/var/lib/beszel-ios/install-state` records per-component release tags and original asset SHA-256 values without storing keys | Installer state format or transactional update logic |
| 20 | Update / rollback | Signed staging, `/usr/local/bin/*.bak` binary backups, automatic rollback, plist preservation, and no Hub data-directory mutation | Release assets, startup/health behavior, or installer transaction assumptions |
| 21 | Repair / reconfigure / uninstall | Read-only diagnostics; conservative restore/restart; validated Agent key/port and Hub port transactions; application uninstall preserves data by default; purge requires exact typed confirmation | Installer paths, launchd semantics, service health, or data-retention expectations |
| 22 | Real-device documentation | `docs/device-validation.md`, `docs/ios-port-status.md`, `docs/ios-build-notes.md`, and `docs/architecture.md` record the reference device, limitations, and operational contract | A validation claim, supported metric, device layout, or build prerequisite changes |
| 23 | iOS release isolation | `release.yml` and `docker-images.yml` exclude `v*-ios.*`, preventing GoReleaser/Docker automation from handling native iOS tags | Upstream edits to tag triggers or release workflow ownership |

Rows explicitly marked “no iOS-specific fork” are deliberate negative deltas:
shared upstream behavior still needs functional review after an upstream change,
but there is no iOS patch to reapply.

## Upstream change surfaces

The following table covers the requested upstream surfaces against the pending
range `f204dc17..4bf70700`. Each surface has one classification and a concrete
reason. A binary-producing integration still requires the complete real-device
parity checklist below.

| Surface | Evidence in pending range | Classification | Technical reason |
| --- | --- | --- | --- |
| `internal/cmd/agent` | Not touched | UNAFFECTED | The Agent entrypoint, CLI flags, health subcommand, and output path are unchanged in this range. |
| `internal/cmd/hub` / Hub entrypoint | Not touched | UNAFFECTED | Hub startup, CLI flags, `/api/health`, and migration registration are unchanged. |
| `internal/agent` and system collectors | `agent/smart.go` / tests and shared storage-pool code touched | AUTO-MERGE LIKELY | No iOS fork exists in SMART or shared collector logic; build and functional parity still need review. |
| Disk collectors / disk-I/O | `agent/storage_pool.go`, `agent/zfs/*` touched by `98210174` | IOS PATCH REVIEW REQUIRED | The new `zfs_nonlinux.go` has a `!linux` tag selected by `GOOS=ios`; verify it coexists with the iOS build and does not change unavailable-ZFS behavior. |
| Battery collectors | Not touched | UNAFFECTED | `battery_ios.go` and the Darwin exclusion are outside the pending range. |
| Network collectors | Not touched | UNAFFECTED | No network collector or interface handling changed. |
| Embedded frontend | Seven frontend files in `f0f1f798`; two in `6a7b2772`; one in `086091a0` | AUTO-MERGE LIKELY | The iOS Hub embeds the same generated frontend; build output and dashboard behavior must be rechecked. |
| Frontend build output/toolchain | No embed path or package-manager change | AUTO-MERGE LIKELY | Existing `bun install --frozen-lockfile` (pinned Bun 1.4.0) + `bun run build` remains the supported prerequisite, but generated output must be rebuilt. |
| Go version / `go.mod` / `go.sum` | Not touched | UNAFFECTED | The range keeps Go 1.27.1 and the dependency graph unchanged. |
| Build tags / platform assumptions | `zfs_nonlinux.go` added; no runtime or iOS platform file touched | IOS PATCH REVIEW REQUIRED | `!linux` includes iOS, so `GOOS=ios go list/build` is an explicit check even without a textual conflict. |
| Go runtime / A7 assumption | Not touched | UNAFFECTED | The runtime patch is applied to the installed Go toolchain, independent of these upstream commits; its source-shape sentinel remains mandatory. |
| Release asset names | Not touched | UNAFFECTED | `beszel-agent-ios-arm64`, `beszel-hub-ios-arm64`, and `SHA256SUMS` remain the port contract. |
| Database migrations | Not touched | UNAFFECTED | No migration file or collection snapshot changed in the range. |
| Config format | No config schema change | UNAFFECTED | The Hub/Agent configuration and installer plist contracts are unchanged. |
| Agent↔Hub protocol | No `internal/common`, WebSocket action, or minimum-version change | UNAFFECTED | CBOR/WebSocket compatibility thresholds and payload semantics are unchanged. |
| Ports / CLI arguments | Not touched | UNAFFECTED | Agent 45876, Hub 8090 defaults, flags, and health command remain unchanged. |
| Health endpoint | Not touched | UNAFFECTED | `/api/health` behavior is not modified. |
| Version/release handling | No `beszel.go` or release workflow change | UNAFFECTED | The code base remains `0.19.0`; no release tag or revision is created by this audit. |
| Hub authentication/users | `internal/hub/api.go`, `internal/users/users.go`, and tests touched | AUTO-MERGE LIKELY | Changes are upstream Hub behavior with no iOS-owned source conflict; login/account and trusted-header behavior need regression testing. |

### Pending commit classification

The fetched untagged upstream commits, oldest to newest, are:

| Commit | Change | Classification | Reason |
| --- | --- | --- | --- |
| `086091a0` | Discard pending history when switching to live charts | AUTO-MERGE LIKELY | Frontend-only behavior; it is consumed by the embedded UI and has no iOS patch overlap. |
| `6a7b2772` | Follow system theme preference live | AUTO-MERGE LIKELY | Frontend-only behavior; rebuild the embedded UI and check login/dashboard rendering. |
| `98210174` | Skip ZFS utility calls when `/dev/zfs` is unavailable | IOS PATCH REVIEW REQUIRED | Adds a `!linux` file selected by iOS and changes shared storage-pool behavior; run the iOS package/build checks. |
| `a0bf3387` | Set Hub batch request/body limits | AUTO-MERGE LIKELY | Hub default settings only; no iOS source conflict or protocol change, but exercise Hub startup and alert-related behavior. |
| `50f6fc07` | Atomic first-user bootstrap and tests | AUTO-MERGE LIKELY | Hub/users behavior only; no migration or iOS source conflict, but repeat login/bootstrap checks. |
| `f0f1f798` | Persist view preferences and language | AUTO-MERGE LIKELY | Embedded frontend plus user settings behavior; rebuild and test dashboard/settings persistence. |
| `18f7a4bb` | Revert SMART warnings for attributes 5/197/198 | AUTO-MERGE LIKELY | Shared SMART parser deletion with no iOS fork; behavior becomes upstream-canonical. |
| `4bf70700` | Restrict trusted auth headers by proxy IP/CIDR | AUTO-MERGE LIKELY | New optional Hub authentication setting with unchanged default behavior; repeat auth and direct/proxy access tests. |

No pending commit is an `IOS PATCH CONFLICT`. The two `IOS PATCH REVIEW
REQUIRED` classifications are build-tag/platform reviews, not claims that a
textual merge conflict is expected. Any release containing these binary or
frontend changes is `REAL-DEVICE RETEST REQUIRED`.

## Reproducible upstream-sync procedure

Run this procedure on `ios` with a clean tree. It is intentionally review-first
and does not update `main`.

1. Fetch separately and inspect refs:

   ```sh
   git fetch origin
   git fetch upstream --tags
   git branch --show-current
   git status --short
   git rev-parse HEAD
   git rev-parse origin/iOS
   git symbolic-ref --short refs/remotes/upstream/HEAD
   ```

2. Identify the target upstream stable tag. Prefer the newest `vX.Y.Z` tag
   over an untagged `main` tip. Record the old `beszel.Version`, current iOS
   commit, fork point, target tag/commit, and tag date.

3. Read the upstream release notes/changelog and inspect the complete range:

   ```sh
   git log --stat <old-upstream-ref>..<target-ref>
   git diff --name-status <old-upstream-ref>..<target-ref>
   ```

   Classify every touched surface as `UNAFFECTED`, `AUTO-MERGE LIKELY`,
   `IOS PATCH REVIEW REQUIRED`, `IOS PATCH CONFLICT`, or
   `REAL-DEVICE RETEST REQUIRED`, with a technical reason.

4. Integrate only after that review. On `ios`, use a normal reviewed merge
   that preserves history, for example:

   ```sh
   git merge --no-ff --no-commit <target-ref>
   ```

   Resolve conflicts deliberately, retain the iOS behavior, inspect the
   staged result, and commit. Do not rebase, rewrite history, force-push, or
   merge upstream into `main`. If the target is only an untagged development
   tip, do not represent it as a stable release.

5. Re-run the patch inventory and inspect all changed iOS-sensitive files.
   The runtime patch must match the installed Go runtime source shape; never
   weaken or bypass a failed sentinel.

6. Run validation in this order:

   ```sh
   sh tests/ios-patch-sentinels.sh
   sh tests/upstream-sync-test.sh
   git diff --check
   sh -n install.sh
   sh -n scripts/ios/install-beszel.sh
   sh -n tests/install-sh-test.sh
   sh -n tests/bootstrap-test.sh
   shellcheck -s sh install.sh
   shellcheck -s sh scripts/ios/install-beszel.sh
   shellcheck -s sh tests/install-sh-test.sh
   sh tests/bootstrap-test.sh
   sh tests/install-sh-test.sh
   expected=$(sed -n 's/^engine_sha=//p' install.sh)
   printf '%s  scripts/ios/install-beszel.sh\n' "$expected" | sha256sum -c -
   ```

7. On a macOS runner, build both `GOOS=ios GOARCH=arm64` outputs using the
   workflow-equivalent frontend build, A7 runtime patch, iPhoneOS clang
   wrapper, and exact output names. Validate Mach-O, arm64, iOS 12.0 load
   commands, checksums, and the three-file artifact.

8. Compare the functional surface with the parity checklist. Only after all
   automated and build checks pass should a release candidate be considered.
   Real-device validation on the reference hardware is required before
   publishing or marking a new iOS release Latest.

## Drift check

`.github/scripts/check-upstream-drift.sh` is a read-only informational check.
It parses the `Version` assignment in `beszel.go`, queries upstream tags with
`git ls-remote --tags`, considers only exact stable `vX.Y.Z` tags (annotated or
lightweight), and prints both the iOS base and newest upstream stable version.
It does not fetch into the checkout, edit source, merge, bump versions, create
tags, create issues, or publish releases.

Exit codes:

- `0` — exact stable versions match (`IN SYNC`).
- `1` — stable-version drift or an unexpected iOS base ahead of upstream.
- `2` — malformed source, missing stable tags, or remote/query error.

The comparison deliberately reports stable-release drift, not every untagged
commit on `upstream/main`. Untagged main changes still require the manual audit
above. `tests/upstream-sync-test.sh` uses a local fixture remote to test tag
parsing, version reporting, and all status outcomes without network access.

`.github/workflows/upstream-drift.yml` runs the check weekly and on manual
dispatch with `contents: read`, no secrets, no write permissions, no issue
spam, and no release action.

## iOS patch regression sentinels

`tests/ios-patch-sentinels.sh` performs static, fail-loud checks for:

- A7 runtime patch presence, exact source-shape guarding, `CNTVCT_EL0` /
  `CNTFRQ_EL0` recognition, and runtime-patch-before-build ordering.
- `GOOS=ios`, `GOARCH=arm64`, complete Go-source workflow path coverage,
  `-mios-version-min=12.0`, Mach-O/arm64 checks, and iOS load-command checks.
- Both Agent/Hub output names, `SHA256SUMS`, and the two build entrypoints.
- iOS battery source/build tag, `AppleARMPMUCharger`, `ioreg`, and the
  `darwin && !ios` exclusion.
- Installer/release tag syntax and exclusion of iOS tags from upstream
  GoReleaser/Docker workflows.
- The `//go:embed all:dist` contract, ignored `dist`, and frontend-before-Go
  build ordering. No empty fake frontend directory is accepted.
- The iOS system metadata hook and non-iOS no-op build-tag companion.

The sentinel checks are source/workflow guards, not a substitute for the
macOS cross-build or real-device test. The runtime patch script itself is also
tested against an isolated copy of the current Go runtime and an intentionally
unknown shape.

## Build reproducibility findings

The fresh-checkout workflow establishes the required prerequisites explicitly:

1. The iOS workflow runs `tests/ios-patch-sentinels.sh` and the deterministic
   `tests/upstream-sync-test.sh` before installing toolchains or compiling.
2. `actions/setup-go` reads the exact Go version from `go.mod` (`1.27.1` at
   this audit snapshot).
3. `oven-sh/setup-bun` installs pinned Bun 1.4.0.
4. `bun install --frozen-lockfile` and `bun run build` populate the ignored production
   `internal/site/dist` before compiling the Hub.
5. The runtime patch runs before either Go build and refuses an unknown
   runtime implementation.
6. `xcrun` resolves the current iPhoneOS SDK and clang; the wrapper carries the
   arm64 and iOS 12.0 flags.
7. Agent and Hub are built independently to the two contract names.
8. `file` and `otool` validate Mach-O, arm64, and the minimum deployment load
   command.
9. `shasum -a 256` creates and verifies `SHA256SUMS`.
10. Only the three release payload files are uploaded; the release job verifies
   the exact three assets and two checksum entries again.

The workflow push paths now include root and nested Go files, so changes under
`internal/hub`, `internal/migrations`, `internal/entities`, and other Go
packages cannot silently skip the iOS build. Documentation-only changes still
do not trigger branch builds; iOS tag pushes always run the full pipeline.

`internal/site/dist` is ignored and absent after a fresh checkout. A direct Hub
build without the real frontend build is therefore unsupported and must not be
“fixed” with an empty directory: that would produce a Hub without the
production dashboard. Linux local environments cannot reproduce the macOS
iPhoneOS CGO/Mach-O build; the macOS workflow remains authoritative for that
check.

The frontend dependency step is reproducible from a clean checkout: the
checked-in `internal/site/bun.lock` (lockfileVersion 3, generated with Bun
1.4.0) reflects the `package.json` `overrides` entry
(`@nanostores/router` → `nanostores ^0.11.3`, inherited from upstream since
0.19.0), so `bun install --frozen-lockfile` succeeds without modifying the
lockfile and is followed by the real production build
(`bun run build`: Lingui extract/compile + Vite). `oven-sh/setup-bun@v2` pins
Bun 1.4.0 in `.github/workflows/ios-build.yml`. Do not hide frontend issues
with an empty `dist` directory; the macOS workflow remains authoritative for
actual iPhoneOS binaries.

## Release versioning policy

The code version and installer version are independent:

- Upstream base `0.19.0` plus first iOS binary revision:
  `v0.19.0-ios.1`.
- A new upstream base `0.20.0` resets the binary revision:
  `v0.20.0-ios.1`.
- Installer-only, documentation-only, or CI-only changes do not create
  `ios.2`.
- A Go source change affecting Agent or Hub binaries on the same upstream base
  increments the revision: `v0.19.0-ios.1` → `v0.19.0-ios.2`.
- Release notes must name the actual upstream base. Never claim a version newer
  than the source integrated into the release.
- `INSTALLER_VERSION` remains `1.0.0`; this maintenance audit does not bump it.

No release tag, binary revision, or installer version was created by this
maintenance audit.

## Documentation language maintenance

English documentation (`README.md`, `docs/*.md`) is canonical. Vietnamese
counterparts (`README.vi.md`, `docs/*.vi.md`) are secondary translations.
Update the English canonical document first, then synchronize the Vietnamese
translation so both languages keep the same facts.

## Future parity checklist

Apply this checklist to every future upstream bump. Mark unsupported metrics as
unsupported; do not turn absence of a platform metric into a failure.

### Agent

- [ ] Starts under the reference iOS launchd environment.
- [ ] Connects to the Hub and reconnects after a Hub restart.
- [ ] CPU usage and CPU-core data where supported.
- [ ] Memory and swap data where supported.
- [ ] Filesystem usage and root/custom filesystem handling.
- [ ] Disk I/O and disk-device mapping; test iOS mount edge cases explicitly.
- [ ] Network throughput and relevant interfaces.
- [ ] Uptime and load data where supported.
- [ ] Battery percentage/state through `Primary` on the reference device.
- [ ] Temperature/fans/GPU/SMART/ZFS only where the device and environment
  actually expose them; otherwise record unsupported.
- [ ] Process/container/system behavior as applicable to the device.

### Hub

- [ ] Starts and remains healthy.
- [ ] `/api/health` returns success.
- [ ] Login, first-account/bootstrap, and account settings work.
- [ ] Systems list and Agent authentication work.
- [ ] Historical database opens without data loss.
- [ ] New upstream statistics appear when supported.
- [ ] Migrations apply and preserve existing data.
- [ ] Embedded frontend/dashboard loads after a real frontend build.
- [ ] Agent reconnects and resumes history after a Hub restart.

### Installer

- [ ] Fresh Agent install.
- [ ] Fresh Hub install.
- [ ] Fresh Agent + Hub install.
- [ ] Component update and update no-op behavior.
- [ ] Failed update rollback.
- [ ] Read-only diagnostics.
- [ ] Repair and missing-component restore.
- [ ] Agent/Hub reconfiguration with config rollback.
- [ ] Uninstall with data preserved by default.
- [ ] Retained-data reinstall.
- [ ] State tracking and checksum behavior.

### Persistence

- [ ] LaunchDaemon labels, paths, ownership, and permissions remain correct.
- [ ] After a full reboot, manually reactivate the semi-untethered jailbreak;
  then verify both LaunchDaemons return automatically.
- [ ] Recheck Agent reconnect, Hub health, and data/history persistence after
  reactivation.

The validated reference for these checks remains iPad mini 2 / `iPad4,4` /
`A1489` / Apple A7 / iOS 12.5.7. Other devices and iOS versions remain
untested.

## Current next-phase decision

There is no newer stable upstream release than `v0.19.0`, so no
stable-base integration is required and no new release is created.

There is an optional future integration phase for the untagged upstream main
range:

- Proposed target: `upstream/main` at `4bf70700`, range
  `f204dc17..4bf70700`, containing the eight commits classified above.
- Preferred release target: wait for the next stable upstream tag and audit
  that tag; do not label the untagged main tip as a stable Beszel release.
- Expected textual conflicts: none in the current iOS-owned files. Review
  `agent/storage_pool.go` / `agent/zfs/*` build selection and inspect
  `agent/system.go` / `agent/battery/*` context after integration.
- Go/runtime implication: no `go.mod` change in the pending range; rerun the
  A7 source-shape check against the installed Go runtime anyway.
- Database migrations: none in the pending range.
- Agent↔Hub protocol, ports/CLI, and health endpoint: no changes observed.
- If the range is integrated and released while the base remains `0.19.0`,
  the binary revision would be `v0.19.0-ios.2` after all checks. No such
  revision is being created now.
- Before publishing, repeat the full automated/build/parity checklist and the
  real-device validation, including reboot plus jailbreak reactivation.
