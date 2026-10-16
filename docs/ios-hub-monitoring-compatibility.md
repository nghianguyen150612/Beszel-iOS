# P014 — Hub monitoring rebuild

Status: IN PROGRESS. This is fresh evidence for PR #17 on
`ios/hub-monitor-alert-rebuild`, based on P013
`00c14bbf2756213cc2afef828ef7d5b9af26f1d6`. PR #16 is abandoned, closed and
unmerged. Its source and test results are not used as the implementation baseline.

The Hub's shared entities, transport decoding, persistence, rollups and alert
logic remain independent of local iOS collector selection. Four independently
authored synthetic Agent fixtures describe Linux, Windows, macOS and iOS. They
are protocol coverage on Linux, not native remote-OS or iPad execution.

## Confirmed defects

The fresh baseline regression rejects numeric `cpus` under JSON v2 and historical
base64 `bat` under JSON v1. Strict custom decoding now accepts either representation
and rejects invalid byte values and malformed battery tuples. Current numeric
JSON output and CBOR keys remain unchanged. Entity and fixture suites pass.

The GPU power chart previously grouped adapters by display name and returned zero
when a sample/device lacked power. The repair uses device identity plus name,
preserves valid zero and uses gaps for missing values. Focused Bun tests pass.

Further persistence, rollup, monitor, alert, API and race results will be recorded
only after execution. Native CI remains pending. P014 has not passed and Gate D
also requires P015. No frontend route/localization/dependency acceptance is claimed.

See the [checkpoint ledger](ios-p014-rebuild-checkpoints.md),
[telemetry matrix](ios-remote-agent-telemetry-matrix.md),
[alert/monitor contracts](ios-alert-monitor-contracts.md) and
[machine-readable results](ios-hub-monitoring-validation.json).

## Boundaries

Native iOS Wi-Fi RSSI remains DEFERRED — REFERENCE DEVICE UNAVAILABLE.
Physical A7 execution, iOS trust/kernel behavior and native Windows/macOS Agent
runtime are not established by Linux synthetic fixtures. No production database,
iPad, protected branch, signing, entitlement, LaunchDaemon, Wi-Fi configuration,
release tag or deployment is modified.
