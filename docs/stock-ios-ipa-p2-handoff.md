# Stock iOS P2 handoff

**P2: PARTIAL — RESUMABLE. PENDING PHYSICAL DEVICE VALIDATION.**

Starting feature HEAD: `a9a1749790e04cc73a85c1b0a5df34b8e86bb3d0`.
Branch: `feature/stock-ios-ipa`. PR #35 remains Draft against `iOS`.
No merge, release, deployment, production credentials or device access occurred.

## Implementation

`agent/embedded.go` adds explicit, instance-local configuration and a serialized,
context-cancelled lifecycle. It constructs the **same Agent**, registry and
WebSocket client used by the executable, bypassing default privileged collector
initialization. The existing `NewAgent`/signal-driven executable path and its
storage defaults are retained. The embedded path never starts SSH; `StartServer`
also rejects embedded Agents. It does not read TOKEN/HUB_URL or write proxy
configuration to the environment. Hub public key configuration is required in
addition to the token because the existing protocol verifies a signed challenge.

One runtime owns dialing/reconnect. Cancellation closes dialing/handshake sockets,
stops reconnect, closes the active WebSocket, and joins its reader. Concurrent
start/stop is serialized; replacement requires Stop. Each attempt clears Hub
verification. A random installation fingerprint persists in the app directory,
without probing protected hardware identifiers. Diagnostics contain no token.
The injected collector must return promptly; arbitrary blocking third-party
collector implementations have not been qualified.

`ios-app/BridgeCore/runtime.go` owns the application singleton. C exports in the
existing `GoBridge` provide Start, Stop, StatusJSON, UpdateMetrics and Free.
Configuration/sample C input is copied during each call. Returned strings are
caller-owned and freed with BeszelGoFree; nil means success for Start/Update.
Stop and all runtime operations run on a serial Swift worker queue. No Swift
pointers are retained by Go and no external executable is launched by this path.

The Swift controller adds URL, public key, secure token field, Save, Connect,
Disconnect, error details, actual Go/Beszel versions, device metadata, native
sample time and transport timestamps. Connected means the signed Hub challenge
was verified, not merely that a socket opened. It does not prove the Hub accepted
stats. LastMetrics remains unset when the backend cannot supply valid stats.

## Metrics and protocol limitation

No Hub schema, endpoint or database migration was changed. Transport uses
`/api/beszel/agent-connect`, X-Token, X-Beszel, CBOR request IDs, original
fingerprint verification and original response structures.
HTTPS/WSS is required in the application. HTTP/WS requires an explicit Go
insecure-development option used only by tests. TLS verification remains enabled.
Embedded diagnostics use generic network/TLS errors rather than echoing URLs,
credentials or server close reasons. Authentication failures have their own state.

| Measurement | Source / units | Native availability | Hub stats behavior |
| --- | --- | --- | --- |
| Model identifier | public uname machine | supported, fallback unavailable | configured Details; full stats blocked |
| Device name / OS | UIDevice | supported, OS may restrict name | configured Details; full stats blocked |
| Architecture | Go runtime | supported | configured Details; full stats blocked |
| Physical memory | ProcessInfo.physicalMemory, bytes | supported | not confused with used memory |
| Battery percentage | UIKit batteryLevel × 100 | optional; unknown -1 omitted; zero retained | local snapshot only |
| Battery state | UIKit batteryState | unknown/charging/full/unplugged | local snapshot only |
| Volume capacity | Foundation volumeTotalCapacity, bytes | optional | accessible app-container volume only |
| Available to app | Foundation important-usage capacity, bytes | optional, may include purgeable space | not whole-device filesystem usage |
| Process memory | not implemented | unavailable | never labeled system memory |
| System CPU / used memory / load | no qualified public collector | unavailable | **stats blocked** |
| Network throughput / disk I/O / processes / temperature | no qualified collector | unavailable | not fabricated |

**BLOCKED:** existing `system.Stats` and `system.Info` contain mandatory numeric
CPU, used-memory and percentage fields. Neither omission nor zero expresses
unavailable to the current Hub; decoding omission still yields numeric zero.
NaN/null are not a qualified alternative for Hub persistence and charts. Therefore
the stock collector deliberately returns an error. GetData uses the existing
`AgentResponse.Error` envelope and sends **no CombinedData**. Native real metrics
reach the Go singleton and diagnostics, but **do not yet reach an accepted Hub
stats response**. Completion criterion 4 remains unmet. No fake zero series is
sent. A successful synthetic stats test does not qualify real stock metrics.

## Credential and state storage

