# Stock iOS IPA — P0 audit and P1 design checkpoint

## Scope and baseline

- Base: `iOS` commit `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`.
- Base `beszel.Version`: `0.19.0`. No version upgrade is part of this track.
- Feature branch: `feature/stock-ios-ipa`, independent of the stacked P030 Draft PR.
- Existing jailbroken Apple A7/iOS 12 binary, installer, and LaunchDaemon workflows are untouched.
- Initial target: stock iPhone 13 (A15), iOS 27.0.1; no jailbreak or exploit assumption.
- Physical-device testing and distribution/release are **not authorized** by this task.

## Source and dependency map

| Source | Reuse | Blocker/adapter needed |
| --- | --- | --- |
| `beszel.go` | Shared version and protocol thresholds | None for a basic Go bridge |
| `agent/agent.go` | Actual shared `Agent` type and metric pipeline | `NewAgent` initializes collectors before runtime policy is injectable; no stock startup yet |
| `agent/client.go` | WebSocket/CBOR client, Hub request/response handling | `HUB_URL` / `TOKEN` from environment, TLS and token handling; inject Keychain-backed configuration |
| `agent/connection_manager.go` | Reconnection state machine concepts | Signal-driven, health-file updates, SSH fallback, no foreground lifecycle `Stop`; replace with app-owned cancellation |
| `agent/server.go` | Existing request handlers/response encoding | Inbound SSH listener is forbidden for the stock target |
| `agent/battery/battery_ios.go` | Keep existing jailbreak collector | Runs `/usr/sbin/ioreg`; stock must use UIKit battery state |
| `agent/system_platform_ios.go` | Keep jailbreak system details | `sysctl` and system plist are not guaranteed within a sandbox; use supported APIs and explicitly mark unavailable fields |
| `agent/data_dir.go` | Shared preference for persistence | Current fallback to `/var/lib` / home is unsuitable; inject an app container URL |
| `.github/workflows/ios-build.yml` | Existing jailbreak binary pipeline | Must remain unchanged |
| `ios-app/GoBridge` | New Go C ABI c-archive wrapper | P1 bridge only; does not start `agent.NewAgent` |
| `ios-app/App` | New SwiftUI foreground host | P2 Keychain, Hub settings, real connection and lifecycle diagnostics remain pending |

### Existing outbound Hub transport

The shared `agent/client.go` uses `HUB_URL` to construct the WebSocket
endpoint `/api/beszel/agent-connect`, with `X-Token` and `X-Beszel` headers.
It reuses the Beszel Hub CBOR request/response protocol. The stock target
should adapt this client rather than create a second protocol implementation.
The Hub and its database schema must not be changed for this target.

### Stock iOS metrics policy (P2)

Use public, sandbox-compatible APIs for device identity (`UIDevice`),
battery percent/state (UIKit monitoring), physical memory (`ProcessInfo`),
app-process memory (current-process Mach task APIs) and app-container storage
capacity (`FileManager`). Distinguish total device memory from process memory,
and do not mislabel container-available space as universal disk free space.
CPU and network metrics need validated sources before they can be published.
Unavailable host-level metrics must not be represented as zero/healthy values.
No arbitrary `exec`, system plist, LaunchDaemon, privileged sysctl or background
execution assumption is allowed.

### Architectural stages

1. **P1 (this branch):** Go `c-archive` with exported functions linked into
   a real SwiftUI iPhoneOS app. Version and Go target are read from real native
   Go code. An ad-hoc-signed development IPA is packaged for later re-signing.
2. **P2:** split injectable runtime/storage/collector backends, use the shared
   Agent's outbound WebSocket/CBOR client, and provide a Keychain-backed config.
3. **P3:** app-owned lifecycle state machine with foreground-only collection,
   clean suspension/reconnect behavior, no background-mode abuse.
4. **P4:** strengthen native CI/provenance; add proper SideStore resigning
   instructions without storing Apple signing credentials in public workflows.
5. **P5:** integration/metric tests, optional simulator, physical-device
   acceptance **DEFERRED** until expressly authorized.

## P1 acceptance boundaries

`ios-app/BridgeCore` derives Beszel version directly from this repository;
`ios-app/GoBridge` exports it from a real iPhoneOS Go c-archive.
`ContentView` calls Go symbols; it displays **not started** for Agent,
**not configured** for Hub and **not collected** for metrics. It is not a mock
Hub or an Agent implementation.

`scripts/stock-ios/build-ipa.sh` packages an ad-hoc-signed development IPA.
An ad-hoc signature does **not** make an IPA installable on a stock iPhone.
SideStore must apply a valid device-specific signature and provisioning.
Building or re-signing on CI does **not** prove execution on an iPhone.

The target deployment minimum in this P1 SwiftUI host is iOS 16.0, independent
of the jailbreak binary's iOS 12.0 target. No A7 runtime workaround is applied
to the new A15-targeted stock application by default.

## Validation evidence required

- P0 source mapping: completed by inspection.
- Bridge source and CI scaffolding: implemented; build result must be read from
  the exact feature-source Actions run.
- Host-only Go bridge test: automated in `ios-ipa.yml`; actual result pending.
- Swift compile, Go c-archive link, IPA metadata, Mach-O and ad-hoc signature:
  only PASS after the hosted macOS workflow completes.
- SideStore re-sign and actual stock iPhone launch: **NOT RUN / DEFERRED**.
- Agent↔Hub connectivity, real stock metrics, Keychain and suspension: **P2/P3,
  NOT IMPLEMENTED**.
