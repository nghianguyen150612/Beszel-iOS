# Disabled iOS Wi-Fi helper boundary

This package is not imported by normal Agent polling. `wifi_ios.go` still returns
nil. There is no environment flag that activates experimental native acquisition.
The code validates a prospective helper's output; it cannot establish that a
private API's association claim is fresh or that its number is measured dBm.

The only acquisition argv is `--snapshot-v1`; the wire encoding is one UTF-8 JSON
object, at most 16 KiB, with version 1 and source `mobilewifi-ios12`. CBOR, unknown
fields, duplicate keys, trailing values, excessive nesting, and more than sixteen
devices are rejected. An unavailable result has no devices:

```json
{"version":1,"source":"mobilewifi-ios12","status":"unavailable","reason":"activation-blocked","devices":[]}
```

A future verified producer may emit status `ok` with device fields `interface`,
`powered`, `station`, `association`, `current_network`, optional `ssid_b64` and
optional `rssi`. Conversion requires all of powered/station/current_network,
association `associated`, and a valid name resolving to a real OS interface.
A non-NULL current-network object alone is insufficient. Repeated interface
identities invalidate the result. No SSID/BSSID fallback invents an identity.

Only the integer dictionary component `rssi.RSSI_CTL_AGR` in -150..-1 is eligible
as dBm. This is a protocol constraint, not proof of source units. Floating-point,
scaled, zero, positive, overflowing and sentinel values yield unavailable signal.
The existing shared history conversion retains its documented weak-value
saturation. SSID is base64 for arbitrary 802.11 octets, at most 32 decoded bytes;
invalid UTF-8/base64 or missing/hidden/redacted SSID is omitted independently of
RSSI. No BSSID or password field is accepted or emitted.

Every parse produces a new snapshot. Errors return nil; no prior result is kept.
Only a firmware-verified native producer and independently justified privileges
can supply these claims in a future activated provider. An opt-in flag or a fake
executable satisfying this protocol does not meet those activation gates.
