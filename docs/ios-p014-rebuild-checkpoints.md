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

## F — GPU display contract (partial)

The power chart now keys series by device identity and name rather than name
alone. Missing device/power fields produce gaps, measured zero remains zero, and
package power stays in watts. Focused Bun tests: 2 PASS, 0 FAIL. Broader API
snapshots, frontend contracts and final build are still pending.

Fresh Python audit suite: 54 PASS. Upstream sync, patch sentinels and bootstrap
suites PASS. Installer lifecycle remains in progress. Initial Hub/race dependency
compilation is still in progress; these runs are not yet passing evidence.

## C — optional history repair

Independently reproduced on unchanged P013 rollup source: sparse temperatures,
fans, filesystems, GPUs/engines, CPU arrays and network interfaces were diluted
by absent observations; GPU package power was dropped; a null filesystem panicked.
The original failing transcript is `docs/evidence/p014/baseline-json-rollup-failures.txt`.

Rollups now use observation counts for each optional device/array entry, preserve
zero readings, retain GPU package power in watts at existing precision, and skip
null filesystems. A changed GPU name at the same identity begins a new aggregate.
The existing per-pool ZFS and per-interface Wi-Fi policies remain intact.

`go test -tags 'testing no_ui' ./internal/records -run '^TestP014'` PASS in the
combined focused run: slice regressions plus real PocketBase ten-minute rollup,
close/reopen of the same disposable DB, valid zero, package power, and no new
system history after disconnection. Broader regression/retention checks remain
pending. No database schema/migration or retention interval changed.

## B/D — authenticated ingestion and monitor ordering

Fresh focused tests PASS for all four OS payloads through authenticated WebSocket
(signature/token/fingerprint binding), scheduled polling, PocketBase writes,
authorized API/history reads, denied outsider access, reconnect with partial
telemetry, OS details and revoked membership. Source-derived API goldens are in
`internal/hub/testdata/p014` and omit nondeterministic IDs/timestamps.

Authenticated, host-key-pinned SSH tests reproduce generic-only response rejection
for every platform at both 0.19 and 0.21. The periodic SSH path now uses the shared
response decoder rather than discarding a valid generic response and reading EOF.
Typed framing still passes; legacy raw framing remains version-gated.

Reproduced stale monitor overwrite (new sequence 5/value 25 overwritten by
sequence 4/value 90). Accepted probe sequences now gate both live updates and
history/alerts; a newer sequence remains valid after wall-clock rollback. Task
configuration changes and SSH reconnect reset sequence state. Complete polls and
configuration requests are serialized. Immediate results recheck current ownership,
enabled state and exact configuration under a transaction; they share history
ordering and cannot recreate a deleted monitor. A focused immediate-lifetime
regression has been added for the final canonical/race runs.

Combined `go test -tags 'testing no_ui' ./internal/records ./internal/hub
./internal/hub/systems ./internal/alerts -run '^TestP014' -count=1 -timeout=10m`:
all four packages PASS after independently reproduced defects were fixed.
The first fixture setup failure (missing fingerprint row) and an overly specific
403/404 denial assertion were test issues, not relaxed production authorization.

## E — optional alert recovery

Fresh regressions reproduced immediate GPU recovery when its telemetry vanished
and failure to recover at a genuine zero-degree temperature. GPU alerts now hold
when optional data is absent; a present idle GPU still evaluates zero. Temperature
presence is determined from sensor data, so zero is eligible for recovery. Positive
legacy dashboard readings remain accepted. Invalid/null extra filesystem entries
are ignored during instantaneous disk evaluation.

Actual alert-manager tests PASS for trigger, unchanged incident, unavailable data,
and fresh zero recovery. Test mailers are synthetic; no real notifications are
sent. Full supported alert, authorization and notification-queue regressions are
running in the canonical and race suites; no complete P014 acceptance is claimed.

## F — complete shared payload checks

Authenticated optional-action fixtures now exercise actual SMART and ZFS refresh
requests, Linux package inventory, container history and systemd rows with owner
API reads and outsider denial. Unsupported collectors return errors and leave
empty collections. Synthetic per-platform action JSON files accompany the API
goldens. All four-platform authenticated API cases PASS.

Bun frozen install, genuine production build and 52 frontend tests PASS, including
source-derived API snapshots, measured zero CPU/per-core/GPU/battery/sensors,
package inventory, container history, absent iOS collectors, missing monitor
history and loss-only/null-latency windows. GPU usage/VRAM/engine charts now also
leave gaps for missing devices, and malformed null Wi-Fi entries are ignored.
No visual redesign or P015 route/localization/dependency acceptance is included.

