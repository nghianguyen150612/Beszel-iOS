# P015 dependency and advisory review

## Scan context

Review date: 2026-10-17 UTC. Target toolchain: Go 1.27.1 and Bun 1.4.0.

- Go inventory: 127 modules from go.mod/go.sum and go list -m all.
- govulncheck v1.9.0 source scan; vulnerability database last modified 2026-10-08T22:31:09Z.
- Bun audit v1.4.0 against the frozen Bun graph.
- npm audit 11.9.0 against retained package-lock.json. CI builds with Bun, not npm ci.
- Focused Go update: `golang.org/x/net` v0.59.0 → v0.60.0. No frontend direct dependency, `package.json`, or `package-lock.json` changed.
- Go checksums remain in go.sum; npm package integrity metadata remains in package-lock.json and bun.lock. No package was added. A comprehensive license compatibility scan was not performed.

## Bun graph and focused remediation

The original frozen Bun graph produced 29 advisory records across nine transitive package families: one low, eight moderate, nineteen high, and one critical. They were build-time dependencies, not frontend runtime bundles. Bun audit fix selected releases within declared semver ranges and resolved 28 records in eight packages:

| Package | Prior | Updated | Compatibility |
| --- | --- | --- | --- |
| @babel/core | 7.28.3 | 7.29.6 | Declared ^ range |
| brace-expansion | 5.0.5 | 5.0.12 | Declared ^ range |
| browserslist | 4.25.1 | 4.28.7 | Declared ^ range |
| js-yaml | 4.1.1 | 4.3.2 | Declared ^ range |
| nanoid | 3.3.11 | 3.3.18 | Declared ^ range |
| postcss | 8.5.6 | 8.5.23 | Declared ^ range |
| source-map-js | 1.2.1 | 1.2.2 | Declared ^ range |
| tar | 7.5.13 | 7.5.21 | Declared ^ range |

The tar advisory is GHSA-23hp-3jrh-7fpw (advisory severity critical; CVSS 7.5). tar is a development transitive of @tailwindcss/oxide and @tailwindcss/vite, and is not shipped in the generated frontend or Hub. All eight updates are in the build dependency graph. Two frozen clean builds, TypeScript, Lingui, Vite, Bun tests, Hub serving, and browser acceptance passed with the updated lock. Manifests were unchanged.

One Bun finding remains: GHSA-vfj7-8cjw-p6xm, braces <=3.0.3, high severity, CVSS 7.5. The lock resolves braces 3.0.3 through micromatch/chokidar under @lingui/cli 5.4.1, a development-only dependency. The latest compatible braces remains 3.0.3 and no compatible fix is published. Bun/npm suggest a major Lingui update to 6.10.0; it was not applied. The finding is limited to build-time pattern processing and is not bundled into the Hub frontend.

The retained npm lock audit reports five high package-wrapper records (@lingui/cli, @lingui/vite-plugin, braces, chokidar, micromatch), all for that same GHSA; they collapse to one unique advisory. The npm lock remains unchanged.

## Focused Go module update

The initial Go scan found module-level `golang.org/x/net` findings fixed in v0.60.0 (including GO-2026-6617, GO-2026-6612, GO-2026-6611, GO-2026-6610, and GO-2026-6603). P015's minimal compatible update from v0.59.0 to v0.60.0 removes those module findings in the final govulncheck scan. The standard-library findings with overlapping IDs remain separately listed below; the pinned Go 1.27.1 runtime was not changed. Frozen Go tests, targeted race tests, and Linux Agent/Hub builds passed with x/net v0.60.0.

## Go findings and disposition

govulncheck reported 12 unique reachable Go 1.27.1 standard-library advisories in source traces. The Go vulnerability database does not supply severity; severity and CVSS below are from linked CVE/NVD data. Fixes for this target toolchain are in Go 1.27.2.