Token: Keychain generic password, non-synchronizable,
`kSecAttrAccessibleWhenUnlockedThisDeviceOnly`. Missing entries yield an empty
field; read/write failures are shown. Replacement uses SecItemUpdate; insertion
uses SecItemAdd. No token is put in UserDefaults, files, environment or diagnostics.
URL and public key use UserDefaults. Save persists changes; Connect stops the
previous runtime before applying a replacement. Changes do not silently mutate an
active session. Keychain items may survive uninstall; device restore does not
migrate ThisDeviceOnly credentials. A missing credential requires re-entry.

Agent state lives under Application Support/Agent in the app container and is
excluded from backup. There is no use of /var/lib, LaunchDaemons or system bins.
The legacy executable retains its original data-directory discovery.

## Lifecycle

Start is explicit. Inactive/background transitions enqueue Stop and release the
connection; return to foreground preserves configuration but requires Connect.
There are no background modes, private frameworks, indefinite tasks or daemon
claims. Forced termination, suspension timing, rapid scene transitions and actual
Keychain behavior require P3/device validation. No process-signal handler is
installed by the embedded runtime.

## Automated evidence

Local Linux Go 1.27.1:

- **PASS** `go test -tags testing ./agent/... ./ios-app/BridgeCore`.
- **PASS** `go vet -tags testing ./agent/... ./ios-app/BridgeCore`.
- **PASS** `go mod verify`.
- **PASS** `go test -race ./agent ./ios-app/BridgeCore`.
- **PASS** iOS patch sentinels: 44 passed, 0 failed.
- **PASS** upstream sync regression script: 10 passed, 0 failed.

Protocol fixture covers URL/security validation, rejected token, untrusted TLS,
signed authentication, request-ID CBOR stats response with synthetic data,
Hub disconnect/reconnect, stable persisted identity, unavailable stats errors,
shutdown during HTTP upgrade, repeated and concurrent start/stop and SSH rejection.
Adapter tests cover bytes vs GiB, unknown battery, valid zero, omitted values,
capacity scope/consistency, integer overflow, JSON and credential-free status.
These are Go adapter tests, not execution tests of UIKit/Foundation APIs.

Hosted native build and artifact evidence will be recorded below after completion.
Linux cannot compile SwiftUI or create an iPhoneOS binary. The isolated IPA
workflow adds shared regressions, vet, module and race tests before macOS builds.
The existing jailbreak workflow and release automation are unchanged. Source
regressions do not constitute A7/iOS 12.5.7 physical execution evidence.

## Remaining acceptance / exact next action

1. Resolve unavailable mandatory metrics with an approved protocol-compatible
   approach; any Hub schema/protocol change needs explicit approval. Until then,
   preserve the error response and mark P2 partial.
2. Qualify actual stock metrics through the real Hub stats persistence path after
   that solution exists; transport-only authentication is insufficient.
3. Add native Swift metric/Keychain/lifecycle execution tests, including unavailable
   APIs, restoration, replacement, process termination and rapid scene changes.
4. P3: qualify real suspension/termination/relaunch without background workarounds.
5. **DEFERRED — OWNER APPROVED:** physical iPhone 13 (iPhone14,5), A15, stock iOS
   27.0.1, SideStore. No installation or device connection is authorized now.

Future device checklist (prepare only): obtain fresh authorization; verify artifact
SHA; re-sign/provision for the device; verify Keychain locked/missing/replaced
credentials; connect to a disposable TLS Hub with its public key; compare native
battery to UIKit/UI and capacity scope; verify auth failure, TLS failure, network
loss, Stop, background/foreground, relaunch and stable identity; inspect actual Hub
records. Ad-hoc signing alone does **not** make this IPA installable on stock iOS.

## Actual Hub qualification

**PASS (local):** `go test -tags testing ./internal/hub -run
TestStockEmbeddedActualHub -count=1`. This test creates temporary PocketBase
state, a disposable user/system/token and the actual Hub SSH signer, then invokes
the unchanged registration handler over a local development HTTP server. It
verifies persisted fingerprint registration and a real Hub GetData request.
The stock backend returns the existing error envelope and LastMetrics remains
unset. **BLOCKED:** real stock metrics accepted/persisted by an actual Hub.
The local workspace already contained built site assets; clean CI must build the
real frontend before compiling the Hub package's embed directive.

## Intermediate Actions history

