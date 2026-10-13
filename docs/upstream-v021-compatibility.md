# Upstream v0.21 compatibility matrix

The canonical scenario inventory is
[`upstream-v021-compatibility-matrix.csv`](upstream-v021-compatibility-matrix.csv).
It records combinations, behavior, protocol dependencies, fixtures, evidence,
status, owner, and blocking conditions. A `planned` row is a test obligation,
not evidence that the 0.21 iOS port works. P003 has not built the upgraded iOS
Agent or Hub.

## Evidence and status

The frozen upstream source is `bb23f99b3eeaced9608e11e280e7c986dba650be`;
P002's source audit is the file-level index. Current iOS production source is
the verified `origin/iOS` baseline `7e29c185458f79be274bb09ca57a0be9fc1b63f7`.
The recorded physical device evidence applies to the 0.19.0 iOS binaries only.

Status values mean:

* **historically validated** — P002/device records show this 0.19.0 behavior on
  the reference device; this is not a 0.21 result.
* **planned** — source evidence identifies the intended contract, but the
  required upgraded build or test has not run.
* **unsupported** — no iOS implementation is claimed; an absent reading or
  explicit no-op is required until its owner supplies an adapter.
* **blocked** — a prerequisite prevents a meaningful test.
* **verified** — reserved for a completed test against the stated version pair.

## Agent and Hub version contract

Upstream v0.21 declares `MinVersionNetworkMonitors = 0.20.0` in `beszel.go`.
That string is not a blanket Agent/Hub compatibility promise. The Hub's
`internal/hub/systems/network_monitors.go` checks the connected Agent version
before monitor sync and returns without sending monitor actions when the Agent
is older than 0.20.0. The regular stats path has separate compatibility checks:
`MinVersionAgentResponse = 0.13.0` and `MinVersionCbor = 0.12.0`. These affect
response/request handling and encoding, not network-monitor support. Unknown or
unparseable versions must be tested through the actual handshake and fallback
paths.

The required pair matrix covers iOS 0.19 Agent/Hub, iOS 0.19 Agent to upstream
0.21 Hub, upgraded iOS Agent to an existing 0.19 Hub, upgraded iOS Agent/Hub,
Linux/Windows/macOS 0.21 Agents to the iOS 0.21 Hub, and older remote Agents to
the upgraded iOS Hub. Until P007–P018 produce binaries and run their tests,
every upgraded-binary combination remains planned.

## Protocol and networking contracts

The CSV tracks WebSocket handshake and authentication headers, SSH fallback,
reconnect after connection or handshake failure, keepalive and stale-peer
cutoff, connection ownership and locking, cancellation, deadlines, version
negotiation, monitor resynchronization, CBOR, JSON v2, and absent-versus-zero
fields. `agent/client.go` validates `HUB_URL`, reads one token from `TOKEN` or
`TOKEN_FILE`, parses `HUB_HEADERS` as header pairs, ignores caller-supplied
`X-Token` and `X-Beszel`, and appends `CA_CERT_FILE` roots to the system pool.
TLS verification remains enabled by default. The per-monitor
`SkipTLSVerify` option is separate and must be explicitly opted into and tested
as an exception for that monitor only.

The `omitzero` changes in upstream entities are part of the JSON v2 contract.
Tests must compare JSON v1/v2 and CBOR behavior for absent values, explicit zero,
false, and empty data. A fresh response struct must replace cached data so an
omitted field cannot leak a previous nonzero value.

## iOS platform constraints

P007 must keep `GOOS=ios`, `GOARCH=arm64`, `CGO_ENABLED=1`, the iPhoneOS SDK,
and the iOS 12.0 deployment metadata. The A7 workaround patches the known Go
ARM64 `runtime.procyieldAsm` shape and rejects unknown shapes; source inspection
or a successful link alone cannot establish A7 runtime safety.

P008 owns iOS system metadata and CPU/RAM/disk/battery/network-counter
validation. The existing iOS battery parser uses iOS 12's
`AppleARMPMUCharger`/`ioreg -l` output and maps attachment, charging, full, and
empty state. It must not be replaced by macOS `AppleSmartBattery -a` behavior.
Historical 0.19 evidence exists for the reference iPad; 0.21 collection remains
planned until rebuilt and exercised.

P010 must exclude or provide a supported no-op/adapter for Linux/macOS/Windows
collectors unavailable on iOS, including GPU, SMART, ZFS, systemd, and
platform-specific network-monitor probes. Missing readings must remain absent;
unsupported values must not be represented as fabricated zeros. P011 owns a
read-only Procursus/APT package-update adapter with a bounded timeout and no
package-list refresh or install action. P012 owns Wi-Fi RSSI fallback behavior.
The upstream macOS Wi-Fi collector invokes `/usr/bin/osascript` with JXA and
CoreWLAN. That is not an iOS implementation; iOS must be explicitly excluded
from that build path until a native adapter is validated.

The iOS Hub is also a Hub for supported remote Linux, Windows, and macOS Agents.
Local iOS collector exclusions must not remove Hub parsing, storage, or display
of supported remote telemetry.

## Required tests and release gates

Each row links to acceptance cases in
[`upstream-v021-acceptance.md`](upstream-v021-acceptance.md), or to P017's
protocol/Hub/Agent test fixtures. Migration-related rows remain blocked from a
production upgrade until the disposable 0.19-to-0.21 migration and restore path
passes the rollback gate. Physical A7 runtime and service validation belongs to
P018. No P003 row asserts a new build or device result.
