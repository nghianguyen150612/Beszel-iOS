# P026 signing and isolated runtime

**Status: NOT STARTED.** Device trust and launch behavior are unknown. No transfer, signing, or execution authorization has been given.

## Preconditions and authorization

Before transfer, signing, or execution, obtain a separate explicit operator approval for that operation. Confirm the Phase A identity, physical access, reliable power, verified recovery path, sufficient space, and exact artifact hashes. Keep candidate files in an isolated staging directory; do not replace production binaries.

Preserve the unsigned binaries and their hashes. If ldid is used to create ad-hoc signed copies, hash the signed copies separately. Do not compare a post-signing hash to the unsigned artifact hash and call the signing change corruption.

Do not change trust caches, disable security checks, inject speculative entitlements, or enable private MobileWiFi calls. Stop on signing rejection or uncertain trust behavior.

## Isolated startup

After separate approval, use disposable Agent/Hub data directories and nonconflicting ports. Never point the candidate Hub at the live production database during initial execution. Keep candidate copies separate from /usr/local/bin and the active LaunchDaemons.

For each process, record the exact unsigned and signed hashes, command, data directory class (disposable), port class (nonconflicting), start result, exit code, sanitized loader diagnostics, and available sanitized crash report. Do not include keys, database contents, user history, or unredacted logs.

### A7-DEVICE-01 — Agent

Required physical evidence:

- Agent launches on the iPad4,4 / iOS 12.5.7 device.
- No SIGILL, unexplained SIGKILL, recurring crash loop, or signing rejection.
- CPU and memory use remain bounded during repeated samples.

### A7-DEVICE-02 — Hub

Required physical evidence:

- Hub launches without touching production data.
- Local health endpoint responds on the isolated port.
- No unexpected exit or recurring crash loop.

The hosted artifact consumer reports the expected ARM64/iOS 12.0 Mach-O metadata and its static A7 runtime check passes for the reported Go runtime symbol. That cloud evidence does not establish signature trust, loader acceptance, whole-program instruction safety, or device launchability.

If startup fails, stop and classify the observed failure as SIGILL, SIGKILL, signing/trust rejection, loader error, or runtime error before continuing. A mandatory failure blocks Gate F pending a source fix and a newly qualified candidate.