- **PASS** `45355468391`, code `14ba52d19019dd6eaa706bec0f3bf3f4a815d365`:
  shared regressions, Swift/iPhoneOS archive, ad-hoc signing and packaging.
  Artifact `14902651257`, ZIP digest
  `bc470cb24d174cfef2a7dad3829195c03051719b4f2be900d175ec278a28ec60`.
  This predates the stronger IPA verifier and stock CPU-init exclusion.
- **FAIL** `45355670886`: new verifier incorrectly rejected the required
  `Payload/` ZIP directory; native compile/signing succeeded. Fixed in `56583eb4`.
- **FAIL** `45356122875` / `45356173662`: clean CI lacked ignored site/dist assets
  needed to compile the actual Hub. Fixed in `837ba42f` by building the real
  frontend before integration tests, without changing Hub source or protocol.

## Changed file inventory

Shared Agent: `agent/agent.go`, `agent/client.go`, `agent/cpu.go`,
`agent/cpu_init.go`, `agent/embedded.go`, `agent/embedded_test.go`,
`agent/handlers.go`, `agent/server.go`.

Native host: `ios-app/App/AgentController.swift`, `ios-app/App/ContentView.swift`,
`ios-app/Bridge/BeszelGoBridge.h`, `ios-app/BridgeCore/runtime.go`,
`ios-app/BridgeCore/runtime_test.go`, `ios-app/GoBridge/main.go`,
`ios-app/Metrics/StockMetrics.swift`.

Validation: `internal/hub/stock_embedded_test.go`,
`.github/workflows/ios-ipa.yml`, `scripts/stock-ios/build-ipa.sh`,
`scripts/stock-ios/verify-ipa.py`.

Documentation: this handoff, `docs/stock-ios-ipa-p0-audit.md`,
`docs/stock-ios-ipa-p1-validation.md`, existing PR #35 description.

## Source commit history

```text
420b22fbe890bfacddec6a1fe743225f759f3b02 feat(agent): add cancellable embedded WebSocket runtime with explicit configuration
14ba52d19019dd6eaa706bec0f3bf3f4a815d365 feat(ios): wire foreground runtime, Keychain and native metric diagnostics
4de5d9663d4941430f59809532562b92eab01d56 test(ios): strengthen lifecycle concurrency and IPA provenance checks
2f1044e3343bcf37e9a9ce740142f805d7c07f01 test(ios): qualify actual Hub registration and isolate stock CPU initialization
56583eb4c655e69ae8717ff304f0abc11b38b6ff fix(ios): accept the required Payload directory in IPA verifier
837ba42f9a1b0fb7f9c5e38f6fa7b3fcbc7e8aab fix(ci): build required Hub assets before disposable integration test
93d88aee73ab0960014049dabead4db08f4aded3 fix(agent): send graceful close before cancelling established socket
```

**PASS:** Go file selection confirms cpu_init.go is included for legacy iOS
and excluded only for the stockios iOS build. This is source/build-selection
evidence, not A7 runtime execution. The final handoff commit is documentation only.

## Acceptance accounting

| Gate | Status / exact scope |
| --- | --- |
| Actual shared Agent embedded | PASS in initial P2 iPhoneOS build; final-source build recorded separately |
| Start/stop | PASS Go lifecycle tests and Swift compilation; native execution NOT RUN |
| Existing WebSocket/CBOR | PASS fixture and disposable actual Hub registration/request tests |
| Real stock metric in accepted Hub response | BLOCKED by required unavailable fields |
| No fabricated missing metrics | PASS; error envelope instead of partial CombinedData |
| Keychain credentials | PASS implementation/compilation; native storage execution NOT RUN |
| Foreground lifecycle | PASS implementation/compilation; actual suspension DEFERRED to P3 |
| Automated integration | PASS locally and final hosted Linux job |
| Native final-source build/artifact | See final qualification below |
| Jailbreak compatibility | PASS source/regression sentinels; A7 execution NOT RUN |
| Physical device | DEFERRED — OWNER APPROVED; PENDING PHYSICAL DEVICE VALIDATION |

Initial P2 artifact was independently downloaded through the GitHub connector
(after direct runner artifact download returned HTTP 403). Both ZIP digest and
inner IPA checksum were verified locally; corrected Mach-O/structure verifier
passed. Inner IPA SHA-256:
`11e5bdb12e544f9404a46371d68c8b7d2dbb99dbf3960055b12fa0b289cdce4d`.
This evidence belongs to `14ba52d1`, not the final source.

## Final-source hosted qualification

Code SHA: `93d88aee73ab0960014049dabead4db08f4aded3`.
Actions run: [45356510358](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/45356510358).

