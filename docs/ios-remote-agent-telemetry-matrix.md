# P014 remote Agent fixture matrix

All fixtures are synthetic, independently authored for this rebuild, with distinct
system/owner IDs. JSON v1 and CBOR schema round trips PASS. Each platform is sent
through the same shared Hub code. No native Windows/macOS execution is claimed.

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

Fixture wire files: `internal/testutil/telemetry/testdata/*.json`. The iOS fixture
intentionally omits optional battery/package counts too; this describes unavailable
collection, not removal of P008/P011 capabilities. Fields unsupported by a platform
are absent. Wi-Fi uses native dBm; no signal-quality conversion is permitted.

Required scalar zero defaults in the upstream schema do not establish an optional
measurement. GPU presence is established by its map; battery presence by the
representative state/device map; successful zero package count by a present array.
The unchanged scalar GPU power fields omit zero on the wire; no availability
claim can be inferred from an omitted power field. The frontend must leave a gap.
