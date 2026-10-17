# P014 final validation evidence

## Source and preservation

- Starting P013 integration: `00c14bbf2756213cc2afef828ef7d5b9af26f1d6`.
- Feature branch: `ios/hub-monitor-alert-rebuild`; active PR #17 targets
  `ios/upstream-v021-integration`.
- P014 backend/source candidate: `3f153dca940651ab394663b9d85d14cbde64b957`.
- Latest source checkpoint before evidence documentation: `bf9c4d4b3c8910ab4e2546fbbf5d4e75c93864b6`. `ca7108b7` formats touched TypeScript/TSX; `bf9c4d4b` only extends the readiness validator to accept verified risk dispositions and adds its regression test. No Go, Agent, Hub, entity, transport, database, or alert behavior changed. Bun tests/build and Python audits were rerun, and the generated frontend was checked against the actual embedded frontend in the native Hub artifact.
- PR #16 (`ios/hub-monitor-alert-compatibility`) is closed and unmerged. Its
  commits are not ancestors of PR #17 and were not cherry-picked.
- Frozen upstream ancestry: 178/178; logical iOS patches: 13/13. Production `iOS`
  and frozen `main` remained at their protected SHAs during validation.

## Hub and platform fixtures

Linux, Windows, macOS and unavailable-iOS payloads are independently authored,
source-derived synthetic fixtures. Actual Hub WebSocket/SSH decode, request,
polling, PocketBase persistence, history retrieval, realtime/API serialization,
monitor processing, alert evaluation, reconnect and authorization paths ran in
disposable synthetic tests. No native Windows, macOS, or iPad Agent execution is
claimed. `TestP014AuthenticatedRemoteTelemetryAPI` verifies per-system identity,
authorized reads and outsider denial. P014 fixtures do not replace P015's browser
route checks.

The matrix/readiness validators passed with 178 upstream commits, 35 historical
iOS commits, 13 logical patches, 24 acceptance cases and 33 compatibility
scenarios. Python audit tests: 55 passed. Shell suites: upstream sync 10, patch
sentinels 45, bootstrap 54, installer lifecycle 814; all passed.

## Go and races

Go 1.27.1 canonical command passed on backend source `3f153dca`:

```text
env -u NETWORK go test -tags 'testing no_ui' ./... -count=1 -timeout=20m
PASS — all packages; 27 packages with tests reported ok, remaining packages compiled.
```

Full affected race suites passed in bounded groups:

- Agent, Agent Wi-Fi, system entities, Hub transport, Hub WebSocket.
- Hub and Hub systems.
- Records and alerts.

The slowest race group was alerts at 672.772 seconds, within its 15-minute timeout.
Repeated race runs passed: the P014 Hub API/monitor ordering/cancelled poll/SSH
platform tests three times (Hub 40.079s; systems 153.509s); P014 record and alert
regressions three times (records 14.440s; alerts 51.844s); P014 byte/GPU entity
codecs three times (1.018s); and `TestAgentWebSocketIntegration` three times
(15.720s).

The P009 test-only race was reproduced against unchanged P013 at the unsynchronized
`ConnectionManager.State` read. P014 tests now use the existing locked reader and
cancel/join fixture Agents. The repeated candidate race test passed; production
WebSocket state/authentication code did not require a race fix. Reproduction details
remain in `concurrency-reproductions.txt`.

## Frontend and portable builds

Bun 1.4.0 frozen dependency installation passed; catalog generation/compilation,
TypeScript project build, and production Vite build passed. The suite passed 55
of 55 tests with 267 assertions across 9 files. The build emitted 95 assets and
kept `bun.lock` unchanged. Generated `.po` edits were restored; no localization
work is claimed.

At source candidate `ca7108b7`, Linux Agent and Hub builds passed. Their SHA-256
values are recorded in `ios-hub-monitoring-validation.json`. Windows Hub systems
and FreeBSD transport test binaries compiled successfully; these are compile-only
results, not native runtime execution. Source selection passed for iOS, Darwin,
Linux, Windows, and FreeBSD targets.

## Hosted native iOS evidence

The iOS Build workflow passed on source
`3f153dca940651ab394663b9d85d14cbde64b957`:

- [iOS Build run 38941248597](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38941248597): Procursus package checks and native build succeeded; `release-ios` was skipped.
- [Installer lifecycle run 38941249967](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38941249967): passed.
- [P013 Hub migration fixture run 38941249970](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38941249970): passed.

The iOS build used Go 1.27.1, Bun 1.4.0, macOS 14 and the iPhoneOS SDK,
`GOOS=ios GOARCH=arm64 CGO_ENABLED=1 GOARM64=v8.0`, with minimum iOS 12.0. The
isolated patched GOROOT preserved its original source. Both unstripped binaries
passed Apple A7 runtime instruction checks and are arm64 Mach-O, `MH_EXECUTE`,
`LC_BUILD_VERSION`, iOS platform 2, minimum 12.0, VCS-clean at `3f153dca`:

- Agent: 9,542,984 bytes; SHA-256
  `72b0d235e6b2d682a0613c656139a05f4015d9cb1bc3f83f4c96a94252a1c8fc`.
- Hub: 31,977,208 bytes; SHA-256
  `8bd41ef8cefcf91135221dbe696da9525afaf3fe4f42dca6ab66ece2b21ff7a0`.
- Embedded frontend: 95 files. The independently downloaded native Hub payload
  matched the current frontend build with `verify-ios-artifacts.py`.
- `SHA256SUMS` verified for both binaries. No production MobileWiFi hard link
  appears in the native linkage report. The disabled diagnostic is a separate
  artifact, not packaged into the production payload.

GitHub artifact archive SHA-256 digests:

| Artifact | SHA-256 | Size |
| --- | --- | ---: |
| `beszel-ios-arm64` | `9c3cd14381a8ca31d0cbc88fb4479e60b3590f442b85bd7a2b7aee0fc6ae5bd0` | 16,284,566 bytes |
| `ios-native-validation` | `ca429ae8eab0870deccaa5cb8484b7c7a4a7e397d69d0060575bf0a4d6d38a92` | 7,816 bytes |
| `ios-wifi-diagnostic-disabled` | `cc6055444127ae13a9309778543d410ae384a193e664623cba53a7d3aff4edb7` | 6,051 bytes |

The later commits `ca7108b7` and `bf9c4d4b` changed only frontend formatting
and readiness-validation documentation tooling; neither changes Agent/Hub build
inputs. The Bun build passed after formatting, and the 95-file output matched the
downloaded, hosted native Hub artifact byte-for-byte through the repository's
artifact verifier. The validator-only commit then passed all 55 Python audit
tests. The native binary payload remains applicable; the hosted workflow is
accurately recorded against `3f153dca`, not described as running on later commits.

## Acceptance boundaries

`IOS-HUB-05`, `R-REMOTE-OS-TELEMETRY`, and the P014 frontend/API data-contract
scope are verified. `IOS-HUB-04` browser routes, embedded route smoke, localization,
and dependency/advisory work remain for P015. P018 still owns physical iPad/A7
runtime validation. Native iOS Wi-Fi RSSI remains
**DEFERRED — REFERENCE DEVICE UNAVAILABLE**. No production database/device,
release, tag, signing, entitlement, or deployment operation occurred.
