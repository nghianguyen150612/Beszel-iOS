# P014 — Hub monitoring, alerts, and telemetry compatibility

Status: **validation PASS; normal merge pending**. The current source candidate is
`bf9c4d4b3c8910ab4e2546fbbf5d4e75c93864b6` on
`ios/hub-monitor-alert-rebuild`, PR [#17](https://github.com/nghianguyen150612/Beszel-iOS/pull/17),
against `ios/upstream-v021-integration`. It starts from P013
`00c14bbf2756213cc2afef828ef7d5b9af26f1d6`. PR #16 is closed and unmerged; none
of its commits are ancestors of this branch.

## Remote data path

Independently authored, source-derived synthetic fixtures represent Linux,
Windows, macOS, and iOS with unavailable optional collectors. Synthetic WebSocket
and SSH peers exercise the real Hub decoders and polling paths. Hub tests persist
the snapshots into disposable PocketBase applications, read system details and
history through authenticated APIs, verify cross-user denial, and check reconnect
and removal behavior. The fixtures do not claim native Windows/macOS Agent or
iPad execution.

Linux, Windows, and macOS records retain their platform-specific fields when the
local iOS Agent omits those collectors. JSON v1/v2 byte metrics and legacy CBOR
remain readable. CPU, memory, disk, network, battery, Wi-Fi, GPU, containers,
SMART, ZFS, Btrfs, systemd, and package-action surfaces retain the source
contract documented in the [telemetry matrix](ios-remote-agent-telemetry-matrix.md).

## Confirmed defects and fixes

- Per-core JSON arrays emitted by current Agents and historical base64 battery
  tuples were rejected by Hub decoding. Strict decoding now handles both forms,
  rejects overflow and malformed tuples, and preserves current JSON/CBOR output.
- Sparse histories divided optional device values by every system record. Rollups
  now use per-field and per-device observation counts, preserve explicit zero,
  and skip null filesystem entries. GPU power and package power keep separate
  presence and averages.
- Missing GPU, temperature, and filesystem samples could cause false alert
  recovery or fabricated disk readings. Alert evaluation now waits for a valid
  optional sample; a fresh zero remains eligible for recovery.
- Older monitor responses and delayed run-now results could overwrite newer data
  or recreate deleted monitor records. Sample sequence ordering, configuration
  checks, and transaction-scoped ownership checks now reject stale results while
  preserving Agent clock rollback and older peers without sample counts.
- A cancelled or replaced poll could finish after its owner was retired. Poll
  context checks and joined updater registration prevent its snapshot from being
  committed. No collection schema or migration changed.
- Frontend accessors now leave gaps for unavailable values, retain valid zero,
  preserve legacy MiB/s values, and handle absent or null GPU, disk, network,
  filesystem, pool, and Wi-Fi fields.

## Validation

The canonical Go suite passed at backend source `3f153dca940651ab394663b9d85d14cbde64b957`:
`env -u NETWORK go test -tags 'testing no_ui' ./... -count=1 -timeout=20m`.
The subsequent `ca7108b7` commit only formats touched TypeScript/TSX files; the
later `bf9c4d4b` commit only adjusts the readiness validator and its regression
test. Neither changes runtime behavior. All affected Go race package groups passed, and the repaired Hub,
monitor, record, alert, and JSON decoder regressions passed repeated race runs.
The earlier `TestAgentWebSocketIntegration` race was reproduced on unchanged P013
and traced to the test reading connection state without the production mutex. Its
test accessor now uses the existing read lock; the integration test passed three
race repetitions at P014. Production transport authentication was unchanged.

Pinned Bun 1.4.0 frozen installation, catalog generation/compilation,
TypeScript build, and production build passed. The frontend suite passed 55 tests
and 267 assertions. The current build contains 95 embedded assets. Matrix and
readiness audits, all 54 Python audit tests, upstream-sync, patch-sentinel,
bootstrap, and installer suites passed. Linux Agent/Hub builds and Windows Hub
systems / FreeBSD transport test compilation passed. Cross-compilation is not
reported as native runtime testing.

Native iOS CI passed on source `3f153dca940651ab394663b9d85d14cbde64b957`:
[iOS Build](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38941248597),
[installer lifecycle](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38941249967),
and [P013 Hub migration fixture](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38941249970).
The later frontend formatting commit reran Bun tests/build, and the downloaded
native Hub independently verified against the current generated frontend assets.
The two native binaries are arm64 Mach-O, iOS platform 2, minimum iOS 12.0; the
isolated Go 1.27.1 A7 runtime patch, binary instruction checks, embedded assets,
SHA256SUMS, and production linkage checks passed. Exact hashes and artifact
provenance are in [final validation evidence](evidence/p014/final-validation.md)
and [machine-readable validation](ios-hub-monitoring-validation.json).

## Scope boundaries

`IOS-HUB-05`, `R-REMOTE-OS-TELEMETRY`, and the P014 data-model portion of
`R-IOS-FRONTEND-API` are verified by Hub-side processing and frontend contract
tests. `IOS-HUB-04` route and localization validation remains owned by P015; Gate
D therefore remains open. Native iOS Wi-Fi RSSI remains
**DEFERRED — REFERENCE DEVICE UNAVAILABLE**. P018 still owns physical iPad/A7
runtime checks. No production database, device, signing, entitlement, release,
tag, or deployment was changed.
