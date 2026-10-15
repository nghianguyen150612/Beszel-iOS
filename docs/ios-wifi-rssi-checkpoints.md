# P012 durable checkpoint ledger

Branch: `ios/wifi-rssi-compatibility`; base: `ios/upstream-v021-integration`.
Starting integration `33840093bb465f0b164bb7d66aebc8cd85f86a4e`.
Production iOS `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`;
frozen main `bb23f99b3eeaced9608e11e280e7c986dba650be`.
No existing P012 branch or PR was found. Clean existing checkout reused; all
baseline refs fetched and exact values verified; frozen main is an ancestor.

## A — feasibility

Completed: inspected authoritative P003/P007–P011 documentation, current Wi-Fi
sources, entity contracts, Agent polling and CI/source-selection rules; fetched
pinned historical headers, iOS 12.4 exports and Procursus source. Specific native
ABI/freshness/authorization/bounded-call blockers are in the feasibility record.
Chosen policy: explicit iOS unavailable fallback, no speculative native calls.

Focused baseline Wi-Fi/entity tests and diff check: see commit validation.
Remaining: explicit adapter/selection tests, history input validation, polling
and protocol regressions, full regressions, native CI, EN/VI/JSON evidence.
Native RSSI acceptance remains deferred. Keep PR Draft; no merge while required
P012 native capability is blocked. P013 has not begun.

## B — explicit iOS adapter and source selection

Completed: `wifi_ios.go` owns iOS selection and returns nil without private calls;
unsupported targets keep their fallback. The validator requires the iOS file and
negative fixtures reject macOS/Linux/Windows/generic Wi-Fi implementations.
Production adapter compiled and executed by explicit Go file list on the cloud
host: race tests, 3 repetitions, 1,000 serial polls and 3,200 concurrent Collect
calls per repetition. These prove only unavailable behavior. Build-contract
Python tests pass. No native API boundary exists to simulate or authorize.
Remaining: shared history validation, Agent snapshot lifecycle/protocol tests,
full regressions, final-head hosted native builds and consistent evidence.

## C — signal and SSID semantics

Confirmed shared defect: Signals converted positive/huge/non-finite inputs to
int8, potentially wrapping invalid data into plausible negative history. It now
rejects inputs outside the collectors' -150..-1 dBm range and NaN before rounding.
The existing -150..-129 to -128 history saturation policy is retained; no shared
model or protocol field changes. Raw precision remains in Info for valid sources.
Typical/weak/edge/sentinel/overflow/normalized-signal and UTF-8/CBOR fixtures pass
under the race detector, 10 repetitions. The fallback has no RSSI/CF parser;
native object/dictionary fixtures are not claimed as executed.
Remaining: Agent polling lifecycle and remote protocol regressions, full checks,
hosted builds and evidence. Native provider blockers remain unchanged.

## E / F — authorization, resources and CI coverage

Completed: documented candidate manager-access privilege evidence and its limits;
no entitlement, signing, service or release change. The fallback acquires no
native objects, framework handles, workers, or persistent state and returns
synchronously even for a canceled context. The shared two-second context is
unchanged. Native IPC cancellation remains a blocker, not a claimed guarantee.
Actual iOS adapter + shared tests passed 20 race repetitions on the cloud host;
iOS build-validator tests pass. Added hosted fallback/lifecycle/race tests and
native dependency/import inspection; native CI has not run at this checkpoint.
README and capability matrix now describe unavailable behavior accurately.
Remaining: full stable-source regressions, hosted CI, EN/VI/JSON evidence and
final review. Native capability deferred; PR remains Draft; no merge/P013.

## D — lifecycle and protocol

Completed: extracted the existing default/realtime Wi-Fi update into a small
production method with a collector parameter. Fixtures exercise connected,
disconnected, disabled, denied, temporary/repeated failure, renamed/multiple
interfaces, hidden SSID, new SSID and recovery. Default acquisition replaces old
state; five realtime requests per step reuse the snapshot without acquisition.
JSON/CBOR roundtrips cover absent/SSID-only/RSSI-only/multiple/UTF-8 snapshots.
Pinned numeric-key legacy wire fixtures exercise the actual Hub transport decoder
for Linux/macOS/Windows interface names, omission and recovery. Existing synthetic
Hub record disconnect/reconnect and WebSocket omission tests pass. Race tests:
3 repetitions. No production DB was used; test apps use temporary fixtures.
Remaining: entitlement/resource documentation, full regressions and hosted CI.

