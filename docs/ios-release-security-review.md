# P017 release-candidate security disposition

Date: 2026-10-18. Baseline: Go 1.27.1, Bun 1.4.0, `golang.org/x/net` v0.60.0.

- **GO-2026-6608 / CVE-2026-94440 — HIGH, OPEN.** Go multipart MIME parsing has a memory-limit bypass, reachable via Hub PocketBase binding and Agent WebSocket handshake. A fixed Go 1.27.2 toolchain is not validated for the A7 runtime patch. Do not claim a fix; restrict exposure to trusted networks and prioritize a separate native CI + A7 retest before release.
- **GHSA-vfj7-8cjw-p6xm — HIGH, OPEN.** `braces` v3.0.3, transitively used by Lingui build tools through micromatch/chokidar. Not embedded in shipped Hub frontend; no compatible in-range update as of P015 review. Do not classify it as fixed.

The P015 review documents HTTP/2 server ALPN disabled, default HTTP/2 clients disabled, and 16-range request cap for other high/critical Go findings. These mitigations remain part of unchanged P017 product source; they do not fix the multipart issue. There is no new P017 source vulnerability scan; do not claim one.

Go/P017 source and frontend regressions passed; no new unmitigated critical release-blocker was identified within documented coverage. **Gate E cloud acceptance is not final public release approval.** Gate F requires physical A7 validation, explicit operator risk acceptance, and release-specific authorization. Never publish automatically.
