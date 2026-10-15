# Connected Wi-Fi signal

Reports connected station interfaces only (no scans). `info.wf` holds the current
snapshot keyed by interface (`s` SSID, `r` RSSI in dBm when available);
`stats.wf` stores available RSSI as integer dBm. Collected on the default
interval only; real-time requests reuse the last snapshot. Set `SKIP_WIFI=true`
on the agent to disable collection.

- Linux: nl80211 via `github.com/mdlayher/wifi`. Docker needs `network_mode: host`.
  Skipped when `/sys/class/ieee80211` lists no wireless devices.
- macOS: CoreWLAN via `osascript` (JXA). SSID may be redacted by privacy settings.
- Windows: native WLAN API, keyed by adapter alias (e.g. `Wi-Fi`), falling back
  to interface GUID.
- iOS: explicit `wifi_ios.go` unavailable adapter. Native MobileWiFi acquisition
  is deferred pending a verified iOS 12 ABI, fresh association, authorization and
  bounded-call contract. No private framework is loaded or linked. See
  [provider feasibility](../../docs/ios-wifi-provider-feasibility.md).
- Other platforms: unsupported.

History conversion accepts finite readings in the existing collectors' -150..-1
native dBm range. Invalid, positive, normalized 0..1, and non-finite values are
omitted. The existing int8 history policy saturates valid -150..-129 values to
-128; Info retains the original valid reading. Range checking alone cannot prove
units: a provider must establish that its source is raw dBm before supplying it.