## G — stable-source full regressions and documentation

Source checkpoint: `d3f33c83c8d6164717a26749826408359fed84ea`.
PASS: canonical `env -u NETWORK go test -tags 'testing no_ui' ./...`; full Agent,
Wi-Fi, battery, delta tracker, transport and WebSocket race packages. The broad
race command's explicit 300-second per-package limit expired in Hub systems;
that package then passed alone with the standard Go timeout (140.535 seconds),
without source changes. The focused lifecycle/transport/Hub record race run
passed three repetitions in a separate clean log. P007 Python runtime/ABI source
fixtures and P008–P012 portable regressions pass. The P011 actual APT read-only,
network-disabled fixture passed three repetitions on this cloud host.

PASS: upstream matrix/readiness, 49 audit tests, sync 10, sentinels 45, bootstrap
54, installer 814, diff check. Actual ancestry: 178/178, 13 logical patch IDs.
PASS: real frozen Bun 1.4.0 install, catalogs, 44 tests, production build with 95
assets and unchanged lockfile. Generated PO edits restored. iOS/macOS/Linux
ARM64 CGO=1 source selection passed. Linux Agent/Hub built; Windows Wi-Fi and
FreeBSD fallback test executables compiled (not executed).

EN/VI reports and machine-readable evidence prepared. Hosted iOS Build on this
source: https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38724163923 .
Installer PASS: https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38724168872 .
The source native build result/artifacts are pending at this ledger entry.

The shell GitHub credential expired during validation (401); connected GitHub
operations remain available and public unauthenticated reads work. Use connected
Git tree/commit/ref tools to publish the exact reviewed tree, then fetch it
without the expired credential and reconcile local HEAD by verifying tree
identity. Never print credentials or force-push. Workflow branch filters now
include this feature branch so the final checkpoint receives final-head checks.

Remaining: collect native artifact/provenance/import/checksum evidence, publish
this documentation checkpoint, verify final-head hosted CI, and leave Draft
because native acquisition remains blocked. Required merge/post-merge steps are
inapplicable until those blockers are resolved; P013 has not begun.

## H — source hosted native verification complete

Source SHA `d3f33c83c8d6164717a26749826408359fed84ea`: iOS Build PASS,
installer PASS. Downloaded both real artifact ZIPs through connected GitHub;
verified their GitHub archive digests, payload SHA256SUMS, arm64/iOS platform 2 /
minimum 12.0 metadata, Go 1.27.1 CGO=1 A7 provenance and all 95 embedded assets.
Agent SHA256 `993c82e71cf570ee490c731ee1ea2a6299b7e6573515b6b1790f707d962f7f6f`;
Hub SHA256 `b3ea5f9327e5024ee84f2870088dc01a243ea011df522f1b4bf53b7054cb10b1`.
Private Wi-Fi dependencies/imports absent. Isolated A7 patch, instruction checks
and separate-cache repeat builds pass. Release job skipped. Full evidence and
matching EN/VI reports are ready for publication.

Next: publish this exact reviewed tree, verify new final-head push workflows and
record final SHA/results in PR #14. Keep Draft: native RSSI remains blocked and
Gate C pending. Do not merge, start P013, deploy, sign or modify devices/data.

## P012A A — resume and firmware-specific investigation

Resumed exactly `88dcf9b89389cd58de6f4859d4619485451dfa82`; all four remote refs
match the expected baselines; clean branch and Draft PR #14 retained. Reviewed
previous EN/VI/JSON evidence, ledger, source, workflow and final CI comment.

New: machine-readable per-symbol ABI/ownership manifest with exact pinned source
hashes, conservative evidence statuses, and firmware-derived build 16H81 metadata.
Reusable declaration validation rejects mismatched source bytes. Found the exact
Apple reference restore image; no smaller verified extraction was found. A
separate cloud-only workflow will inspect the checksum-pinned image read-only,
without running Apple firmware or touching hardware. This inspection is pending.

Focused ABI/firmware manifest tests and existing iOS build tests pass. Production
polling fallback is unchanged. Next: collect firmware evidence, implement strict
helper result conversion/state fixtures and bounded process orchestration.
P012 remains BLOCKED; Gate C PENDING; P013 not started.