As of 2026-11-24 08:37 UTC: **PASS** Linux regression job including actual Hub
integration. **BLOCKED (runner availability)** final-source macOS job remains
queued, with no steps started. At that checkpoint, no artifact ID or checksum existed.
The previous build PASS is explicitly scoped to `14ba52d1` above. Superseded run
`45356317718` was cancelled to prioritize the final source.


## P2.1 checkpoint — 2026-11-24

**P2: PARTIAL — RESUMABLE. STOCK STATS ACCEPTANCE: BLOCKED.**
Physical-device validation remains **DEFERRED — OWNER APPROVED**.

The workspace initially had a clean `work` checkout at the `iOS` baseline
`0c3e4cd22ac24264834c401ad412e44aeda4e4f3`. The fetched feature branch
and Draft PR #35 both started at `2f73a45552f188c2e78a31b50480091501f60837`.
The P2.1 implementation and qualification source is
`3fb7b11a3ddf55e612f1b522b84e54121b8da246`. This handoff update is a separate
documentation-only commit. Branch `feature/stock-ios-ipa` and PR base `iOS` continue.

### Reconciliation of historical P2 run

[Run 45356510358](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/45356510358)
is pinned to `93d88aee73ab0960014049dabead4db08f4aded3`. Attempt 1
recorded Linux **PASS**. Its [macOS job 138316250019](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/45356510358/job/138316250019)
was cancelled at 08:45:32 UTC. The sole retry, attempt 2, reused the successful
Linux result. Its [macOS job 138364126009](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/45356510358/job/138364126009)
was cancelled at 10:16:39 UTC. Both macOS jobs show no runner, no job steps
and no build logs. Check-run annotations say the hosted Arm runner never
acquired the job after multiple attempts and cite capacity constraints.
Overall run conclusion is **failure** on both attempts. Artifact count is **0**.
Historical P1 and intermediate P2 artifacts remain evidence for their own SHAs.

### BUILD VERIFIED — P2.1 implementation SHA

[Run 45372067725, attempt 1](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/45372067725/attempts/1)
completed **PASS** at `3fb7b11a3ddf55e612f1b522b84e54121b8da246`.
Linux regression job `138368574956`, native job `138369283103`, and independent
artifact-consumer job `138372244473` all passed. P2.1 uses `macos-15-intel`
to use a separate hosted capacity pool. The build still targets iPhoneOS arm64.
The native log records Go 1.27.1, Xcode 16.4, iPhoneOS SDK 18.5, host Go checks,
Swift algorithm tests, injected Keychain tests, c-archive compilation, C probe
and SwiftUI linking, app signing, checksum verification and upload.

| Evidence | Result |
| --- | --- |
| Actions artifact | [14909806701](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/45372067725/artifacts/14909806701), development ZIP, 3,728,154 bytes |
| Outer ZIP SHA-256 | `1e1e21189fd174b4edd48403e30f0cc1e6d69c9d6ae6649bcd94950fda80e614` |
| Inner IPA SHA-256 | `dfa565795bb276d65b857c2254c9177d8639ce5377171ce8029aec8778638fb1` |
| Source and integrity | `SOURCE.sha` equals qualification SHA, ZIP integrity for both archives and attached IPA SHA256SUMS verified independently after downloading from the GitHub connector. Direct CLI artifact download returned HTTP 403 at the storage redirect |
| Bundle | `Payload/BeszelStockAgent.app`, `dev.nghianguyen.beszel.stockagent` 0.1.0 (1), iPhone family, iPhoneOS platform, Info.plist minimum 16.0, PrivacyInfo.xcprivacy present |
| Executable | Thin Mach-O arm64 MH_EXECUTE, LC_BUILD_VERSION iOS (2), minimum 16.0, SDK 18.5.0. Go build info shows `GOOS=ios`, `GOARCH=arm64`, Go 1.27.1 and `stockios`. Defined Go ABI, C CPU and C VM symbols verified independently |
| Signing | Hosted strict `codesign` verification passed. Ad-hoc CodeDirectory and resource seal present. CDHash `421ebbf908abe2085ed2056ea506a2c242a02ceb`. Team identifier unset, no embedded entitlements or provisioning profile |

### HUB PROTOCOL VERIFIED — existing Agent and disposable Hub

The actual Hub verifies the signed fingerprint exchange, stores the installation
fingerprint, sends CBOR GetData over WebSocket and decodes responses. Integration
tests cover invalid registration token, reconnect with the same fingerprint,
clean Stop and no inbound SSH listener. Protocol fixtures check request-ID echo.
A stock-style backend error produced **zero** `system_stats` rows and no
successful LastMetrics timestamp. The current Hub can close the connection and
mark the system down following that error.

