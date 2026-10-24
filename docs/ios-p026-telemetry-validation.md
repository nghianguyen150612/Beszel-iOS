# P026 telemetry and functional validation

**Status: PENDING physical execution.** Do not collect against production data during initial isolated startup. No physical telemetry result is recorded.

Proceed only after isolated Agent and Hub startup pass and the operator authorizes the functional test scope. Use disposable or explicitly authorized data and retain only redacted evidence.

## Agent checks

- CPU use and per-core telemetry are current and plausible.
- RAM values are bytes and remain compatible with the legacy consumer contract.
- Missing or unsupported measurements remain absent/unknown; no values are fabricated.
- Battery percentage and charging status match the device.
- Disk and network counters move plausibly under controlled activity.
- Timestamps advance and identify fresh samples.
- Repeated collection remains stable without crash loops or unbounded resource use.

Record sample count and sanitized ranges, not personal activity or identifiers. Confirm each reported value against an independent device reading where available.

## Hub and transport checks

- Hub local health responds on the isolated test port.
- Agent connects to the isolated Hub and reconnects after a controlled network interruption.
- WebSocket/SSH transport works with test credentials; record only pass/fail and redacted diagnostics.
- Authentication and authorization enforce expected access.
- Network monitor checks HTTP and HTTPS endpoints.
- TLS certificate monitoring returns a fresh result for valid and invalid test certificates.
- Historical records are collected from disposable or explicitly authorized data.
- Frontend renders on a real client and presents missing data honestly.

Never record Agent keys, passwords, private keys, production database rows, personal telemetry, or unredacted logs.

## Deferred and unsupported signals

Native Wi-Fi RSSI remains DEFERRED. Do not enable private MobileWiFi calls or infer Wi-Fi RSSI from another collector. Unsupported features must remain unavailable rather than displaying synthetic values.
