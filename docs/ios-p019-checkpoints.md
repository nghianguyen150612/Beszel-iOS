# P019 durable checkpoint ledger

## Reconstructed integration

The reported old P019 commits were not present in the local object database,
reflogs, or GitHub. The old work is classified as **UNRECOVERABLE UNPUSHED
LOCAL WORK**. P019 was reconstructed with new commit identities from integration
`f9d6500c7a68572dc8e464dd305d79bf6c1e79a9` and pinned upstream
`b5357343550c8e2493d0e65472a06c1007396384`.

The normal upstream merge is `fdac7746bc5eb4f2e6a3a24656f08790c105a65d`.
Its second parent is the pinned upstream commit. It accounts for all 41 upstream
commits after baseline `bb23f99b3eeaced9608e11e280e7c986dba650be`; the tracked
patch map still contains 13 logical iOS patch IDs. P019 Draft PR #23 targets
`ios/upstream-v021-integration`.

## Durable source checkpoints

| Source SHA | Remote verified | Changes |
| --- | --- | --- |
| `52967f47c91b82d9ea7455a541ea7c7ca099a19d` | Yes | Recovery audit and reviewed normal upstream merge |
| `7099bb0aa63b95b4d14c1f525d8fa2be6d87cd47` | Yes | Go 1.27.2 A7 runtime source pin; telemetry and monitor compatibility corrections |
| `cee2fc2518e2abbc58a2d6375dc85d7c996e9a95` | Yes | P019 Go and JavaScript security qualification, report validators, and evidence |
| `8b8284961e37dd2264084071b112c1b38e47bfe3` | Yes | Correct the migration schema expectation, legacy disk units, Procursus isolation runner, and govulncheck stream parser |
| `070c4119fa5ef13d90ea26e3d7b6a9ed84613436` | Yes | Record exact-head security findings and correction run state |
| `b14759d5856cbd8adf799ca05145c1bc9314841d` | Yes | Accept upstream `system_details.sockets:number`; P013 real fixture rerun is pending |
| `4bac1ba75872b72027cb11a89da0f8c01b31f7db` | Yes | Record the P013 quiet-hours ownership comparison failure and exact correction |
| `50ae2d8f4bc63fdeb1eeb2bc9f436059e5170846` | Yes | Add the exact upstream `quiet_hours.updateRule` owner-preservation rule |
| `05d69686fade8ffde24f25ac08f8fee9a1c82bc6` | Yes | Record the following `user_settings.updateRule` comparison failure |
| `14edfae855610b9bf7aa795f452fe82afe81b427` | Yes | Allow the exact upstream user-settings owner-preservation rule |
| `cdc19642b87c1c47f938e61f0c3e85ba5b67461f` | Yes | Record the P013 fingerprint system-reassignment guard finding |
| `7b1a2db386fca2c7985724b3d11d561725a20b0d` | Yes | Allow only the exact upstream fingerprint system-preservation rule |

## Hosted CI on `cee2fc2518e2abbc58a2d6375dc85d7c996e9a95`

These results belong only to the exact source SHA above. They do not qualify a
later source revision.

| Workflow | Run | Result | Finding |
| --- | --- | --- | --- |
| iOS installer lifecycle | [39572887793](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39572887793) | PASS | Installer lifecycle checks passed. |
| Helm charts | [39572887805](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39572887805) | PASS | PR validation passed; registry login and publishing were skipped. |
| P013 Hub migration fixture | [39572887539](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39572887539) | FAIL | The harness allowed only `NetworkMonitorLoss`; pinned upstream also appends `NetworkMonitorCert`. The accepted schema and migration docs now include both ordered additions. |
| iOS Build | [39572887541](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39572887541) | FAIL | The Bun regression expected bytes/sec in legacy fields that are MiB/sec. The test and type comments now use MiB/sec and verify conversion. The Procursus isolation test could not establish a sandbox on the Ubuntu host; the test no longer invokes `bwrap` and is being moved into a hosted container with network disabled and a read-only root. The native build did not complete on this source. |
| P017 release-candidate regression | [39572887761](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39572887761) | FAIL | The Go vulnerability validator assumed line-delimited JSON, while govulncheck emits indented JSON messages. It now decodes the concatenated JSON stream. No security PASS is inferred from this failed run. |

On the subsequent exact source `070c4119fa5ef13d90ea26e3d7b6a9ed84613436`,
the Procursus isolation test passed in the hosted `--network none`, read-only
container; Bun tests, Lingui, TypeScript, Vite, source selection, the Go scanner,
and Bun/npm audits also passed. P013 built the real database fixture but failed
schema comparison because upstream adds `system_details.sockets` (number). The
harness and migration docs now include this required CPU-socket migration; a
fresh real fixture run is pending. P017 full Go regression and the macOS
portable telemetry suite were still running when this checkpoint was written.

On exact source `4bac1ba75872b72027cb11a89da0f8c01b31f7db`, the genuine P013
fixture accepted the `system_details.sockets` field and then reported that
upstream tightens `quiet_hours.updateRule` to append
`@request.body.user:changed = false`. The harness and migration docs now accept
only this exact ownership-preserving rule change. The focused Hub migration
unit suite passed again (7 tests) on the correction tree. Fresh hosted checks
on pushed source `50ae2d8f4bc63fdeb1eeb2bc9f436059e5170846` then reached
`user_settings.updateRule` and found the same exact owner-preservation clause.
The harness and migration docs now permit that exact rule for both
`quiet_hours` and `user_settings`. The Hub migration unit suite passed again
(7 tests). On exact source `cdc19642b87c1c47f938e61f0c3e85ba5b67461f`, the
fixture then found that upstream also prevents `fingerprints.updateRule` from
changing the system. The harness and docs now permit only the exact
system-preservation guard. The Hub migration unit suite passed again (7 tests).
Fresh hosted checks are running on pushed source
`7b1a2db386fca2c7985724b3d11d561725a20b0d`.

