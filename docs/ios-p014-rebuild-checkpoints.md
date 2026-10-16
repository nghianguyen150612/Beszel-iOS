# P014 rebuild checkpoint ledger

PR #16 is CLOSED / UNMERGED and abandoned. No commits from that attempt are
imported. New work starts at P013 `00c14bbf2756213cc2afef828ef7d5b9af26f1d6`
on `ios/hub-monitor-alert-rebuild`, targeting `ios/upstream-v021-integration`.

Initial verification (2026-10-16 UTC): both refs match that baseline; old HEAD
`05b249043a1a7a3cb4da2184ad2cfafdabd85876` is not an ancestor; all 178 frozen
upstream SHAs are ancestors; matrix/readiness validators account for 13 logical
iOS patches. Production iOS and frozen main match their protected SHAs.

## A — fresh fixtures

Independently authored JSON fixtures in `internal/testutil/telemetry/testdata`
cover Linux, Windows, macOS and an iOS Agent with unavailable collectors. IDs
and owners are synthetic and distinct. The shared Go loader feeds later Hub
integration tests; schema tests verify JSON v1 and CBOR round trips, absent iOS
collectors, measured zero CPU/GPU/battery/sensors and platform metadata.

These are protocol fixtures, not native Windows/macOS/iOS execution. ZFS on
macOS and GPU/sensors on desktop platforms require their actual collector tools.
SMART and package details use separate request actions and will be exercised
separately from periodic telemetry. Btrfs uses the extra-filesystem contract.

P014 gate: IN PROGRESS. Gate D remains pending P014 and P015.
Native RSSI: DEFERRED — REFERENCE DEVICE UNAVAILABLE. No physical-device,
production database, release, deployment, signing or entitlement changes.

Checkpoint A focused validation: `go test -tags 'testing no_ui'
./internal/testutil/telemetry` PASS with Go 1.27.1. JSON v1 and CBOR schemas pass.
New baseline regressions independently reproduced JSON v2 numeric per-core
array rejection and JSON v1 historical battery byte-string rejection. Repairs
belong to Checkpoint B. Existing Bun suite: 44 PASS / 0 FAIL.

New Draft PR: https://github.com/nghianguyen150612/Beszel-iOS/pull/17
Checkpoint A remote commit: `9e11e665` (push verified).

## B — decoding repair

Reproduced on the unchanged P013 model: JSON v2 rejects numeric `cpus` arrays;
JSON v1 rejects historical base64 `bat` strings. Added shared strict byte-metric
JSON decoding for numeric arrays and legacy byte strings. Battery tuples require
exactly two bytes; invalid/overflowing input is rejected before publication.
CBOR tags/encoding and current numeric JSON output are unchanged.

Focused entity and platform-fixture suites PASS after repair (Go 1.27.1).
Authenticated Hub ingestion/API tests are in progress; B is not yet complete.
