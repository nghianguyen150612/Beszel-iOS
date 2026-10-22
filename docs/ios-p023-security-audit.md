# P023 security audit

**Qualified implementation source:** fd6b1595ff2e8b4d7037229ce6d71af87f907b4e
**Branch:** ios/p023-security-hardening
**PR:** [#27](https://github.com/nghianguyen150612/Beszel-iOS/pull/27), Draft, targeting ios/upstream-v021-integration
**Starting integration:** fd334076f53d8354a4924a4c734f29cf65a43879
**Qualification snapshot:** 2026-10-22

P023 upgrades the Go toolchain to 1.27.3, verifies the native x509 fix, retires the redundant P021/P022 x509 backport, preserves the fail-closed Apple A7 runtime patch, and records current Go and JavaScript advisory exposure. The implementation source passes the P017, native iOS, installer, migration, Procursus, and Helm hosted workflows. The documentation checkpoint that contains this report must also pass exact-head checks before merge.

A Gate H PASS qualifies this cloud integration candidate only. It does not authorize a public release or close P018 Gate F. The GO-2026-5932 and GHSA-vfj7-8cjw-p6xm release-owner decisions remain OPEN.

## Verified starting state and preserved history

Before P023 edits, GitHub refs were verified as:

| Ref | SHA |
|---|---|
| Integration | fd334076f53d8354a4924a4c734f29cf65a43879 |
| Production iOS | 0c3e4cd22ac24264834c401ad412e44aeda4e4f3 |
| Fork main | e984ee12081f4e112601c3f718c9d81c112fe718 |
| Pinned upstream | b5357343550c8e2493d0e65472a06c1007396384 |

PR #26 is merged and Gate G passed. Live upstream main was b97abf7662a7c7a2dd7542ea2062405abd22d4dc, five commits ahead of the pinned snapshot, both at baseline and at this review. Those commits are excluded from P023.

P017's source-ancestry check passes on fd6b1595. It confirms 41/41 pinned upstream commits, 178/178 frozen historical audit rows, 35/35 historical iOS commits, and 13/13 logical iOS patchsets. P019–P022 normal merge ancestry is preserved.

## Go and module findings

The official Go 1.27.3 toolchain is pinned and checked across go.mod, Actions, native build scripts, provenance validators, Docker builders, and release validators. P017 uses govulncheck v1.9.0 in symbol-level source mode with Go 1.27.3 and reports only GO-2026-5932, as a module-only finding for golang.org/x/crypto v0.58.0.

- GO-2026-6608 is absent from the Go 1.27.3 source scan. Go fixed it in 1.27.2.
- GO-2026-6809 is absent from the Go 1.27.3 source scan. The official native fix is verified by source hashes and crypto/x509 tests; the custom P021/P022 backport is no longer applied.
- GO-2026-5932 remains in raw scanner output. A repository-wide AST scan and six host/iOS build-tag and test dependency profiles find no OpenPGP import or package. The x/crypto module remains for SSH. This is reachability evidence, not a fix for the unmaintained OpenPGP code.

The reviewed Go vulnerability records do not publish a severity score for these findings. The P021 historical review classified GO-2026-6608 as HIGH. The other Go advisory severity fields remain unscored.

## JavaScript dependency finding

Bun and npm lockfiles resolve braces 3.0.3 through Lingui CLI, chokidar, and micromatch. Bun and npm audits report GHSA-vfj7-8cjw-p6xm / CVE-2026-93687 as HIGH (CVSS 7.5). npm shows five affected package entries that trace to this one advisory. A source-bound OSV scan covers all 510 npm lock nodes and 494 unique package/version queries and reports only this GHSA. No fixed version is listed by the reviewed advisory.

The affected packages are development/build tooling and are not established as Hub runtime dependencies. The Hub embeds generated static assets. Build jobs run on isolated hosted runners with read-only contents permission, no persisted checkout credentials, no release secrets, and bounded timeouts. A contributor can still cause build-time denial of service by changing checked-in patterns. P023 keeps the warning OPEN and records a build-only mitigation with residual risk; it does not claim the package is fixed.

## GitHub Actions and downloaded tools

All 54 external action use sites are pinned to 19 upstream-resolved full commit SHAs. P023 also narrows write permissions, disables reusable caches on native/release builds, disables checkout credential persistence, and moves required write credentials to short-lived GIT_ASKPASS helpers.

On the exact fd6 source, pinned actionlint 1.7.12 and ShellCheck 0.11.0 pass; pinned zizmor 1.31.1 reports zero findings. The workflow evidence records tool versions, archive digests, source SHA, raw logs, and exit codes. Zizmor does not provide a complete CVE scan of every bundled action runtime; that scope limit remains disclosed.

Go 1.27.3, Bun 1.4.0, govulncheck 1.9.0, actionlint, ShellCheck, and zizmor were version or digest checked. Docker Go builder images are digest-pinned and assert their Go version.

## Exact-source cloud evidence

All links below are hosted GitHub checks on source fd6b1595ff2e8b4d7037229ce6d71af87f907b4e:

| Gate | Run | Result |
|---|---:|---|
| P017 regression and security | [39718772365](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39718772365) | PASS; full serial Go suite and portable Agent/Hub builds, Go and JS scans, actionlint, ShellCheck, zizmor, source ancestry |
| Native iOS and Procursus | [39718772276](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39718772276) | PASS; exact Go 1.27.3 runtime/x509 checks, iOS ARM64 Agent/Hub, Mach-O, provenance, checksums, assets; release jobs skipped |
| P013 Hub migration | [39718772288](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39718772288) | PASS; genuine disposable v0.19 fixture and recovery tests |
| Installer lifecycle | [39718772319](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39718772319) | PASS |
| Helm workflow | [39718772278](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39718772278) | PASS; no publish job executed on the PR |

The native payload archive, validation archive, disabled Wi-Fi diagnostic archive, Go scan archive, JavaScript audit archive, workflow scan archive, and P017 regression manifest were independently downloaded. Every archive SHA-256 matched GitHub's artifact digest. Agent and Hub payload SHA-256 values matched the included SHA256SUMS and the independent artifact validator.

The independent local payload validator confirmed both Mach-O binaries are ARM64 iOS executables with minimum iOS 12.0, Go 1.27.3, GOARM64 v8.0, CGO_ENABLED=1, clean VCS metadata at fd6b1595, and Hub embedding all 97 production frontend assets. The A7 runtime instruction reports pass with no counter-register access. Agent and Hub have no private MobileWiFi/CoreWLAN linkage. This does not establish launchability on a physical A7 device.

P017 release-isolation tests pass. The native release validation and publishing jobs were skipped for the PR; no release or tag was created.

## Regression scope and remaining decisions

Existing P019–P022 OAuth default-role, record ownership, universal-token authorization, secret masking/redaction, no secret-bearing GET URL, TLS certificate validation, WebSocket/SSH reconnect, timeout/cancellation, network-monitor recovery, HTTP resource-limit, installer rollback, artifact checksum/provenance, and release-isolation checks remain enabled. P023 adds only advisory and workflow supply-chain guards; it does not add product endpoints or weaken existing tests.

The outstanding release-owner decisions are:

1. Accept or require removal of the unmaintained but unimported OpenPGP module-level residual, GO-2026-5932.
2. Accept the HIGH build-time braces denial-of-service residual or wait for a genuine fixed release/dependency path, GHSA-vfj7-8cjw-p6xm.

The first local audit unittest invocation omitted TMPDIR and briefly created automatically cleaned Python fixtures under the default /tmp. The failed invocation used a nonstandard /usr/bin/go wrapper. The successful rerun used Go 1.27.3 and placed HOME, TMPDIR, TMP, TEMP, and Go caches under /workspace/Beszel-iOS/.p023-task. This workspace-boundary incident is recorded in the validation file. No other workspace violation was recorded.

Production iOS, fork main, production data, devices, tags, and public releases were not changed or accessed. No sudo was used.
