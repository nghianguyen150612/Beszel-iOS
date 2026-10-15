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
