# Remote Agent telemetry fixtures — P014

Starting integration: `00c14bbf2756213cc2afef828ef7d5b9af26f1d6`.
Fixtures: `internal/testutil/telemetry/fixtures.go`, enabled only by `testing`.
Every snapshot is **source-derived synthetic**, not a captured payload or an
executed Windows, macOS or iOS Agent. Deterministic data is serialized through
the real CBOR entities. Each platform has a distinct system and owner identity.
Fixture tests assert schema, units, absence, independent maps and compact size.

| Capability | Linux fixture | Windows fixture | macOS fixture | Local iOS fixture |
| --- | --- | --- | --- | --- |
| CPU, per-core/breakdown, memory, filesystem usage, uptime | present | present | present | present (P008 contract) |
| Network throughput / interface totals | eth0 | Ethernet | en0 | en0 |
| Physical disk I/O | bytes/s and cumulative | bytes/s and cumulative | bytes/s and cumulative | absent |
| Load average | present | default tuple; unavailable on Windows | present | default tuple in minimal fixture |
| Additional filesystems | data, Btrfs-shaped data | data | data | usage only |
| Battery | absent in this server fixture | absent in this desktop fixture | present | source-derived ioreg mapping |
| Temperatures | coretemp and GPU | absent | GPU | absent |
| Fans | RPM | absent | absent | absent |
| GPU | two devices, including idle | idle NVML-shaped device | macmon/powermetrics-shaped device | absent |
| Wi-Fi | nl80211 dBm / wlan0 | WLAN native dBm / Wi-Fi | CoreWLAN dBm / en0 | absent; approved P012 deferral |
| Containers | present | absent | absent | absent |
| systemd | active service, journal capability | absent | absent | absent |
| ZFS | tank identity, capacity, I/O, health | absent | absent in fixture | absent |
| Package updates | APT [2,1], details | absent | absent | Procursus APT [0], security unknown |
| Network monitors | five protocol scenarios | same Hub contract | same Hub contract | same Hub contract |

Collector support is conditional on actual hardware/tools; the matrix describes
representative fixture variants, not universal availability. SMART uses its
separate request and `smart_devices` contract, covered by existing Hub tests.
Btrfs uses shared filesystem/pool fields; no invented Btrfs wire key is added.
GPU temperature is carried by `Stats.Temperatures`, not `GPUData` JSON.

Source references at the starting SHA:

- `agent/system.go`, `agent/system_platform_ios.go`, `agent/ios_{metadata,memory,disk}.go`: core values, OS enumeration and iOS metadata override.
- `agent/wifi/wifi_{linux,windows,darwin,ios}.go`: native dBm versus unavailable local iOS.
- `agent/gpu_nvml.go`, `agent/gpu_darwin.go`, `agent/gpu_test.go`: conditional remote GPU identity, memory, utilization and temperature.
- `agent/storage_pool.go`, `agent/btrfs`, `agent/zfs`: filesystem/pool units and backend selection.
- `agent/systemd.go`, `agent/systemd_nonlinux.go`: Linux services and unavailable non-Linux details.
- `agent/package_updates*.go`, `internal/entities/system/package_updates.go`: known zero, unavailable counts, unknown security and cached metadata.
- `internal/entities/{system,container,monitor}`: exact CBOR integer keys and JSON API representation.

P014 checkpoints and test outcomes are recorded in the monitoring validation
manifest. P015 retains full frontend routes/assets/localization/advisory
acceptance. P018 retains physical device validation. No native RSSI acquisition
or signing/entitlement change is authorized here.
