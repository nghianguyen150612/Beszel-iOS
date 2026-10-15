# P012 — iOS Wi-Fi RSSI compatibility

P012 is **BLOCKED for native RSSI acquisition**. The implemented iOS adapter
returns unavailable safely. This is a published source/fixture checkpoint, not a
claim that MobileWiFi works on the reference device. The PR remains Draft and
must not merge while the required native contract is unresolved. Gate C is
**PENDING**, not weakened to obtain a pass. P013 has not begun.

## Provenance and feasibility

Starting integration: `33840093bb465f0b164bb7d66aebc8cd85f86a4e`.
Production iOS: `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`.
Frozen main: `bb23f99b3eeaced9608e11e280e7c986dba650be`.
Branch: `ios/wifi-rssi-compatibility`.
[Draft PR #14](https://github.com/nghianguyen150612/Beszel-iOS/pull/14).
The exact baseline refs were fetched and matched; the existing clean checkout
was reused without discarding progress. All 178 upstream commits remain
ancestors and all 13 logical patches remain accounted for.

The [feasibility record](ios-wifi-provider-feasibility.md) pins historical Cykey
headers, iOS 12.4 SDK arm64 exports, Procursus netctl source and Apple API/CF
ownership documentation. It records candidate signatures, borrowed versus owned
objects, and specific evidence gaps. Historical MobileWiFi is the leading
candidate; it was not selected for execution. Apple80211 lacks a verified iOS 12
contract; CaptiveNetwork supplies metadata, not RSSI; NEHotspotNetwork's 0–1
strength is not dBm and fetchCurrent requires iOS 14+. CoreWLAN/JXA, nl80211,
quality, bars, speed and throughput are not substitutes.

The iOS 12.4 stub exports the six candidate manager/device/network functions
listed in the feasibility record. Exports do not establish their argument ABI,
object validation or daemon policy on iOS 12.5.7. Procursus' example releases a
dictionary before extracting its borrowed number and prints a scaled metric as
dBm; neither pattern was copied. No firmware-matched, bounded private acquisition
contract was established reliably enough to ship calls.

## Implementation and data contract

`agent/wifi/wifi_ios.go` uses the `ios` build constraint and returns nil.
The generic unsupported file excludes iOS; macOS continues selecting CoreWLAN
through osascript, Linux nl80211 and Windows native WLAN. Their platform collector
files are unchanged. The source validator requires exactly the iOS implementation
and rejects foreign or duplicate Wi-Fi collectors.

No framework is loaded, no symbols are resolved or invoked, and no CGO bridge,
private link dependency or entitlement is introduced. All framework/symbol,
manager/list, permission, association, RSSI and interface failure states therefore
have the same unavailable outcome without entering an unverified API. This does
not constitute tests of those APIs' actual errors. There are no CF objects or
retained pointers to release; no workers, native cache, retry loop or native
initialization. The adapter returns synchronously for active and canceled
contexts. The existing shared two-second deadline is preserved. It cannot cancel
synchronous C IPC; a future provider needs verified bounded calls or reviewed
process isolation and reaping before activation.

`Info.WiFi`, `Stats.WiFi`, interface-keyed maps, JSON fields and CBOR integer keys
are unchanged. No local interface name is invented. SSID remains optional, and
the existing UTF-8 sanitizer drops invalid octets without suppressing valid RSSI.
Missing/hidden/privacy-redacted SSIDs do not become BSSIDs. No MAC, password,
remembered-network inventory or nearby-network scan is emitted or requested.

The confirmed shared defect was unchecked int8 conversion of invalid signal
values. Positive/huge/non-finite inputs could become plausible history. `Signals`
now rejects NaN and values outside the collectors' existing -150..-1 dBm range
before conversion. This also rejects normalized 0..1 and negative fractions near
zero; range checking alone does not establish source units. Native providers
must still prove raw dBm. Valid history keeps the existing rounding/saturation
policy: -150..-129 becomes -128, never wraps; valid Info retains source precision.
No iOS RSSI number is emitted by this fallback.

The Agent's existing Wi-Fi block is extracted into a small method with a collector
argument for deterministic tests. Only default interval polls acquire data;
every result, including nil or empty, replaces the current snapshot. Realtime
requests reuse the latest snapshot and are not new measurements. Failure clears
old RSSI at the next default poll. SKIP_WIFI and its prefixed environment form
remain supported; no independent background polling is introduced.

## Entitlements and security

`com.apple.wifi.manager-access` is a candidate privilege supported by pinned
Procursus packaging evidence, which also includes broader entitlements. This
neither proves necessity/sufficiency nor a minimal read-only set on iOS 12.5.7.
Root and jailbreak status are not authorization evidence. No entitlement,
production signing, LaunchDaemon or ldid invocation was changed. P016 must review
any future minimal entitlement and its exact signing scope; P018 must validate
the denied and authorized CLI/LaunchDaemon cases. No wifid, kernel, configuration,
Wi-Fi power, join/disconnect or security-enforcement changes are permitted.

## Validation and acceptance boundaries

[Machine-readable evidence](ios-wifi-rssi-validation.json) records actual tested
SHAs, commands, results and hosted URLs. The [checkpoint ledger](ios-wifi-rssi-checkpoints.md)
records durable phases and remaining actions. Explicit-file Go tests execute the
actual iOS fallback on cloud hosts without selecting the host collector; these
are not hardware or simulated MobileWiFi permission tests.

Coverage includes repeated/concurrent unavailable acquisition, SKIP_WIFI,
finite/range/overflow/edge history validation, SSID UTF-8/CBOR safety, JSON/CBOR
snapshot roundtrips, default/realtime acquisition counts, lifecycle replacement,
legacy numeric-key transport fixtures for remote platform interface names and
Hub synthetic record disconnect/reconnect. The isolated Hub test database is
synthetic and temporary. Native CF parsing, symbol-failure injection, CF release
balances, real daemon permissions and actual station identity are **not tested**
because no private integration is shipped.

The required regression commands, real frozen Bun install/catalogs/tests/build,
Linux Agent/Hub builds, platform source selection, hosted iOS builds, A7 runtime,
Mach-O metadata, imports, checksums, provenance and embedded asset results are
reported in the JSON. A successful build validates this unavailable adapter,
not a native RSSI provider. Historical P011 runs and the iOS 0.19 device records
are not counted as new P012 validation.

P012 native acceptance: **BLOCKED** (`WIFI-ABI-12`, `WIFI-ASSOCIATION-12`,
`WIFI-AUTH-12`, `WIFI-BOUND-12`). Fallback/source/fixture checks can pass separately;
they cannot satisfy the missing required native acquisition. Canonical acceptance
allows owned absent/no-op feasibility deferrals, but this PR does not claim a new
Gate C PASS or reinterpret its acceptance cases. Gate C remains PENDING pending
resolution or explicit acceptance of this capability deferral. Gate D/F work and
physical acceptance are not completed by these cloud checks.

P018 / Gate F must verify actual 12.5.7 framework/symbols, ABI/object shapes,
least privilege, raw RSSI dBm against native observations, current interface
identity, disconnected/disabled and denied behavior, reconnection/SSID changes,
bounded latency/resource cleanup, no SIGILL and service stability on iPad mini 2
A1489 / iPad4,4 / Apple A7 / iOS 12.5.7 / Amethyst/Procursus. No device RSSI,
battery, memory or authorization result is claimed here.

Next concrete action: resolve the pinned firmware contract and bounded-call
blockers before replacing the adapter; assess minimal privilege in P016 and
physical acceptance in P018. Resume from the published branch and ledger.
P013 — Hub PocketBase migrations, database backup/restore and authentication /
security compatibility — is the next roadmap milestone and was not started.
There was no device/production database modification, release/tag, deployment,
or change to production iOS or frozen main.

## Completed source native-build evidence

Source SHA: `d3f33c83c8d6164717a26749826408359fed84ea`.
[iOS Build PASS](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38724163923),
[installer PASS](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38724168872).
macOS 14.8.9 arm64, Xcode 15.4 / iPhoneOS SDK 17.5, Go 1.27.1, Bun 1.4.0.
Agent/Hub repeated builds with isolated caches were byte-identical; the isolated
A7 runtime patch and instruction checks passed. Both are arm64 MH_EXECUTE,
iOS platform 2, minos 12.0, CGO=1, GOARM64=v8.0, with clean VCS provenance matching
that exact source. The real Hub embeds 95 assets. Downloaded artifact archive
hashes, Mach-O metadata, all payload SHA256SUMS, source provenance and local
production-frontend embedding were independently revalidated in the cloud.

| Payload | SHA256 |
| --- | --- |
| Agent | `993c82e71cf570ee490c731ee1ea2a6299b7e6573515b6b1790f707d962f7f6f` |
| Hub | `b3ea5f9327e5024ee84f2870088dc01a243ea011df522f1b4bf53b7054cb10b1` |

`otool -L` / `nm -u` confirm no MobileWiFi, CoreWLAN, Apple80211 or private WiFi
imports. Dependencies are CoreFoundation, Security, libresolv and libSystem.
The release job was skipped. Final documentation/workflow branch-filter changes
receive a new feature-head push run; the exact final commit and final CI URLs
are recorded in PR #14 to avoid claiming that an earlier run tested a later SHA.
These build results establish the unavailable adapter only. P012 native RSSI
remains BLOCKED and Gate C remains PENDING.

## P012A continuation

See [native blocker resolution](ios-wifi-native-blocker-resolution.md) and the
`p012a` evidence extension. Exact reference-firmware exports are now verified;
strict conversion, bounded inactive helper client and separate disabled native
diagnostic are implemented. These do not activate RSSI. Historical evidence above
is preserved at its original tested SHA. P012 BLOCKED / Gate C PENDING remains.