Platform source selection PASS for iOS/macOS/Linux and all five Wi-Fi targets.
Full Go run initially passed Agent, Hub, systems, transport, WS, migrations and
users, but failed an old rollup assertion expecting absent samples to count as
zero, and the alert suite reached its default 10-minute timeout during active
fixture setup. This is NOT a pass. The assertion is being corrected and the full
suite will be rerun with a realistic timeout after the newly reproduced sparse
alert-window issue is repaired.

## E follow-up — sparse alert windows

A correctly dated historical fixture independently reproduced false temperature
and extra-filesystem recovery: valid readings 90 and 30 average 60, but an absent
sample incorrectly diluted them to 40. Per-sensor/filesystem counts now exclude
missing observations and require sufficient observations for that device. The
focused alert and record suites PASS, including unchanged incident state at the
correct mean. The fixture asserts stored timestamps; PocketBase's automatic
creation field requires explicit SQL dating for these synthetic historical rows.
The DB restart/rollup fixture was strengthened to date its rows the same way.

The old mixed-optional rollup test now expects the available sample rather than
half its value. No threshold, retention interval, notification authorization or
database migration changed.

## C follow-up — individual power-field presence

Reproduced an omitted power field being treated as 0W while its GPU still reported
usage: available 8W and explicit 0W should average 4W; the old result was 2.67W.
GPU power and package power now have independent observation counts. JSON/CBOR
decoding retains explicit legacy zero presence; null/omitted fields are unavailable.
Current Agent struct omission rules and numeric CBOR tags are unchanged. Explicit
zero survives JSON v1/v2, CBOR and persisted rollups. Null/empty GPU device payloads
are rejected rather than fabricated as idle devices.

Focused entity/fixture/record tests PASS, including legacy zero round trips,
current zero-package-power omission, zero/absent distinction and the corrected
power averages. Final canonical and affected race checks remain required.

## G — reproduced race/fixture findings (in progress)

The reported P009 race was independently reproduced: the Hub integration test
read ConnectionManager.State while handleStateChange wrote it under the mutex.
A testing-only accessor now uses the existing production reader lock. Repeated
checks also exposed a premature fingerprint assertion and unowned test Agents;
those fixtures now wait for committed fingerprint state and cancel/join Agents.
Production WebSocket authentication/state algorithms are unchanged.

The broad race attempt also reproduced a P010 test hang: a 30ms context could
expire before TCP accept, leaving its fixture waiting forever on Accept. The
fixture now cancels only after an explicit accept barrier and bounds cleanup.
That test PASSES three times under -race. The expired broad run is NOT a pass.

A second race showed the Hub startup producer adding child updaters while test
cleanup waited on an empty WaitGroup. Startup registration is now tracked before
it spawns children, and test cleanup cancels producers before joining. The short
SSH handshake-deadline fixture also needs a realistic race-build setup budget;
it still verifies that established sessions outlive the original deadline.
Repeated Hub/SSH checks and final source validation remain in progress.

G focused repeated repair checks: PASS in Agent (3.040s), Hub (164.714s) and SSH
transport (19.194s), three repetitions each of applicable certificate cancellation,
Agent WS identity, universal-token isolation, API authorization/startup cleanup,
and handshake-deadline tests. These are actual completions, not expired runs.
Read/write race excerpts and original log hashes are preserved in
`docs/evidence/p014/concurrency-reproductions.txt`. The broad pre-final attempt
also expired on slow Hub/systems fixtures and retained stale rollup assertions;
remaining superseded work was stopped. It is not final acceptance evidence.

## G/B follow-up — retired polling response

A fresh real-WebSocket fault test reproduced a successful persistence result after
its owning system context was cancelled while telemetry was in flight. Live
polls now check their owner before fetching, after receipt and inside the write
transaction. A replaced connection cannot commit that late snapshot. Resume polls
are registered with the same updater group so cleanup joins them too. Explicit
manual test-record seeding remains available independently of live poll contexts.

`go test -race -tags 'testing no_ui' ./internal/hub/systems -run
'^TestP014CancelledPollDoesNotPersistLateResponse$|TestRemoveAllSystemsWaitsForUpdater'
-count=3 -timeout=5m`: PASS (67.685s), including late response rejection and DB
lifetime through cleanup. No request framing, auth algorithm or retention policy
changed. Final frozen-source canonical/race/native validation remains pending.
