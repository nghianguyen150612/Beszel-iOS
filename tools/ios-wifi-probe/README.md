# Separate disabled native diagnostic

This tool is **not an RSSI provider**. No private function is cast or called. It is
not linked into Agent/Hub, installed, signed for production, or included in their
review payload. Normal `wifi_ios.go` remains unavailable. No opt-in enables it.

`--snapshot-v1` always returns protocol-v1 unavailable without loading MobileWiFi.
`--identity-v1` emits bounded model/build/revision strings. `--symbols-v1` loads
only the fixed system framework and inventories six symbol addresses, without
invoking them. Symbol presence proves neither ABI nor daemon authorization.
`--build-info-v1` identifies the exact source revision and disabled activation.
No mode scans, changes power/configuration, or reads SSIDs/BSSIDs/passwords.

`core.c` contains reusable public CoreFoundation conversion paths. It accepts
only an established CF dictionary with integer RSSI_CTL_AGR in -150..-1 and
converts borrowed children before releasing an explicitly owned container. This
neither establishes private object ownership nor proves that firmware values use
dBm. Opaque manager/device/network objects are never cast to public collections.

The cloud host fixture builds use address/undefined sanitizers, an allocator
counter over repeated successful/error release paths, invalid types and strings,
and deliberately incompatible fake symbol exports that abort if invoked. They
execute public CF logic and loader inventory only. The iOS cross-build checks
ARM64/platform 2/minimum 12.0, source hashes, embedded revision and absence of hard
private imports. The separate artifact is for review, never automatic deployment.

See `docs/ios-wifi-native-blocker-resolution.md` for boundaries and the staged
P018 diagnostic protocol. Bundling, separate signing, executable ownership and
least privilege belong to P016 after the unresolved contracts are established.
