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
