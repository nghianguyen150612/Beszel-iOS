# P008 — iOS system metrics and battery compatibility

**P008 source/fixture/native-build gate: PASS** at
`a096d1eabe11e41667a726470f502e21a6b93650`. Final documentation-inclusive
HEAD verification and integration merge are recorded in [PR #10](https://github.com/nghianguyen150612/Beszel-iOS/pull/10).

P008 validates source, synthetic fixtures, shared regressions and native builds.
It does not claim Beszel v0.21 runs correctly on a physical iPad. Gate C remains
pending P009–P012 and device-dependent acceptance; P018 / Gate F owns hardware
execution. No P009 implementation is included.

## Starting state and provenance

The mandatory fresh clone selected the case-sensitive `iOS` branch. Starting
integration: `2b5d8fa31e6b6da25a3073c5148b3375301f40dd`; product iOS:
`0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; frozen main/upstream:
`bb23f99b3eeaced9608e11e280e7c986dba650be`. All refs matched before edits.
The frozen target and every canonical upstream SHA remain reachable: **178/178**.
All **13 logical iOS patches** remain accounted for in the JSON preservation
record. Work uses `ios/ios-system-metrics`, targeting only
`ios/upstream-v021-integration`, with ordinary commits and a normal merge.

The starting source, P006 preservation record, acceptance/compatibility plans,
risk register/crosswalk, P007 runtime/native reports and historical build/device
notes were inspected. Initial source copies and baseline test logs were captured
outside the repository. The baseline canonical Go suite and real Bun build/
tests passed. Detailed tested SHAs, commands, hashes and workflow conclusions
are in [the evidence JSON](ios-system-metrics-validation.json); final-head
verification is also recorded in the PR because a commit cannot contain its own
SHA. EN and VI reports describe the same implementation and acceptance limits.

Historical reference only: iPad mini 2, A1489 / iPad4,4, Apple A7, arm64,
iOS 12.5.7, Amethyst / Procursus. The 0.19 record remains unchanged. Its battery
42/100, raw capacity 3989 and memory observations inform sanitized fixtures;
no device identifier, memory total or OS version is hardcoded for all devices.

## Confirmed defects and bounded changes

| Path | Starting behavior / confirmed defect | Production behavior |
| --- | --- | --- |
| CPU | Missing initial baseline could calculate lifetime usage; empty source could return nil error; individual resets and NaN were not rejected; global baseline maps lacked a shared lock. | Explicit unavailable baseline/error result; finite monotonic counters and identity checks; mutex around runtime acquisition; reset/recovery and per-core length checks. Existing default-interval seeding remains. |
| RAM | gopsutil Darwin VM acquisition dereferences an unchecked `vm_kernel_page_size` lookup and subtracts 32-bit page counters. The generic missing-cache fallback derives a cache value from other memory categories. | iOS-only native Mach adapter, OS page size and `hw.memsize`; widened/saturating arithmetic; retain free+inactive availability; no inferred cache/swap. Linux/macOS acquisition and Linux cgroups remain. |
| Disk | Physical root I/O could be guessed from the busiest macOS device; unsigned byte/op subtraction could wrap; an absent/failed device could retain ExtraFs throughput. | iOS usage identities come from actual partitions/mountpoints, default `/`; physical I/O explicitly unavailable. Shared rates avoid underflow/overflow, disappearances break all interval baselines, and fresh ExtraFs snapshots do not alias mutable filesystem state. |
| Network | A fixture with zero receive or zero transmit was incorrectly excluded. Nonempty startup selection hid later interfaces. Reset treated the new cumulative value as interval traffic; multiplication could overflow; anomaly totals disagreed with per-interface rates. | One-way traffic is eligible; reconcile identities every poll; active addressed zero-counter interfaces or explicit NICS inclusion establish baselines; rollback produces no rate, anomaly clears aggregate and interface rates together. |
| Battery | Unbounded subprocess; capacity multiplication could overflow; missing booleans implied discharge; multiline nested keys and incomplete/mixed blocks could be accepted. | Same AppleARMPMUCharger source, fixed deadline/output budget, shared cache, exact portable parser, complete direct-property blocks, correct class, safe percentage arithmetic, unknown states and absent failed data. |
| Metadata | Every `iPad4,*` identifier mapped to A7, including unknown identifiers; inherited metadata could survive missing iOS sources. | Exact supported A7 IDs; unknown SoCs stay generic; injectable sysctl/plist sources clear unavailable values and validate counts. |

The unchecked VM symbol and macOS disk-driver assumptions are confirmed source
risks, not observed v0.21 iPad crashes or proof that no iOS driver can expose
counters. P008 intentionally avoids those assumptions; a future physical I/O
adapter would need independent source/device evidence. The CPU non-Linux file
was **not** a standard-CPU disablement defect: it disables cgroups only.

## CPU, RAM and filesystem semantics

CPU retains gopsutil v4.26.9's Darwin `host_statistics(HOST_CPU_LOAD_INFO)` and
`host_processor_info(PROCESSOR_CPU_LOAD_INFO)` through libSystem. User/system/
nice/idle ticks are converted to seconds; ratios use counter deltas, independent
of the tick frequency and cache interval. iowait/steal are zero because this
source has no such categories; no alternative measurements are invented.
Totals and breakdowns stay finite and within 0–100. A missing/unchanged/invalid
sample omits optional breakdown/per-core data; the required upstream CPU scalar
retains its existing zero default. Per-core vectors are withheld after a length
or identity change and, on iOS, when they contradict known logical-thread count.
They are never synthesized. Source errors break continuity; later valid samples
establish a baseline and recover. The `runtime.GOOS != "ios"` model-probe guard
remains, and `/proc/cpuinfo` fallback is Linux-only.

RAM reads `hw.memsize`, `host_statistics(HOST_VM_INFO)` and the OS page size via
an iOS-only CGO boundary. Its acquired host port is deallocated each call.
Available = free + inactive, bounded by physical total; used = total − available.
Page counters widen before arithmetic; inconsistent/transient counts saturate
within total. Active/inactive/wired counters do not become a Linux cache value.
`MEM_CALC=htop` does not reinterpret this iOS source. Separate cache/buffer and
swap measurements are **unavailable**: optional swap fields are omitted, and the
required upstream `MemBuffCache` scalar keeps its zero default. That zero does
not prove the device has no cache. Missing physical memory produces the existing
required-scalar defaults, not a copied previous sample. Linux
`calculateHostMemoryUsage`, `saturatingSub`, cgroups and ZFS ARC regressions pass.

Filesystem discovery uses Darwin getfsstat at initialization and statfs usage
at polling, with no recursive traversal. Default `/` means the root filesystem,
including read-only roots; it is not a claim about total physical handset
storage. `/private/var` is selected only through actual partition/configuration
information. `FILESYSTEM` accepts mountpoint/device identity and custom name;
`EXTRA_FILESYSTEMS` explicitly selects extras. Same mount/device is deduplicated;
related mounts are not automatically added or summed. Separate APFS volume
reports can share container capacity and must not be interpreted as additive
physical capacity. Missing devices use the mountpoint as an internal usage key,
never an invented `/dev` name. Inaccessible/zero-capacity samples stay safe and
later polling retries; root is refreshed each poll, extras retain the existing
configured usage-cache duration. **Physical disk I/O is unavailable on iOS**:
throughput, cumulative counters and time statistics are omitted using existing
optional keys. No Linux diskstats, SMART or ZFS adapter is introduced. Shared
Linux ZFS/Btrfs/pool discovery and 32-bit time-counter wrap behavior remain.

## Network, battery and metadata semantics

Darwin interface byte counters come from gopsutil's routing/sysctl source.
Rates are bytes/second per interface, summed to the upstream bandwidth tuple;
raw totals retain their original meaning. Each cache interval has its own
baseline. First samples, removed/reappearing interfaces, resets, source errors,
zero elapsed time and future timestamps produce no invented rate. Multiplication
and aggregation cannot wrap; existing anomaly thresholds remain. Default
loopback/docker/bridge/veth/bond/cali filters remain. Active unfamiliar VPN,
cellular or AWDL interfaces follow the existing policy; they are not rejected
because of their name. NICS exact/wildcard whitelist overrides automatic filters,
including zero counters; blacklist applies alongside automatic filters. For an
untracked interface with both counters zero, UP plus an address establishes its
eligibility; a down/unaddressed interface waits, and later one-way traffic can
make it eligible. Discovery failure never invents an interface. No throughput
counter is treated as Wi-Fi RSSI; native RSSI remains P012.

Battery acquisition remains `/usr/sbin/ioreg -r -c AppleARMPMUCharger -l`, without
macOS AppleSmartBattery or plist-mode substitution. `CommandContext` limits a
call to **2 seconds**; Wait reaps the killed direct subprocess, and WaitDelay
bounds inherited pipe waits to **250 ms**. Stdout is capped at **1 MiB**, stderr
is discarded, and raw system output is not logged. The **60-second** cache
covers both success and failure, avoids expensive realtime subprocesses and
returns copies. Failures clear readable battery data; the next eligible poll
can recover. No persistent timer, background retry or shell is introduced.

The production entry calls the same portable acquisition/parser tested on host.
Recursive and legacy pipe-prefixed complete registry blocks work; nested
BatteryData/child keys, wrong registry class, duplicate keys, multiple battery
blocks, truncation, negative/malformed/overflow capacities and zero/missing
maximum are rejected. Percentage uses overflow-safe integer arithmetic and is
bounded 0–100. Missing BatteryInstalled retains the prior assumption; explicit
No or malformed installed state rejects the battery. Optional raw maximum only
supplies full-charge capacity when valid. The upstream state enum is unchanged:
unknown 0, empty 1, full 2, charging 3, discharging 4, idle 5. Known disconnected
or charging state takes precedence; full/empty/idle require adequate source
information. Missing/invalid charging fields do not imply charging/discharging.
`Primary`, `Battery` and `Batteries` payload conventions remain unchanged.

Metadata preserves `hw.machine`, `hw.physicalcpu`, `hw.logicalcpu`,
`kern.hostname`, `kern.osrelease`, SystemVersion.plist/ProductVersion and
`runtime.GOARCH`. Os remains upstream `system.Darwin` (numeric 1); OsName is
`iOS` or `iOS <ProductVersion>`, never a macOS label. Exact A7 mapping covers
`iPad4,1`–`iPad4,9` and `iPhone6,1`/`iPhone6,2`; unknown IDs get
`Apple SoC (<id>)`. Missing optional values stay absent, invalid/absurd counts
are zero, cores do not exceed known threads, and malformed/missing plist files
leave the plain iOS label. A non-arm64 fixture checks that architecture is read
from the selected runtime source rather than hardcoded.

## Tests, protocol and performance limits

Fixtures are synthetic. Battery shape/provenance is recorded in
`agent/battery/testdata/ios/README.md`; historical numeric references come from
STEP 39 of the unchanged 0.19 record. CPU, pages, partitions, byte counters,
metadata and errors are injected into production boundaries, not copied logic.

| New family | Top-level tests | Named subtests |
| --- | ---: | ---: |
| CPU | 2 | 10 |
| Memory | 2 | 6 |
| Disk | 3 | 9 |
| Network | 2 | 0 (sequential lifecycle/rate scenarios) |
| Battery | 4 | 37 |
| Metadata | 2 | 22 |
| Shared rate arithmetic | 1 | 0 |
| Payload presence | 1 | 3 |

The subprocess helper is infrastructure, excluded from the four battery test
families. Coverage includes first/steady/idle/saturated/reset/invalid CPU;
source failure/recovery and core changes; normal/low/zero/malformed/huge memory,
missing cache/swap and repeated polls; root-only/related/explicit/missing/
read-only filesystems, unavailable I/O, device disappearance, time/reset/error
recovery; dynamic/idle/one-way/VPN/explicit/excluded network interfaces, multiple
intervals and overflow; battery malformed/missing/nested/state/deadline/output
limit/error/cache/concurrency/recovery; known/unknown/missing metadata and plist.

No shared entity layout, CBOR key, JSON tag, dependency or transport production
code changes. JSON v1/v2 and CBOR tests cover required zero scalars, legitimate
empty battery, present metrics, absent swap/disk/RSSI, one-way interfaces and
per-core arrays. Existing cache intervals and Agent lock remain. CPU baselines
have an additional shared lock; fresh ExtraFs payloads copy mutable counters.
The macOS hosted full Agent regression exposed an overly long Unix-socket path
in an existing SSH fixture; a shorter disposable directory repairs the fixture
without weakening its listener/payload assertions or changing transport code.

Bounded subprocess tests and shared race checks establish host logic/cleanup.
They do not measure Apple A7 CPU/RAM overhead, native tick correctness, Mach
permissions or actual iOS collector latency. There are no overhead estimates.

## Executed validation and acceptance

Exact local results, source candidate SHA, hosted URLs, native checksums and
artifact conclusions are recorded in the JSON and PR. Audit/readiness, 49 audit tests (35 P007), sync 10, sentinels 45, bootstrap 54,
installer 814, canonical Go suite, shared race tests, genuine Bun build/44 tests
and Linux Agent/Hub builds pass. All 95 real frontend assets are embedded.
Hosted macOS Agent/race/GPU tests and Agent/Hub builds pass. Native Agent/Hub
builds, target/runtime/frontend/checksum/repeat-build gates pass. Native builds retain
Go 1.27.1, Bun 1.4.0, isolated A7-patched GOROOT, CGO/iPhoneOS SDK, ARM64 v8.0,
iOS platform 2/minimum 12.0, repeated independent-cache comparisons, runtime
instruction inspection, actual frontend embedding and SHA256SUMS verification.
The final documentation-inclusive feature HEAD is rerun; release job is skipped.

| Acceptance | P008 evidence | Remaining obligation |
| --- | --- | --- |
| IOS-AGENT-01 | Mapping, sysctl/plist error fixtures and native linkage | P018 device sysctl/plist versus payload comparison |
| IOS-AGENT-02 | CPU/RAM/disk/network synthetic/error/recovery tests; builds | P018 actual kernel accounting, mount semantics and Wi-Fi/VPN counters; physical disk I/O explicitly unavailable |
| IOS-AGENT-03 | Exact battery parser/acquisition/cache/state/deadline tests | P018 real output and charging/discharging/idle/full observations |
| Gate C | P008 source/test/build portion | P009–P012 plus applicable physical acceptance |
| Gate F | No new device result | P018 physical A7 execution and acceptance |

Unavailable fields and device-only questions remain explicit; neither macOS
execution nor an iOS link proves iOS kernel behavior. The acceptance companion
below separates those categories. No production database access, device command,
signing, install, reboot, service change, release/tag creation or deployment was
performed. Product `iOS` and frozen `main` remain unchanged.


Source-candidate native CI: [38526577598](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38526577598).
Installer: [38526580705](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38526580705).
Both succeeded on the source SHA above; release was skipped. Xcode 15.4
(15F31d), iPhoneOS SDK 17.5. Downloaded payloads were independently validated:
Agent SHA-256 `2a63b88ffc6b28bd3ca72ef2650e7d06df03e81efd8bee274841ad87ed8f8082`;
Hub `0cc91d9b516158cbc81ee50c71d99c8f860ccab74d785fbf165521b33b428b7f`.
Both are ARM64 MH_EXECUTE, platform 2/minimum 12.0, Go 1.27.1, CGO=1,
clean VCS revision matching the candidate. Final-head hashes differ with VCS
metadata and are recorded in the PR's final review. Two failed macOS fixture
runs and one superseded cancellation remain recorded in JSON; none is pass
evidence. Direct CLI artifact retrieval returned HTTP 403; the connected
GitHub artifact tool retrieved both archives, whose upload ZIP digests and
payload checksums matched. No validation gate was bypassed.