## P012A B — production helper protocol and state conversion

Added the inactive `agent/wifi/ioshelper` package, not imported by normal Agent
polling. Strict versioned JSON conversion enforces size/count/depth limits,
rejects duplicate/unknown keys, wrong encodings/types, incomplete/non-station or
unknown/disconnected/powered-off association, and nonexistent OS interface names.
Only integer RSSI_CTL_AGR in -150..-1 is accepted; scaled/float/positive/overflow/
sentinel data is unavailable. Hidden or invalid SSIDs do not suppress valid RSSI.
No identifiers are invented; no prior snapshot is cached. The producer's claimed
association is not treated as independent evidence of native freshness.

Actual production parser tests pass ten race repetitions, including all requested
association, multiple-device, SSID, failure/recovery and JSON/CBOR contract cases.
The exact Apple restore image was checksum/model/build verified by research run
38738978309, but MobileWiFi is cache-only; cache extraction is the next ABI task.
An intermediate normal build was canceled to avoid repeating unchanged-Agent CI
while source work remains in progress. Final stable source will get full CI.
Normal iOS acquisition remains nil; no private call or entitlement was activated.
Next: bounded client and fault executables, cache extraction, native CF fixtures.

The first exact-image research run succeeded in checksum, j85ap board and
SystemVersion verification, and established that MobileWiFi is cache-only.
The follow-up extractor uses Apple's host dsc_extractor bundle (signature pinned
to Apple dyld 852.2 source), never executes an iOS binary, and records the actual
reference cache/hash and extracted exports. This requires another image read
because the first disposable job preserved only its small derived report.

## P012A C — bounded inactive helper client

Implemented the production client boundary without importing it into Agent
polling. Absolute regular executable, fixed argv, minimal environment, discarded
stderr, 16 KiB stdout cap, two-second acquisition context, one active process,
process-group cancellation, bounded pipe wait and direct-child reaping. Close
prevents new acquisition and cancels/awaits the existing child with an explicit
caller deadline. No source snapshot survives failure; no goroutine is abandoned.

Actual fake executables test valid/unavailable/empty/malformed output, crashes,
nonzero exit, missing/symlink/non-executable paths, real permission denial,
output flood, repeated timeouts, inherited pipe holders, 64 parallel callers,
cancellation, shutdown and recovery. Three race repetitions passed. Each direct
child must disappear after Wait; Linux deliberately orphaned fixture descendants
are adopted/reaped by test-only instrumentation, not a production subreaper.
The trusted native helper will be single-process; iOS kill/wait behavior and
abnormal kernel behavior are not inferred from these host tests.

Research run 38740375596 failed before cache extraction: host clang lacked the
macOS SDK sysroot. Corrected the command, kept this failure recorded, and added a
checksum-validated restore download cache to avoid repeated Apple transfers.
Next: finish cache evidence and public-CF conversion/ownership fixtures, build a
separate non-acquiring native diagnostic, and document minimal privilege/P018.

## P012A D — exact firmware exports and disabled native diagnostic

Checkpoint C published as `6b859dd6bdf4b7be457546567ba77ef5dfc4e070`.
Research run 38741425098 PASS: exact checksum/model/build/root confirmed, reference
ARM64 cache extracted, all six exports present; binary/report/disassembly hashes
recorded in the ABI manifest and retained small derived evidence. Typed ABI is
strongly inferred, not declared verified. Several functions lack NULL argument
guards. Association freshness/raw units/minimal authorization remain unresolved.

Added a separate disabled C diagnostic: identity/build-info/symbol inventory only;
its snapshot always unavailable. No private function pointer is cast or invoked.
Public-CF conversion/owned-container fixtures and loader fault inventories are
prepared for hosted sanitizer execution; they have NOT run on this Linux host.
The hardened workflow builds the iOS diagnostic independently, excludes it from
normal Agent/Hub payload, checks its Mach-O/provenance/imports, and executes host
fixtures plus actual C-snapshot→Go-client integration. Source validator additionally
rejects inactive-client imports and checks Wi-Fi selection for five platforms.

