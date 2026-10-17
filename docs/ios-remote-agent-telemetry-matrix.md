# P014 remote Agent fixture matrix

All four fixture payloads are synthetic, independently authored from repository
platform source, and use distinct system/owner IDs. Linux, Windows, macOS and iOS
fixture schemas pass JSON v1 and CBOR checks. No native Windows/macOS/iOS Agent
execution is claimed.

| Surface | Linux | Windows | macOS | iOS unavailable fixture |
| --- | --- | --- | --- | --- |
| CPU/per-core, memory, filesystem, interfaces, uptime | Present | Present | Present | Present |
| Disk physical I/O | Present | Present | Present | Absent |
| Battery, GPU, temperatures, Wi-Fi | Present | Present | Present with tooling | Absent |
| Fans, systemd | Present | Absent | Absent | Absent |
| ZFS | Present with tooling | Absent | Present with tooling | Absent |
| Btrfs | Extra filesystem | Absent | Absent | Absent |
| Containers | Present | Present | Absent (remote daemon requires explicit config) | Absent |
| Package counts | Successful zero | Absent | Absent | Absent (no successful optional collection) |
| SMART / package inventory / ZFS details | Separate action, existing suites | Separate action where supported | Tool-dependent separate action | Unavailable adapters |

Fixture wire files: `internal/testutil/telemetry/testdata/*.json`. Hub integration
uses the same synthetic values in fake WebSocket/SSH peers; the shared production
Hub path performs decoding, polling, PocketBase writes, history retrieval, monitor
and alert processing, realtime publication, and authorized API reads. These are
Hub-side executions, not native Agent executions. The iOS fixture intentionally
omits optional battery/package counts too; this describes unavailable collection,
not removal of P008/P011 capabilities.

Wi-Fi uses native dBm. Hidden/redacted SSID and missing RSSI remain absent. Local
iOS RSSI remains unavailable under the approved P012 deferral. No signal-quality
conversion is permitted.

Required scalar zero defaults in the upstream schema do not establish an optional
measurement. GPU presence is established by its map; battery presence by the
representative state/device map; successful zero package count by a present array.
The unchanged scalar GPU power fields omit zero on the wire; decoded legacy explicit
zero is retained, while omitted power remains unavailable.

The source-derived protocol matrix covers raw legacy CBOR at 0.12 and typed
responses for 0.19/0.21 where supported. It does not claim every field is available
on every arbitrary version pair. Protocol and Hub-side evidence is detailed in
`docs/evidence/p014/final-validation.md` and
`docs/ios-hub-monitoring-validation.json`.
