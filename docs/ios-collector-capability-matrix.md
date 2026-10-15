# Local iOS Agent capability matrix — P012

This matrix covers the local Agent compiled with `GOOS=ios GOARCH=arm64
CGO_ENABLED=1`. Build selection and portable fixtures do not establish physical
runtime support. The iOS Hub retains the shared parsing/storage paths for remote
Linux, Windows and macOS Agents. No protocol keys or migrations are removed.

| Capability | Local iOS policy / selected path | Data contract / remote behavior |
| --- | --- | --- |
| TCP, DNS, HTTP/HTTPS | Portable Go probes; bounded contexts; secure per-monitor HTTP client | Measured microseconds; failed checks contribute loss. Physical DNS/trust/network behavior remains P018. |
| ICMP IPv4 / IPv6 | Independent raw → datagram → validated executable selection; no assumed privileges | Unavailable methods return errors/loss, never inferred RTT. All four real cloud socket modes unavailable; physical iOS remains P018. |
| GPU usage/power/temperature | Constructor returns `errors.ErrUnsupported` before capability discovery; `gpu_darwin_unsupported.go` | No local GPU map/reading; remote GPU fields and macOS powermetrics/macmon remain supported. |
| SMART / NVMe | Constructor returns unsupported before smartctl discovery/device scan | Empty SMART response using the existing handler contract; no disk health fabricated. Shared ATA/SCSI/NVMe parsing and Hub storage remain. |
| ZFS | Manager has no iOS backend; `zfs_ios.go` rejects direct utility acquisition | No pools/datasets/capacities. Empty complete detail snapshot under existing schema. Linux/FreeBSD/macOS utility behavior and remote Hub records preserved. |
| Btrfs | No iOS backend; `btrfs_unsupported.go` | No synthesized filesystem or pool; Linux kernel implementation retained. |
| systemd | `systemd_nonlinux.go` | Nil local service data, explicit unavailable detail/log errors; remote Linux service/log storage retained. |
| Docker / containers | No automatic local daemon discovery; explicitly configured `DOCKER_HOST` retains portable, timeout-bounded API client | Absent local daemon gives no container reading; remote Hub containers and configured remote Docker API remain supported. |
| Temperatures | `sensors_ios.go` returns no sensors; excludes macOS SMC/IOKit acquisition | Optional temperature map absent; dashboard scalar default is not a measured sensor value. Remote temperature values retained. |
| Fans | `fans_other.go` supplies empty hwmon root; no sysfs walk | Optional fan map absent; remote Linux RPM data retained. |
| eMMC / mdraid health | `emmc_stub.go`, `mdraid_stub.go` | No sysfs/device scans or health readings; Linux paths and remote health parsing preserved. |
| CPU / memory cgroups | `cpu_unsupported.go`, `memory_unsupported.go`; Linux utils excluded | Standard P008 CPU/Mach memory remains; no cgroup reading. |
| Linux network/sysfs/procfs | `network_counters_unsupported.go`; Linux collector files excluded | P008 Darwin interface counters remain; no Linux counter substitution. |
| Filesystem usage / physical disk I/O | P008 `ios_disk.go` actual mount/statfs usage; physical I/O unavailable | Optional physical I/O absent; no inferred disk throughput or device. Linux storage paths retained. |
| Battery / metadata | P008 ioreg/Mach/sysctl adapters and their safe absence/cache rules retained | Optional battery absent on failure; source fixtures are not current hardware measurements. |
| Package updates | P011 iOS-only Procursus/APT adapter; rootful and rootless binary candidates, APT-config-derived database/index paths, allowlisted simulation only | Nil means unknown/unavailable; [0] means verified zero. Procursus security classification is unknown. Cached metadata may be stale; no refresh/download/mutation. Physical iPad validation remains P018. Linux and remote Hub package telemetry remain supported. |
| Wi-Fi RSSI | `wifi_ios.go`: explicit unavailable adapter; MobileWiFi acquisition DEFERRED | Nil snapshot / RSSI map; no private calls or guessed zero. ABI, fresh association, authorization and bounded-call blockers: [feasibility](ios-wifi-provider-feasibility.md). P012 follow-up; P016 privilege packaging; P018 hardware. Remote Linux/macOS/Windows paths retained. |

The actual Go dependency graphs are checked for ios/darwin/linux ARM64 with
CGO=1 by `.github/scripts/check-ios-source-selection.py`. Negative fixtures
reject a foreign temperature, macOS GPU/Wi-Fi, Linux cgroup, utility-based iOS
ZFS implementation, duplicate family or missing real frontend assets.
Gopsutil's Darwin sensor files can remain transitively linked by shared portable
sensor types; the local iOS acquisition hook does not call them.

Unsupported optional maps/slices stay nil or empty according to their existing
JSON/CBOR contract. Upstream required scalars may retain zero defaults; those
zeros must not be represented as measured temperatures, GPU load or health.
No capability is disabled globally in the Hub based on the local Agent's OS.
