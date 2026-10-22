# P023 security audit

**Reviewed candidate code commit:** `f7a4525aeb7513163439b56b330ddfbd527c4da8`\
**Branch:** `ios/p023-security-hardening`\
**PR:** [#27](https://github.com/nghianguyen150612/Beszel-iOS/pull/27), Draft, targeting `ios/upstream-v021-integration`\
**Starting integration:** `fd334076f53d8354a4924a4c734f29cf65a43879`

P023 upgrades the candidate to Go 1.27.3, replaces the P019/P022 x509
backport with the fix in the official Go release, preserves the exact-source
Apple A7 runtime guard, and adds fail-closed scan policies for reviewed
module-level and frontend residuals. Gate H is pending exact-head hosted
qualification. This report does not authorize a public release.

## Verified starting state

The integration tree matched P022's final source tree. Before P023 edits,
GitHub refs were verified as:

| Ref | SHA |
|---|---|
| Integration | `fd334076f53d8354a4924a4c734f29cf65a43879` |
| Production `iOS` | `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` |
| Fork `main` | `e984ee12081f4e112601c3f718c9d81c112fe718` |
| Pinned upstream | `b5357343550c8e2493d0e65472a06c1007396384` |

PR #26 is merged and P022 Gate G passed. Live upstream main was five commits
ahead of the pinned snapshot at baseline; those commits remain excluded.

## Threat and dependency review

### Go standard library and modules

- Go 1.27.3 is declared in `go.mod`; Go Actions use
  `go-version-file: go.mod`; Docker Go build stages are digest-pinned to
  Go 1.27.3 and assert `go env GOVERSION`.
- `govulncheck v1.9.0` ran in symbol-level source mode for Agent and Hub
  roots against the Go vulnerability database last updated
  2026-10-21 00:27:29 UTC. The raw report is retained by P017.
- The reviewed report contains one module-only `GO-2026-5932` finding for
  `golang.org/x/crypto v0.58.0`. It contains no OpenPGP package or function
  trace. No other Go finding was reported.
- A repository-wide Go AST guard checks every Go file, including build-tagged
  and test files. Six `go list -deps -test` profiles cover default,
  testing/no_ui, development, development/testing/no_ui, iOS testing/no_ui,
  and iOS development/testing/no_ui. They found no
  `golang.org/x/crypto/openpgp` package. The module remains because Agent and
  Hub use `golang.org/x/crypto/ssh`; this is risk reduction, not a fix for
  the unmaintained OpenPGP packages.

### JavaScript build graph

- Both frozen lockfiles resolve `braces@3.0.3` through
  `@lingui/vite-plugin` → `@lingui/cli` → `chokidar` or `micromatch`.
  Lingui and `braces` are development dependencies.
- Bun's raw audit retained only the high-severity
  `GHSA-vfj7-8cjw-p6xm` / `CVE-2026-93687` finding.
- npm audit's latest completed report contains five high-severity affected
  package entries along the Lingui → chokidar/micromatch → braces graph; all
  five stem from one `GHSA-vfj7-8cjw-p6xm` root advisory. P023 checks the exact
  report graph and fails on additional package nodes or IDs. Its suggested
  fix is a major Lingui 6.10.0 upgrade; that suggestion is not treated as a
  verified fixed `braces` release.
- The independent OSV scan covers all 510 npm lock nodes and 494 unique
  package/version queries. The result source-bound to
  `304cc862e79ee821080e2b217da1009e945aa9d2` contains only the same GHSA for
  `braces@3.0.3`, with no fixed version. On an earlier run, the official npm
  registry timed out; only that exact timeout activates the OSV fallback,
  and raw npm errors remain retained.
- The Hub embeds generated static assets and does not ship Node or execute
  `braces` at runtime. A PR contributor can still alter checked-in Lingui
  configuration and cause build-time denial of service. PR builds run on
  isolated hosted runners with `contents:read`, checkout credentials
  disabled, no release secrets, and bounded timeouts. Build-only risk remains.

### GitHub Actions, tools, and release path

- P017 runs workflow-isolation tests for PR, branch, ordinary tag, iOS tag,
  and dispatch behavior. Release and Docker publishing workflows are
  tag-gated and do not publish on PRs.
- PR candidate workflows use read-only repository permission and
  `persist-credentials: false`. Native binaries and scan outputs upload only
  as expiring Actions artifacts.
- Go 1.27.3 and Bun 1.4.0 are version-checked. The official Linux Go 1.27.3
  archive was SHA-256 checked as
  `65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40`.
  The local Bun 1.4.0 archive was checked as
  `184fb4595f0d401a217cf7c78c1bc430ba83314dab7a8b94805babbf7fa7097f`.
  Go vulnerability tooling is pinned to `v1.9.0`; Go module checksum
  verification remains enabled.
- Docker Go builder images use reviewed Go 1.27.3 Alpine and Bookworm
  manifest digests and fail if the image reports another Go version.
- The initial `zizmor v1.31.1` scan found 72 workflow findings: 51 mutable
  action references, 9 checkout credential-persistence findings, 6 cache
  exposure findings, 4 broad permission findings, and 2 shell interpolation
  findings. P023 resolved all 19 unique action release tags to their upstream
  commit SHAs and pinned all 54 workflow action use sites. A regression test
  now rejects any unpinned action reference.
- P023 narrowed GHCR write permission to the push-only Helm publish job, moved
  chart outputs through environment variables, disabled reusable Bun/Go caches
  in release and native build workflows, and stopped checkout from persisting
  credentials. The two workflows that push with explicit secrets now use
  short-lived `GIT_ASKPASS` helpers instead of checkout credential storage.
- `actionlint v1.7.12` and `zizmor v1.31.1` official Linux release archives
  were SHA-256 verified as `8aca8db96f1b94770f1b0d72b6dddcb1ebb8123cb3712530b08cc387b349a3d8`
  and `d185f726bdaf92df78273217dfdd24ebcd6b0ef48cf3ce776ff5762b6e138c99`.
  Both scanned the final workflow set locally: actionlint passed and Zizmor
  returned zero findings. P017 repeats both checks on the exact PR source,
  fails on findings or syntax errors, and uploads raw reports for review.
- Zizmor checks workflow patterns and known vulnerable action use. This is not
  a full dependency/CVE scan of every action's bundled runtime. Commit pinning
  prevents tag movement; residual action-code vulnerability coverage remains
  limited to the scanner's online rules and is disclosed for release review.

## Regression and release safeguards

The exact-head P017 run covers the Go suite and Agent/Hub builds, dependency
scans, actionlint and Zizmor workflow analysis, auth and transport regressions,
source ancestry checks, frontend installation/tests/build, localization,
installer/bootstrap behavior, and release workflow isolation. P013 builds
and migrates a disposable genuine v0.19 fixture. The installer workflow covers bootstrap, rollback, lifecycle,
and artifact integrity. Hosted macOS CI separately builds Agent and Hub with
Go 1.27.3 and validates Mach-O metadata, iOS 12.0 minimum, A7 instructions,
checksums, source provenance, embedded assets, and private MobileWiFi linkage.

Local evidence includes a serial full Go suite PASS, 123 audit tests PASS,
`crypto/x509` tests PASS, frozen frontend installation, Bun tests, Lingui
extraction/compilation, TypeScript/Vite build, and lockfile immutability. The
first local Go full-suite run used parallel package test binaries and exposed
a shared health-marker race; the unchanged suite passed with `-p 1`. No
assertion was weakened. An early frontend command's whole-worktree check
returned nonzero because P023 code was already uncommitted; targeted
catalog/lockfile checks and the frontend rerun passed.

Hosted run 39717185832 on
`304cc862e79ee821080e2b217da1009e945aa9d2` failed the newly added actionlint
step (SC2129 and malformed heredoc indentation); Zizmor reported zero findings.
The workflow fixes are pushed at
`f7a4525aeb7513163439b56b330ddfbd527c4da8`. That source's P017, native,
installer, migration, and Helm runs are in progress or queued; the current IDs
are recorded in [P023 validation](ios-p023-validation.json). The first local
audit test invocation also used Python's default `/tmp` for temporary fixtures;
they were automatically removed. The successful rerun used the workspace-local
pinned toolchain and all temporary/cache paths under `.p023-task`. This boundary
incident remains disclosed in the validation record. Final PR-head runs remain
mandatory for Gate H.

## Boundaries and residual decisions

P022 authentication, ownership, universal-token, secret handling, TLS,
WebSocket/SSH, resource-limit, installer rollback, artifact checksum, and
release-isolation protections remain covered by existing regression tests.
P023 adds no product endpoint or authentication changes.

The remaining security decisions are release-owner decisions for the
unmaintained but unimported OpenPGP packages and the high-severity,
build-only `braces` advisory. P018 Gate F remains pending. No production
`iOS` branch, fork `main`, database, physical device, tag, or public release
was modified or accessed. Gate H does not grant release authorization.