The exact-head Go report confirms `GO-2026-6608` is absent on Go 1.27.2, but it
also reports three reachable `GO-2026-6809` traces through
`internal/ghupdate.FetchLatestRelease` to `crypto/x509.Certificate.Verify`.
The reviewed Go vulnerability entry says this is fixed in Go 1.27.3. The report
also contains one module-level `GO-2026-5932` OpenPGP advisory without a call
trace; the source tree has no OpenPGP imports. Bun and npm report the expected
open high `GHSA-vfj7-8cjw-p6xm` build dependency advisory (five affected npm
dependency records, zero critical findings). Go report artifact ZIP digest:
`76b34bd8b746d04f354ccce7507e8ff80196b7cd086057e597690d0a814a70ab`; JavaScript
report artifact ZIP digest:
`6d036a05215056e8494aff69e4ed37ac584cc80beff4bd2688a825ca17db8990`. Both match
GitHub artifact metadata. The newly disclosed reachable Go advisory remains a
security gate; Go 1.27.2 cannot yet be marked acceptable for P019.

## Local correction checks

On the correction tree, Python audit tests passed (74 tests), Hub migration
harness unit tests passed (7 tests), frozen upstream matrix and readiness
validation passed (178/178 upstream rows and 13 logical iOS patches), and
`git diff --check` passed. Bun and Go suites have not been run in this Work
environment. Commit `7b1a2db386fca2c7985724b3d11d561725a20b0d` is pushed and its
exact remote SHA was verified. Its hosted checks are pending; exact-head hosted
CI remains authoritative. The required Procursus fixture must run in its hosted
network-disabled, read-only container.

## Gate state

P019 remains **IN PROGRESS**. The corrections are committed and pushed but
have not yet been run through CI. Source-level Go vulnerability disposition,
JavaScript advisory disposition, genuine v0.19-to-candidate database upgrade
and restore, full Go/race and frontend suites, native ARM64 iOS artifacts,
source provenance, mandatory reviews, and all exact-head required checks remain
open. Do not merge PR #23 until each required gate has passing evidence.

No release, tag, deployment, production branch update, fork `main` update, or
device action is authorized. P018 physical-device Gate F remains pending.

## P013 final-migration failpoint correction — 2026-10-21

The exact-head P013 run on `5368c60a7794cd3ca931d23544cd01d89c8624fb`
passed the schema and authorization comparisons, then failed the deterministic
post-schema assertion because the failpoint was keyed to
`1790964708_network_monitor_skip_tls_verify.go`. Upstream now applies two later
migrations, `1791926360_network_monitor_cert_alert.go` and
`1792309602_system_details_sockets.go`. The harness now requires all seven
relevant migration ledger entries exactly once and in order, and arms the
post-schema failpoint only after the final `1792309602` entry.

Focused Hub migration harness unit tests passed (7/7); `git diff --check`
passed. Correction commit `7b7052d439cb565bb6b5c16c9c547cb167266c70` was
pushed to `ios/p019-upstream-main-b5357343`, and `git ls-remote` confirmed the
same SHA. Hosted checks for this source are not yet available. The preceding
exact-head run remains a genuine failure; it is not counted as a pass. P019
still cannot merge while mandatory checks and the Go 1.27.2 security
disposition remain open.

## P013 full migration qualification — 2026-10-21

P013 workflow run `39578272401` passed on source
`89e9c2016a2dc02541e64cdf2007ecf986652d94`. The synthetic-only evidence ZIP
SHA-256 is
`67213973461129ff41fd7b7e782130468772b1fc73f0c9c9cd8d246e3a04749c`, matching
GitHub artifact metadata. It records a genuine disposable v0.19 Hub fixture,
all seven candidate migrations once and in order, two healthy candidate starts
with stable schema and record counts, preserved roles/settings, WAL-backed
history, and SSH public-key fingerprint. The complete cold backup restored
with matching inventory and SQLite integrity. All five injected interruption
points recovered through the verified v0.19 restore. The artifact contains
only synthetic records, hashes, public-key fingerprint, and sanitized loopback
failure diagnostics; it contains no database archive, private key, credentials,
or production data.

## Connection regression correction — 2026-10-21

The exact-head native iOS Build run `39578272394` on `89e9c201` failed in the
portable telemetry/macOS test step. `sendEvent` could block WebSocket callbacks
on a nil or full event queue, and `handleAuthChallenge` announced connection
before its fingerprint response was written. The callback now enqueues without
blocking and connect is reported only after the response is sent and verified.
The failed-send regression asserts no connect event is queued. The local full
agent test suite also found that `getOptions` dereferenced an absent Agent in a
TLS-only test; it now falls back to the dialer's background context.

The full agent suite then exposed overly long Unix socket paths in
`TestLegacyHubGetsSinglePayload` on macOS. The helper now creates a short socket
directory under the package checkout and reports listener startup errors
directly. This stays inside the checkout and below macOS's Unix socket path
limit.

The official Go 1.27.2 archive was checksum-verified against `go.dev` and
installed only in `.task/cache`. Focused P009 tests passed, the full agent,
battery, delta-tracker, and system suites passed, and the full `agent` race
suite passed. `git diff --check` passed. Code checkpoint
`ebed4cbc1bd4320f06b81b3f9c29c2f089237e58` is pushed and confirmed by
`git ls-remote`. Hosted P017 and native checks on this corrected source are
pending; failures and in-progress runs from `89e9c201` are not counted as
passes. The Go 1.27.2 `GO-2026-6809` security disposition remains open.