| Go advisory / CVE | Severity | Finding and disposition |
| --- | --- | --- |
| GO-2026-6612 / CVE-2026-78663 | Critical, CVSS 9.1 | HTTP/2 server flow-control refund. Production Hub TLS disables HTTP/2 with a non-nil empty TLSNextProto map. TLS ALPN test confirms HTTP/1.1 even with GODEBUG=http2server=1. |
| GO-2026-6603 / CVE-2026-78659 | High, CVSS 7.5 | HTTP/2 server memory exhaustion. HTTP/2 negotiation is disabled and tested. |
| GO-2026-6617 / CVE-2026-97032 | High, CVSS 7.5 | HTTP/2 server HPACK race. HTTP/2 negotiation is disabled and tested. |
| GO-2026-6610 / CVE-2026-78660 | High, CVSS 7.5 | HTTP/2 client framing. Both executables default HTTP/2 clients off; golang.org/x/net/http2 is not imported. |
| GO-2026-6611 / CVE-2026-78669 | High, CVSS 7.5 | HTTP/2 client/server CPU behavior. Client defaults are off and server negotiation is disabled. GODEBUG=http2client=1 can override the default; do not set it on Go 1.27.1. |
| GO-2026-6609 / CVE-2026-78667 | High, CVSS 7.5 | Excessive Range parsing. Middleware rejects requests with more than 16 byte ranges; unit and live checks confirm 17 returns 400. |
| GO-2026-6608 / CVE-2026-94440 | High, CVSS 7.5 | Multipart MIME memory-limit bypass. Reachable through PocketBase body binding and the Agent WebSocket handshake. Go 1.27.2 fixes it; the pinned 1.27.1 target leaves a residual high risk. Keep Hub/Agent on trusted networks and prioritize a toolchain update when the baseline permits. |
| GO-2026-6605 / CVE-2026-56866 | High, CVSS 7.5 | HTTP/1 client desynchronization after CONNECT rejection. The application does not construct outbound CONNECT requests. |
| GO-2026-6613 / CVE-2026-94439 | High, CVSS 7.5 | HTTP/1 server desynchronization after 2xx CONNECT. No Hub route implements CONNECT or returns 2xx to it. |
| GO-2026-6607 / CVE-2026-97031 | High, CVSS 7.5 | Malformed ECH outer-extension references. No ECH configuration/client ECH setting is used. |
| GO-2026-6599 / CVE-2026-94448 | NVD not scored | html/template context tracking. PocketBase built-in templates are trusted; no attacker-controlled JavaScript template is supplied by the product. |
| GO-2026-6600 / CVE-2026-97030 | NVD not scored | html/template regexp context. Same trusted-template condition as GO-2026-6599. |

GO-2026-6604 is Windows-only and does not affect the iOS/Linux target. GO-2026-5932 concerns unmaintained golang.org/x/crypto/openpgp; the application does not import that package and go mod why says the main module does not need it. Other x/crypto findings concern SSH subpackages or call conditions not used by traced product paths; the module remains required for SSH and ACME.

The critical HTTP/2 issue remains in static govulncheck output because source reachability does not model TLSNextProto ALPN policy. Production policy and the direct TLS regression test are the mitigation evidence. Do not enable HTTP/2 with runtime overrides on Go 1.27.1.

## Gate D disposition

No unmitigated critical advisory remains in the review. Two high findings remain transparently recorded: build-only braces without a compatible fix and reachable Go multipart parsing (GO-2026-6608). The latter requires Go 1.27.2, outside the pinned P015 toolchain, and remains an operational risk. After the focused `x/net` v0.60.0 update, current-source iOS Build [39014286778](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39014286778), installer lifecycle [39014286727](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39014286727), and P013 migration fixture [39014280816](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39014280816) passed at exact tested head `b83e179bd84b2177cd7fc8b5b0f5460d5f2411ba`. The CI release job was skipped. These checks support Gate D acceptance while the two high residuals remain explicitly open for future toolchain/dependency work.
