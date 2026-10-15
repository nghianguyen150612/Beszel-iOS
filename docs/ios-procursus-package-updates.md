# P011 — Read-only Procursus/APT package update telemetry

## Status and provenance

P011 adds local package update telemetry for a supported Procursus APT installation on iOS. The adapter only simulates an upgrade against already cached metadata. It cannot refresh indexes, download packages, or change installed packages.

Repository: https://github.com/nghianguyen150612/Beszel-iOS
Branch: ios/procursus-package-updates
Base: ios/upstream-v021-integration
Starting integration: c2bd226f54f52a5fb2efaa33a63aa9fb632f4216
Starting production iOS: 0c3e4cd22ac24264834c401ad412e44aeda4e4f3
Frozen main: bb23f99b3eeaced9608e11e280e7c986dba650be
PR: https://github.com/nghianguyen150612/Beszel-iOS/pull/13

The baseline refs were fetched and verified before edits. The frozen main commit remains an ancestor of integration. The upstream matrix validator accounts for all 178 frozen upstream commits and 13 logical iOS patches. The complete readiness validator passes. Implementation checkpoints are pushed to the feature branch; the PR is kept Draft until final review and hosted checks.

The authoritative Procursus source reviewed was ProcursusTeam/Procursus at b6185221bf1ce3dda8b4baeee61e75e3f7c7d399. Its APT recipe pins APT 2.9.4; the package recipe and control metadata identify the Procursus package/keyring context. The APT 2.9.4 apt-get manual documents simulation and no-download behavior. This is source and fixture evidence, not proof of the reference iPad's installed binary or configuration.

## Detection and supported layouts

The iOS-specific factory is selected only for GOOS=ios. Linux package-manager detection remains in the existing Linux branch; macOS and Windows receive no Procursus factory.

Detection checks these explicit executable candidates:

- Rootful: /usr/bin/apt-get, /usr/bin/apt-config, /usr/bin/dpkg-query
- Rootless: /var/jb/usr/bin/apt-get, /var/jb/usr/bin/apt-config, /var/jb/usr/bin/dpkg-query

All three must be regular executable files. Bounded, allowlisted queries check apt-get --version, apt-config dump, and whether dpkg-query reports apt and procursus-keyring installed. Detection reads APT's reported Dir, Dir::State, Dir::State::status, Dir::State::lists, Dir::Etc, source-list, source-parts, main-config, and config-parts settings. It validates the resolved status file, APT lists directory and cached Packages index, and relevant configuration paths. It does not assume /var/lib/dpkg or /var/jb/var/lib/dpkg without consulting APT configuration.

The implementation supports the rootful and rootless Procursus shapes when the installed APT configuration resolves a consistent package database, lists, and /etc/apt tree. If both candidates point at conflicting installations, metadata is missing, commands are incomplete, the variant/keyring is unsupported, or required metadata is unreadable, the adapter is unavailable. It never repairs or populates the installation. Other binary locations fail closed.

The three-second detection context bounds startup probes. No Linux package manager such as DNF, Zypper, Pacman, APK, or XBPS is searched or selected by the iOS factory.

## Read-only query and integrity evidence

The only update-count argv is:

    apt-get -s --no-download -o Debug::NoLocking=1 -o Dir::Cache::pkgcache= -o Dir::Cache::srcpkgcache= dist-upgrade

Arguments are passed directly to the validated apt-get path; no shell is involved. The exact executable basename and argv are allowlisted at the subprocess boundary. The simulation flag is supported by reviewed APT 2.9.4 documentation and requests no action or system change. The no-download option prevents package acquisition. Simulation disables locking; Debug::NoLocking makes that policy explicit. Empty package-cache destinations prevent apt cache files from being persisted. Standard output is bounded to 1 MiB, stderr is discarded, and the environment is reduced to LC_ALL=C. A 20-second iOS check context bounds execution.

The query does not run apt-get update and does not create repository metadata. Missing indexes cause the query to fail as unknown. No prerequisites are installed. Results describe the locally cached APT view, whose freshness depends on when the existing metadata was last refreshed by the device owner. The adapter never refreshes it and does not label the count as live repository status.

