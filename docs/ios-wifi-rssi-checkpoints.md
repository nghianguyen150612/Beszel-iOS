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