A separate successful **synthetic** sample persisted 4 GiB total and battery
`[0,4]` in Stats, with `[0,4]` preserved in Info. The same database row contained
false zeros for CPU, memory used/percent/cache and root disk total/used/percent.
Info also stored zero CPU/memory/disk percent, bandwidth and uptime. A transport
decoder regression shows missing or null CBOR values can leave stale values in
the reused Go destination as well. No actual native stock observation is in
either Hub test. See [compatibility matrix and owner decision](stock-ios-ipa-metric-compatibility.md).

### METRIC ACCEPTANCE — BLOCKED

Public native probes now measure logical CPU counts, and attempt per-core host
ticks, aggregate interval utilization and Mach VM free/Darwin-available/
Darwin-used memory for local diagnostics. The snapshot also records physical
memory bytes, UIKit battery level/state, device identity and app-scoped volume
capacity. Each probe reports scope, unit, time and availability. The signed
stock sandbox has not been exercised. Whole-system root-disk usage, traffic,
I/O, load, swap, sensor and other host or privileged series are unqualified.
App-scoped volume estimates cannot stand in for root usage. The stock Agent
still sends the existing error envelope. No production Hub, protocol or schema
changes were made. See [public API semantics and probe tests](stock-ios-ipa-native-capabilities.md).

### Regression and device evidence

- **PASS** local and hosted tagged Agent and BridgeCore, Go module verification
  and vet, Agent and bridge race tests, and disposable Hub integration.
  Local Hub transport, WebSocket, systems, records and focused metric alert
  tests also passed. The broader alerts suite hit duplicate
  `GET /_/extensions.js` registration. `TestUserAlertsApi` reproduced the
  panic on the starting feature SHA. An extra race run of the real Hub fixture
  found existing Hub connection cleanup races, reproduced on the starting SHA
  at `WsConn.IsConnected` during socket close. These two broader failures stay
  open. The normal Agent/bridge race and disposable integration results are scoped
  accordingly.
- **PASS** Swift 6 portable sampling test on Linux and hosted macOS. Hosted
  macOS also passed injected Security tests, and compiled the real iPhoneOS
  Keychain and Mach code. A mirrored iPhoneOS SDK was used for a supplementary
  Linux cross-build of the C probe and Go c-archive.
- **PASS** jailbreak patch sentinels (44), upstream-sync checks (10), Linux
  default Agent and Hub CLI builds and legacy/stock CPU-init file selection.
  A private copy of Go 1.27.1 was patched with the existing A7 runtime script.
  After correcting the Linux cross-linker selection, it produced the legacy
  iPhoneOS arm64 Agent (minimum iOS 12.0). Disassembly confirms `YIELD` in
  `runtime.procyieldAsm`. The stock archive uses the unpatched runtime.
- **DEVICE EXECUTION NOT VALIDATED.** Physical iPhone 13 and A7 execution,
  stock sandbox Mach behavior, real Keychain operations and provisioning/
  installation remain unvalidated. An ad-hoc signature provides development
  bundle evidence only. No P3 lifecycle work, device access, SideStore operation
  or production use occurred.

### P2.1 changes and exact next action

Implementation SHA `3fb7b11a3ddf55e612f1b522b84e54121b8da246` covers
`ios-app/Metrics/{HostMetrics.c,HostMetrics.h,MetricSampling.swift,StockMetrics.swift}`,
`ios-app/App/{AgentController.swift,ContentView.swift,TokenStore.swift}`,
`ios-app/Bridge/BeszelGoBridge.h`, `ios-app/BridgeCore/{runtime.go,runtime_test.go}`,
`ios-app/Resources/PrivacyInfo.xcprivacy`,
`ios-app/Tests/{MetricSamplingTests.swift,TokenStoreTests.swift}`,
`internal/hub/stock_embedded_test.go`, the isolated IPA workflow and the two
`scripts/stock-ios/` build/verification scripts. This documentation-only
checkpoint adds the matrix and native capability notes alongside this handoff.

The next engineering decision is owner approval of the proposed negotiated
partial GetData contract. If approved, first implement a presence-aware
battery-only round trip against a disposable Hub. Check database keys, valid
zero, gaps, rollup denominators and alert transitions. Obtain authorization
separately for physical native measurements before claiming a stock metric
was accepted. If the extension is declined, retain local diagnosis and the
existing error while seeking a complete, scope-correct legacy snapshot.
