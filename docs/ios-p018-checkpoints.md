# P018 checkpoint ledger

## Starting point

- Integration baseline: P017 Gate E normal merge `f9d6500c7a68572dc8e464dd305d79bf6c1e79a9`.
- Branch: `ios/p018-device-validation`, based exactly on that merge.
- No P018 hardware test performed as of the initial checkpoint.
- Production `iOS` and frozen `main` remain protected.
- Candidate source `5a7351f752376e206a3e1809b3698bb03cc87b98`; SHA evidence in `docs/ios-release-candidate-artifacts.json`.

## Checkpoint A — Read-only preflight and safe handoff

Implemented:
- `scripts/ios/p018-device-preflight.sh`: read-only local identity/path/hash/service inventory. Requires no network and makes no mutation. Redact output before attaching.
- `docs/ios-p018-device-handoff.md`: reference-device matrix, two-stage operator approval, backup requirements, staged candidate/rollback, jailbreak-reactivation steps and release stop conditions.
- `docs/ios-p018-validation.json`: structured Gate F matrix with all physical and signing checks explicitly PENDING.

Source scope: preflight and evidence only; no installation, signing, trust-cache changes, LaunchDaemon changes, reboot, database modifications or release.

## Next required step

Get explicit confirmation that the **physical reference iPad** and recovery path are available. Inspect the preflight script, transfer it through an authorized channel, run `sh /var/tmp/p018-device-preflight.sh` on the actual device, and provide redacted results. Do not proceed to candidate staging until complete cold backup, restore rehearsal and operator approval.

P018 cannot reach PASS without real hardware evidence. Native Wi-Fi RSSI remains DEFERRED without an approved ABI/authorization experiment. The high-severity Go multipart and build-only braces advisories remain OPEN.

**Current Gate F: PENDING — REFERENCE DEVICE UNAVAILABLE.**