A disposable APT fixture points at a local loopback source and uses an isolated package database, status/available records, lists, extended state, archive/cache paths, dpkg administration files, and an APT config. The actual host apt-get simulation ran in a bubblewrap sandbox with a network namespace disabled and the filesystem read-only. The loopback listener observed zero connections. The invariant snapshot covered 30 files/directories, including file hashes, modes, sizes, and modification times. The test ran three consecutive times. Each run found all 30 fixture entries and their metadata unchanged; before and after snapshot SHA256 matched in every run: `2bc11d3c9d0bc989db04c1b1f161c9fafc15973d2e5734fcb38654e1aff669d9`, `09c9d24415ee87d2aee29f14bac67f368ca54252a50175dfac85d8a29ac77c62`, and `4aa97a989d6dbf6a3201406a94f710726e79997e59a4e45be6809cce2edfd4ed`.

The sandbox host had APT 3.0.3. Procursus source pins APT 2.9.4, and the compatible options were reviewed against its manual. The disposable run does not prove the reference device's APT behavior. APT output, credentials, source URIs, and subprocess error text are not emitted to normal logs or sent to the Hub.

## Parsing and result semantics

The checked-in apt_procursus_simulate.txt fixture is sanitized and synthetic, based on the documented APT simulation grammar; it was not captured from a physical Procursus device. Tests cover zero and one update; multiple updates; long names; digits, hyphens, plus signs and architecture qualifiers; epoch, tilde, plus and punctuation in versions; repository descriptions and warnings; new dependencies without an installed version; held, removed, obsolete, equal-version and downgrade entries; duplicate and conflicting entries; malformed, localized, empty and truncated output; missing current/candidate versions; oversized output; and the package-count limit.

Only an Inst entry for a package with an installed version and a strictly newer candidate is counted. New dependencies have no installed version and are not updates. Removed packages, held packages, equal versions and downgrades are not counted. Identical duplicate entries are deduplicated; conflicting duplicates or a summary that does not reconcile with parsed entries make the check unavailable. Results are sorted deterministically. Output is capped at 1 MiB and 8,192 package entries, far below the uint16 count limit.

APT's distribution-specific security repository labels are not a reliable Procursus advisory source. Procursus results set SecurityKnown=false and every PackageUpdate.Security=false. No security count or advisory data is fabricated.

The existing shared model and wire contract are unchanged: Agent system stats keep the existing []uint16 field; nil means no successful/available count, while []uint16{0} means a successful check found zero updates. JSON and CBOR tests verify these states remain distinct. The existing GetPackageUpdates action, system.PackageUpdates entity, CBOR/JSON mappings, Hub UI, and remote Linux/macOS/Windows telemetry are preserved. No PocketBase migration or new protocol field is added.

## Timeout, scheduling, cache and recovery

The adapter uses the existing PACKAGE_UPDATES_INTERVAL setting and prefixed environment lookup. An unset value uses the existing one-hour default; a positive duration is honored; zero disables the manager; invalid or negative input follows the existing safe default policy.

Detection is bounded to three seconds. An update query has a 20-second context, a 1 MiB output cap, discarded stderr, a two-second os/exec WaitDelay, and process-group cancellation on Unix so children retaining stdout are killed with the query. Commands use exec.CommandContext directly. Tests cover missing and non-executable commands, nonzero exit, timeout, cancellation, oversized output, a child retaining stdout, and recovery after a failed query.

Checks run in one background refresh and callers share that refresh. They are scheduled by the configured interval, not by every stats request. A failed attempt is throttled for the interval and reports unknown, not zero. Failure clears old cached details and leaves CheckedAt zero. For iOS, the timestamp-free stats count becomes nil when an expired result begins refreshing. The detail action may expose the previous successful package list only with its original CheckedAt until the refresh finishes; a failed refresh clears it. Shutdown cancels the manager context and waits briefly for the refresh to exit. Failures do not block CPU, memory, battery, network, or disk telemetry.

## Regression and native validation