Focused Python tests and Go helper race tests pass locally. The native diagnostic
requires hosted macOS validation next. EN/VI blocker-resolution reports include
minimal privilege limitations and a staged read-only P018 protocol which stops
before unimplemented acquisition. No entitlement/signing/installer change.
Next: run complete final regressions and hosted native fixtures/builds, debug any
actual failures, update exact evidence and retain Draft/BLOCKED/Gate C PENDING.

## P012A E — completed native checkpoint and recovery fixture gaps

Resumed/refreshed remote `aac215abdb89c5fa6bc307a550216d044f0906a9`; protected
refs unchanged. Hosted run 38742936483 PASS, installer 38742936311 PASS, release
SKIPPED. Actual public-CF ASan/UBSan fixtures, ownership allocator counters, loader
inventories and C-unavailable-snapshot→production Go client passed (three races).
LeakSanitizer disabled/unsupported, not a pass. Downloaded three artifact ZIPs,
verified their GitHub digests, Agent/Hub SHA256SUMS, independent C revision/hashes,
ARM64/platform 2/minimum 12.0, clean Go provenance, A7 runtime evidence and 95 assets.
No private call/device execution was performed.

Stable-source full cloud validation PASS: matrix/readiness, 54 audit tests, sync
10, sentinels 45, bootstrap 54, installer 814; canonical Go; full affected race
packages, focused three repetitions and iOS fallback ten repetitions; genuine
frozen Bun install/catalog/44 tests/production 95 assets; five-platform selection;
Linux Agent/Hub builds; Windows/FreeBSD Wi-Fi and inactive-client compile only.
All 178 upstream commits reachable; all 13 real logical patches accounted for
(`none` rows excluded). Normal iOS/Linux/macOS/Windows collectors unchanged.

Recovery review found two fixture gaps, not an active provider defect: invalid
existing library and proof of dlclose before exit. Extracted actual production
inventory into loader.c, preserving fixed-path/no-call behavior. Portable loader
ASan/UBSan fixtures pass 100 repetitions per missing/invalid/partial/all case;
an isolated missing-close mutant is rejected. Native CF adds fractional/NaN/Inf
and owned-property cleanup with missing output. Updated EN/VI/JSON results and
scope policy; no general fallback waiver is used to change native acceptance.

Next exact action after publication: complete final-candidate native CI (new
loader and extended CF tests), inspect/revalidate artifacts and record final HEAD,
checks/digests in PR #14. No need to download/extract firmware again. Test command:
`python3 scripts/audit/ios_wifi/native.py --host --output "$RUNNER_TEMP/wifi-host" --revision "$GITHUB_SHA"`
then workflow's actual C→Go client race integration and separate iOS build.
Keep Draft/BLOCKED/Gate C PENDING; typed opaque ABI/fresh association/raw dBm/minimal
authorization and actual iOS process execution remain P018. No P013/merge/device.

## P012A F — deterministic P009 regression fixture repair

Checkpoint E: `b2fa8b0998fe47c1ff485f8912e260a85a09b2bf`. Its final-head installer
38823770552 PASS; hosted extended CF/ASan/UBSan/immediate-loader-unload and C→Go
integration steps PASS in 38823738085. That native build is superseded during
validation, not counted as a completed final-head build.

Canonical Go and full affected race suite PASS again. A three-repeat focused run
exposed an existing test-fixture race in TestP009ConcurrentConnectClose: queue
draining can precede a delayed old server upgrade; connect(t) then pings a closed
stale socket. Two separate wrapper/socket channels can also interleave upgrades.
This was an actual FAILED test, not ignored as a timeout or relabeled PASS.

Repair is test-only: publish each wrapper/socket pair atomically and select the
server socket matching the current client's real TCP endpoint. Keep all existing
ping/read-loop/auth/recovery assertions. A deterministic queued-stale-upgrade case
plus the original concurrency case PASS 50 race repetitions. Production network
transport and Wi-Fi sources are unchanged. Failure log/result copy retained.

Next after publishing: canonical Go and affected Agent race/focused regressions;
final-head iOS Build and installer; artifact revalidation; final PR checkpoint.
No firmware extraction, toolchain reinstall or unchanged frontend rebuild needed
locally. Existing independent Hub/platform/frontend results remain valid; hosted
final-head workflow repeats its required checks. P012 BLOCKED / Gate C PENDING.
