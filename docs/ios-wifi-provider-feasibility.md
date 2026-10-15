# P012 native Wi-Fi provider feasibility

Decision: **DEFERRED — MobileWiFi native acquisition; retain unavailable fallback**.
This record does not establish an RSSI-capable iOS implementation. Starting
integration: `33840093bb465f0b164bb7d66aebc8cd85f86a4e`. Research performed
2026-10-15 in the cloud; the reference iPad was not accessed.

## Evidence and candidates

| Candidate | Pinned / primary evidence | Version evidence | Output / units | Decision |
| --- | --- | --- | --- | --- |
| MobileWiFi | [Cykey headers](https://github.com/Cykey/ios-reversed-headers/tree/c613e45f3ee5ad9f85ec7d43906cf69ee812ec6a/MobileWiFi), [iOS 12.4 SDK stub](https://github.com/theos/sdks/blob/0222fd5413cf4b9af096f37b4621afa2688572f7/iPhoneOS12.4.sdk/System/Library/PrivateFrameworks/MobileWiFi.framework/MobileWiFi.tbd), [Procursus netctl](https://github.com/ProcursusTeam/netctl/tree/3db04194c08fe93b723bc3da82cca80ae0d7ffa7) | Historical 2013–2014 declarations, reused by 2022 Procursus source; stub lists iOS arm64 exports. No 12.5.7 binary/runtime verification. | Historical `RSSI` CFDictionary containing integer `RSSI_CTL_AGR`; candidate raw dBm. Separate scaled RSSI must never be substituted. | Defer calls until firmware contract, fresh association, authorization and blocking behavior are established. |
| Apple80211 | No verified iOS 12.5.7 command-line ABI / read-only station contract obtained. macOS interfaces do not establish iOS support. | Unverified | Unverified | Reject speculative ABI / ioctl use. |
| CaptiveNetwork | [Apple Wi-Fi information entitlement](https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.developer.networking.wifi-info) | Access restrictions apply to iOS 12; current-network metadata only | SSID metadata; no measured RSSI | Cannot satisfy RSSI. Do not add an entitlement solely for metadata. |
| NEHotspotNetwork | [signalStrength](https://developer.apple.com/documentation/networkextension/nehotspotnetwork/signalstrength), [fetchCurrent](https://developer.apple.com/documentation/networkextension/nehotspotnetwork/fetchcurrent(completionhandler:)) | `fetchCurrent` is iOS 14+, unsuitable for the iOS 12 CLI target | Normalized 0–1 signal; not measured dBm | Reject approximate conversion and entitlement assumptions. |
| CoreWLAN, osascript, nl80211, iwconfig, bars, link speed, throughput | Existing platform sources / API scope | macOS / Linux or wrong metric | Not an iOS native dBm provider | Excluded. |

## Historical candidate ABI (not compiled or invoked)

The pinned `WiFiManager.h`, `WiFiDeviceClient.h`, and `WiFiNetwork.h` declare:

```c
WiFiManagerRef WiFiManagerClientCreate(CFAllocatorRef allocator, int flags);
CFArrayRef WiFiManagerClientCopyDevices(WiFiManagerRef manager);
CFPropertyListRef WiFiDeviceClientCopyProperty(WiFiDeviceClientRef client, CFStringRef property);
WiFiNetworkRef WiFiDeviceClientCopyCurrentNetwork(WiFiDeviceClientRef client);
CFStringRef WiFiDeviceClientGetInterfaceName(WiFiDeviceClientRef client);
CFStringRef WiFiNetworkGetSSID(WiFiNetworkRef network);
```

The three WiFi references are opaque struct pointers. These signatures use C
calling conventions, CF reference arguments/returns, and a C `int` flags value;
no opaque struct layout should be accessed. The pinned iOS 12.4 arm64 stub lists
all six symbols at `/System/Library/PrivateFrameworks/MobileWiFi.framework/MobileWiFi`.
An export list has no argument types and does not prove the 12.5.7 ABI or daemon
policy. The public iPhoneOS SDK does not supply this private contract.

[Apple CF ownership rules](https://developer.apple.com/library/archive/documentation/CoreFoundation/Conceptual/CFMemoryMgmt/Concepts/Ownership.html)
require owned Create/Copy returns to be released; Get values and array/dictionary
elements are borrowed. A future bridge must keep the owner alive while reading
a borrowed value, validate every CF type, handle NULL and partial cleanup, bound
arrays/strings, and verify opaque object type IDs before private calls. The
historical headers do not declare the corresponding private type-ID functions;
the SDK stub's symbol names alone are insufficient function-signature evidence.

The [Procursus RSSI example](https://github.com/ProcursusTeam/netctl/blob/3db04194c08fe93b723bc3da82cca80ae0d7ffa7/wifi/wifi-info.m)
releases the dictionary before extracting its borrowed number, extracts `raw`
but prints the scaled `strength` as dBm, and uses transfer ownership for Get
values. Its [device enumeration](https://github.com/ProcursusTeam/netctl/blob/3db04194c08fe93b723bc3da82cca80ae0d7ffa7/wifi/wifi.m)
selects index zero without checking an empty array or station association. These
are source findings, not executed crashes. They cannot be a safe implementation
or establish raw units/freshness on the reference firmware.

## Privileges, observation and blocking

[Procursus entitlements](https://github.com/ProcursusTeam/netctl/blob/3db04194c08fe93b723bc3da82cca80ae0d7ffa7/entitlements.plist)
include `com.apple.wifi.manager-access` along with numerous broader privileges.
This is evidence of a candidate manager-access requirement, not proof of the
minimal entitlement set for read-only access on iOS 12.5.7. Root or Amethyst must
not be assumed to grant it. No entitlement or signing configuration is changed.
P016 owns any later evidence-backed `ldid -S<plist>` packaging decision; P018
must compare denied access and the minimal authorized command-line LaunchDaemon
case. Broad netctl privileges, keychain access, daemon patches and security
bypasses are not acceptable prerequisites.

Only manager creation, current device/network reads, interface identity and raw
RSSI reads are candidate operations. A future bridge must load the fixed system
path with guarded `dlopen`/`dlsym`, resolve all verified symbols before calls,
avoid a hard optional-framework dependency, and release resources on every exit.
It must never enumerate remembered networks, scan, join, disconnect, change power
or preferences, or emit BSSID/password data.

No bounded-return or cancellation contract was found for synchronous private
manager/property calls. A Go two-second context does not interrupt a blocked C
call. Spawning a goroutine per timed-out poll is unsafe. Before activation,
provide a verifiable bounded native operation or a reviewed, bounded process
isolation design with shutdown/reaping tests. Even dynamic symbol lookup cannot
validate a calling convention or authorize IPC.

## Blockers and ownership

- **WIFI-ABI-12**: verify firmware-matched declarations/type validation and raw
  `RSSI_CTL_AGR` units; SDK exports plus old examples do not suffice. P012 follow-up.
- **WIFI-ASSOCIATION-12**: establish fresh station/power state, exact OS interface
  identity, and disconnection/reconnection semantics across all discovered devices;
  current-network objects may contain cached metadata. P012 follow-up / P018.
- **WIFI-AUTH-12**: establish least privilege and denied behavior on the reference
  firmware; source compilation cannot prove daemon authorization. P016 / P018.
- **WIFI-BOUND-12**: establish bounded acquisition and cleanup when wifid does not
  reply; do not claim cancellation of synchronous native IPC. P012 follow-up.

Selected architecture is an explicit `ios` unavailable adapter: nil snapshot,
no native loading, no allocation/worker/cache, no system mutation. A future
verified provider can replace that adapter while preserving `Collect`, polling,
interface keys and JSON/CBOR contracts. The native capability remains deferred;
this record does not waive P012 acceptance or declare Gate C passed.

## Download integrity

SHA256 of inspected upstream bytes (not shipped SDK files):

| File | SHA256 |
| --- | --- |
| Cykey MobileWiFi.h | `6d1fc8427d75b7e9413042b23624ef879a176d695d3d9982a6b9b830ee144885` |
| Cykey WiFiDeviceClient.h | `ee33e3699db2bb564b9627429baa3af8402f7804f94942938e6f9524a0bdcac6` |
| Cykey WiFiManager.h | `b4549c720b5766307a0888ba0c736ad2d940334ad101a101c996e127c31b803d` |
| Cykey WiFiNetwork.h | `68c963636c1529258761ec81b7e6a64dbb02ed4f63d420cf58329683b3d61835` |
| iOS 12.4 MobileWiFi.tbd | `f7ac468998f286a1e0bc850a16c1f8c12027b6032f7485c41ac4660bc679efdf` |

## P012A follow-up (original P012 evidence above preserved)

Starting feature: `88dcf9b89389cd58de6f4859d4619485451dfa82`; PR #14 remains Draft.
The [ABI manifest](ios-wifi-native-abi-manifest.json) separates verified pinned
header bytes and iOS 12.4 exports from unknown firmware-specific typed ABI.
A new pinned firmware-derived report identifies MobileWiFi 634.21 / UUID
FA49B5A2-2578-38A0-AE5E-5DDBB37D68EA in build 16H81, but its image is iPhone 6 Plus,
not the reference iPad; symbol count alone does not establish the six prototypes.

Apple's iPad4,4 12.5.7 restore image is publicly available. No smaller verified
extracted binary was found. A separate read-only cloud workflow inspects one
checksum-pinned restore image, selects board j85ap from BuildManifest, verifies
mounted SystemVersion, and records exports, binary hashes and selected function
disassembly. It never executes firmware code or accesses a physical device;
only derived evidence is uploaded. Firmware-specific conclusions remain pending
that result. No native provider is activated by these research tools.

### P012A reference firmware result

The exact iPad4,4 12.5.7/16H81 restore and cache were inspected successfully in
run 38741425098 at `6b859dd6bdf4b7be457546567ba77ef5dfc4e070`. All six candidate
exports are now verified against that image. See the ABI manifest and retained
small disassembly/report in `docs/evidence/ios-wifi-12.5.7/`. Typed ABI/opaque
ownership remain strongly inferred; no activation is authorized by export names.
CopyDevices uses manager-side cached devices; CopyCurrentNetwork does synchronous
MIG, whose daemon freshness/authorization is unknown. RSSI_CTL_AGR string presence
does not establish raw dBm. Minimal manager-access necessity/sufficiency remains
unknown. An inactive bounded process client now addresses source-side isolation;
a separate diagnostic inventories symbols but never invokes them. No experimental
flag enables acquisition. Full findings and P018 protocol are in the EN/VI native
blocker-resolution documents. P012 remains BLOCKED; Gate C PENDING.