Focused Procursus detection, path selection, parser, argv allowlist, read-only database invariants, required no-network sandbox, package count limit, timeout, cancellation, child cleanup, manager cache/recovery, serialization, and handler tests pass under pinned Go 1.27.1. The focused race suite passes three repetitions; process fault tests pass repeated runs. The full canonical Go suite and full Agent race suite pass. Linux Agent and Hub builds pass.

The iOS/macOS/Linux ARM64 CGO=1 source-selection validator confirms that the iOS adapter and Unix process helper are selected only in supported builds. Focused Linux regressions cover APT, DNF, Zypper, Pacman, APK, XBPS, handler/cache/security behavior, and container suppression. The audit suite (49 tests), upstream matrix/readiness validators, iOS sentinels (45), upstream-sync test (10), bootstrap test (54), and installer lifecycle test (814) pass. Two existing connection-concurrency tests failed once in intermediate full-suite attempts; the final serial canonical run passed, and no P009 implementation/test file changed in P011. Final hosted native build, installer workflow, and P007–P010 regressions passed on source SHA `d777ad42fe8cfc2e2dec5ab5f942659abc49eae9`; their run URLs and artifact evidence follow.

Final hosted validation ran on feature source SHA `d777ad42fe8cfc2e2dec5ab5f942659abc49eae9`:

- iOS Build: PASS — https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38675307900
- Installer lifecycle: PASS — https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38675307584
- The package-update hosted job passed detection/parser/argv/failure tests. Its disposable network-isolated APT query was explicitly skipped because the Ubuntu runner lacked a network-disabled, read-only filesystem sandbox; the separate local bubblewrap fixture passed three times as described above.
- Runner: macOS 14.8.9 arm64; Xcode 15.4; iPhoneOS SDK 17.5 and Xcode clang. Pinned Go 1.27.1 and Bun 1.4.0 were used. Production frontend install, catalog generation, tests and build passed, and the actual embedded Hub contains 95 assets.
- iOS source selection passed. Agent and Hub built twice with separate caches and byte-identical results. The isolated Go 1.27.1 A7 runtime patch passed; runtime instruction validation found no CNTVCT/CNTFRQ access in either binary.
- Both outputs are arm64 `MH_EXECUTE` Mach-O for iOS platform 2 with minimum iOS 12.0. VCS provenance reports revision `d777ad42fe8cfc2e2dec5ab5f942659abc49eae9` and `vcs.modified=false`.
- Agent SHA256: `9afffa924917c46bd48f18175bb8532d85692664edd642522bf7e5798f46754e`; Hub SHA256: `7eec915001fb1ec3d7a36072b50afefef258147a792b9d1da610dd4af2676a8e`. Both artifact checksums were verified by the workflow.
- `release-ios` was skipped. No release, tag or deployment was created.

At this source candidate, P011 implementation, test, compatibility and hosted build gates pass. The PR remained Draft while final diff review and approval requirements were checked. The source SHA above is the native-tested candidate; subsequent commits only update the evidence files and PR metadata.

## Confirmed limitations and acceptance

Confirmed starting defect: the existing factory returned nil for every non-Linux platform, so the iOS Agent never instantiated package update telemetry. P011 adds only the iOS path and retains the Linux detector. The shared schema has no explicit freshness field for system-stat package counts; P011 uses absent/nil for unknown or stale counts while an iOS refresh is in flight. Package detail results retain their existing CheckedAt timestamp.

Cloud disposable fixtures establish implementation behavior only. The iPad mini 2 / A1489 / iPad4,4 on iOS 12.5.7 with Amethyst and Procursus was not accessed. Its actual package-manager binaries, APT version/configuration paths, read permissions, metadata freshness, and query behavior remain unverified. Physical Procursus read-only validation is deferred to P018 / Gate F. No real package database, production Hub database, or device was modified.

P011 acceptance gate passes on the native-tested source SHA above: the implementation, tests, cross-platform regressions, documentation, and final-head hosted CI satisfy the stated acceptance criteria. PR review/approval and normal merge are tracked separately in the pull request and completion report. Gate C advances with P011; native Wi-Fi RSSI remains P012. Database/restore and release/deployment gates remain pending. P012 is not started.
