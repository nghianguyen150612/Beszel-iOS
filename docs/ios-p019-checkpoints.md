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
| `af9e158f2830c11f5a145adf380c7723956891af` | Yes | Synchronize the Wi-Fi helper fixture startup marker after a hosted macOS race-test failure |
| `a13497171883043a9868a11c428c30b7d86d99eb` | Yes | Replace immediate listener port rebinding with a shutdown-state and connection-refusal assertion after hosted macOS reported the port still in use |
| `bc6967dc3cffe2d46be5b427aa490e87227ab571` | Yes | Backport Go CL 851047 for GO-2026-6809 on the exact Go 1.27.2 source and enforce the patch in P017/native CI |

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

## P019 full qualification checkpoint — 2026-10-21

Code checkpoint `de40fbe8d3b5782d979bfbc2cbd76c5944b46764` is pushed and was
confirmed by `git ls-remote`. It fixes the stale Linux P014 golden response
(upstream removed Btrfs `r`/`w` fields), removes a cross-file Bun module-mock
collision in the token privacy test, and refreshes the frontend asset manifest
from the genuine upstream build.

Local checks on this source passed:

- `env -u NETWORK go test -tags 'testing no_ui' ./... -count=1 -timeout=20m`
- Bun 1.4.0: 60 tests passed, zero failed.
- Lingui sync, TypeScript build, and Vite production build passed; `bun.lock`
  stayed unchanged.
- Frontend validator and Go embed test passed for 97 files (3,402,561 bytes).
- P014 authenticated remote telemetry API regression passed; `git diff --check`
  passed.

Hosted checks on exact source `de40fbe8`:

| Workflow | Run | Result | Evidence |
|---|---:|---|---|
| P017 release-candidate regression | [39583459164](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39583459164) | PASS | Full Go tests and builds, installer/bootstrap and isolation suites, upstream ancestry, Go and JS audits, frontend build. Source SHA `de40fbe8`; 97 assets; canonical frontend digest `3b80dbea2957210afc99b4f4e2322d682015f53eb6034d163f55d3a9fc2a92e1`. |
| P013 migration fixture | [39583459141](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39583459141) | PASS | Genuine v0.19-to-candidate migration and recovery fixture; evidence ZIP digest `435a67fc2dd2c898b8967ca1b15ea72a67bce89f1a650986d00e9c5ed392dd44`. |
| iOS installer lifecycle | [39583459340](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39583459340) | PASS | Installer transaction and rollback workflow. |
| Helm charts | [39583459150](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39583459150) | PASS | PR validation; no publishing. |
| Native iOS Build | [39583459159](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39583459159) | PASS | Go 1.27.2 A7 runtime patch, ARM64 Agent and Hub, iOS 12 minimum, provenance, checksums, frontend embedding, and native validation artifacts. |

The native run patched `runtime/asm_arm64.s` from SHA-256
`47f9313d4f956b4159dcf8b292bf340a12adb6b58ee9a94360fe948045e2c5db` to
`c861b98d76ac487269c7f98c987b94ea9978c5f01740a4596eecbfdb9c8bb91f`. It
produced byte-identical clean-cache builds. Agent SHA-256 is
`31da7e71c4f3067261e9468965ef55ef499e0011871efadbc1f5bb108fc5e506`; Hub
SHA-256 is `6edd4259c87999d1315c41a672fa40204e00babd81fc7948056d37c144be5ed7`.
Both Mach-O files are ARM64 iOS device binaries with minimum iOS 12.0, Go
1.27.2, `CGO_ENABLED=1`, `GOARM64=v8.0`, VCS revision `de40fbe8`, and
`vcs.modified=false`. The Hub contains 97 frontend assets. Review artifact ZIP
digests: native payload `4cce2006cd6553e50889f25181ea13af7b9e553a2ea2590e2bba04c03542352a`,
native evidence `e158d234e055a56cc938ae2041032cec653fcae4edb2afc7c4c9df45ceb19161`,
and the separately disabled Wi-Fi diagnostic `fbe18c074d63e331209c01132a260f64a8fa9cb1fae4418e53aef962302389f6`.

All P019 technical gates are now qualified on this code checkpoint. The Go
security gate remains open: `GO-2026-6608` is absent, while the exact-source
scan reports three reachable `GO-2026-6809` traces fixed in Go 1.27.3 and one
module-level `GO-2026-5932` finding without a symbol trace. The JS audit keeps
`GHSA-vfj7-8cjw-p6xm` OPEN and reports zero critical findings. PR #23 remains
Draft and unmerged pending the owner's Go 1.27.2 security disposition. P018
Gate F remains pending; no device or release was used.

## Latest exact-source cloud qualification — 2026-10-21

The documentation-only PR head `466a5380` exposed a race in the Wi-Fi helper
test: the fake helper published its PID before writing the `.starts` marker.
The test now waits for the marker. On the following source `af9e158f`, native
run [39586153308](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39586153308)
passed that race test but failed later when a shutdown test immediately tried
to rebind a recently used SSH port on macOS (`address already in use`). The
test now checks that the Agent clears its listener and that new TCP connections
are refused, without relying on immediate TCP port reuse. The focused race
suite passed locally three times; the reconnect/shutdown test passed locally
under race detection ten times.

All required hosted checks passed on exact code source
`a13497171883043a9868a11c428c30b7d86d99eb`:

| Workflow | Run | Result | Evidence |
|---|---:|---|---|
| P017 release-candidate regression | [39586589394](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39586589394) | PASS | Full Go suite and portable builds; Go and Bun/npm security scans; installer/bootstrap and isolation checks; frozen upstream ancestry; genuine frontend build. The canonical frontend manifest has 97 assets and digest `3b80dbea2957210afc99b4f4e2322d682015f53eb6034d163f55d3a9fc2a92e1`. |
| P013 Hub migration fixture | [39586589368](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39586589368) | PASS | Genuine v0.19 migration, restart, backup/restore and interruption recovery; evidence ZIP digest `30aafc741e66d0045b403ba6e7b55360c10e2a26d6954cf2ec0687d4a1fdd617`. |
| iOS installer lifecycle | [39586589321](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39586589321) | PASS | Installer transaction and rollback checks. |
| Helm charts | [39586589440](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39586589440) | PASS | PR validation only; no registry publishing. |
| Native iOS Build | [39586589329](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39586589329) | PASS | Procursus container isolation, macOS portable/race suites, iOS adapter, transport and network-monitor checks, native builds, Mach-O, provenance, checksums and frontend embedding. The release job was skipped. |

The native workflow verified the Go 1.27.2 A7 patch against runtime source
SHA-256 `47f9313d4f956b4159dcf8b292bf340a12adb6b58ee9a94360fe948045e2c5db`
and patched source SHA-256
`c861b98d76ac487269c7f98c987b94ea9978c5f01740a4596eecbfdb9c8bb91f`. It
produced clean ARM64 iOS device binaries with minimum iOS 12.0, `CGO_ENABLED=1`,
`GOARM64=v8.0`, embedded Go VCS revision `a13497171883043a9868a11c428c30b7d86d99eb`,
and `vcs.modified=false`. The Agent digest is
`c9df4951124b3d0d9396ed2c161dd090e6ef21cf7108dd0e8e2266b9ad234ff7`
(9,576,416 bytes); the Hub digest is
`59805bc8ba3c26dc0a2149f432e745fbba1c55fe53408ceef98b65d16506bea8`
(32,076,152 bytes). The Hub embeds all 97 verified frontend assets; no private
MobileWiFi linkage is present.

GitHub artifact metadata for run `39586589329` records the iOS payload ZIP
digest `5cc45dc48b7c988c9843a06297a37a45ccbee9e4eae2296fa127dd7e61a8dd5b`,
native evidence ZIP digest
`f26abf637532461235f82f2ae875f219b607eede56ed9e4d8bb21d35af7691ca`, and
separate disabled Wi-Fi diagnostic ZIP digest
`da7af44eb5ee14292b39d20fe6ed63921aff37dfcaa95a6b9b4519c7ec310f07`. The
exact-source Go scan at `a1349717` finds no `GO-2026-6608`; it reports three
reachable `GO-2026-6809` traces (fixed in Go 1.27.3) and one module-level
`GO-2026-5932` finding without a symbol trace. Bun and npm audits report zero
critical findings and retain `GHSA-vfj7-8cjw-p6xm` OPEN. Bun report SHA-256 is
`5a9adf8fbbd2a854066d8ebb81515723fc033927c4bcee0f7ed3ea13b300efba`; npm
report SHA-256 is
`d2292fdbf69076e249eb3825fb7d034218f93e656e7348b5788dfbe9d868f047`.

P019 technical and hosted CI gates pass on code source `a1349717`. The Go
security disposition remains open, so PR #23 remains Draft and unmerged. No
P020 or P021 branch has been started. The P018 device gate remains pending.

## GO-2026-6809 backport and exact-head qualification — 2026-10-21

Code checkpoint `bc6967dc3cffe2d46be5b427aa490e87227ab571` is pushed and
verified on `ios/p019-upstream-main-b5357343`. Go 1.27.2 still resolves the
required `GO-2026-6608` fix. The later `GO-2026-6809` advisory is fixed upstream
in Go 1.27.3, so P019 carries the official Go CL 851047 backport (commit
`7f86f957633ee31a1c17a1eb8452ca7a3d9bc141`) without changing the pinned Go
version or the A7 runtime patch.

The reviewed patch is stored base64-encoded at
`patches/go/1.27.2/GO-2026-6809.patch.b64`; its decoded SHA-256 is
`111bf066e27cb60df43881cbfa9b1d22b7682b720628b4b691a960a853b48150`. The
fail-closed patcher checks the exact Go 1.27.2 source hashes before applying it,
verifies the complete eight-entry intermediate-SPKI denylist and platform
verification hooks, records all patched source hashes, and runs
`go test crypto/x509 -count=1`. The baseline and patched source fingerprints
are embedded in the P017 and native validation manifests. Local Go 1.27.2
`crypto/x509` tests passed on Linux. Both hosted P017 and native macOS tests
also passed after applying the patch to an isolated GOROOT.

All hosted checks passed on exact code source `bc6967dc`:

| Workflow | Run | Result | Evidence |
|---|---:|---|---|
| P017 release-candidate regression | [39596633842](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39596633842) | PASS | Full Go suite/builds, source and dependency vulnerability scans, Bun/npm audits, frontend build, installer/bootstrap/isolation tests, 41-commit ancestry, and 97-asset frontend manifest. |
| P013 Hub migration fixture | [39596633574](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39596633574) | PASS | Genuine v0.19 migration, restart, backup/restore, and interruption recovery; artifact digest `4b911148fc76dfa69391aaa42dbf63bf681d7b450fe222d0b2947879e5a5bc50`. |
| iOS installer lifecycle | [39596633628](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39596633628) | PASS | Transaction and rollback checks. |
| Helm charts | [39596633614](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39596633614) | PASS | Lint, render and package; no publishing. |
| Native iOS Build | [39596633673](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39596633673) | PASS | A7 runtime patch, x509 backport, transport/race suites, repeated ARM64 iOS builds, Mach-O, minimum OS, provenance, checksums, assets, and linkage. Release job skipped. |

GitHub artifact metadata for these runs records the P017 regression manifest
digest `a8b3531e11130634a1e8a8950a3c2a65995c4af86d8178563c43a38c925bf3b4`, Go
vulnerability evidence digest
`216f102450e227d4e6d805eef1ad4f7e788e2d12293004db7e19da75b21cdf8a`, and
JavaScript audit evidence digest
`0ee40c987cd6b5ebce5c1bda53a26f81033af81495c8668fa55bc6d83cbf73cb`.

The exact-source Go scan used govulncheck v1.9.0 and the database last modified
`2026-10-21T00:27:29Z`. It records four occurrences: three reachable
`GO-2026-6809` traces and one module-level `GO-2026-5932` finding;
`GO-2026-6608` is absent. The machine summary retains all raw findings and gives
`GO-2026-6809` the `backported official Go fix` disposition only with the
verified patch manifest and passing standard-library tests. `GO-2026-5932`
remains module-level without a symbol trace; no OpenPGP imports were found.
Bun and npm report zero critical findings and retain
`GHSA-vfj7-8cjw-p6xm` **OPEN** as required.

The native Agent and Hub artifacts embed VCS revision
`bc6967dc3cffe2d46be5b427aa490e87227ab571` with `vcs.modified=false`. Both are
Go 1.27.2 ARM64 iOS device Mach-O files with minimum iOS 12.0, `CGO_ENABLED=1`,
and `GOARM64=v8.0`. Agent SHA-256 is
`dae2b79ffb4593a3a1bd68d226e2f05dd6879f68ec104227635013ae7cd05cab`
(9,576,424 bytes); Hub SHA-256 is
`307d5dce33acb40ed0876ac6ce6aeefd7ad30d6c83aee21a901708e0feaf784d`
(32,076,160 bytes). The Hub embeds all 97 frontend assets. No private
MobileWiFi linkage was found. GitHub artifact metadata records payload ZIP
digest `cf326a4c3a6bb213cf5316b1795a93ed42da9a56c726fd47a3e8c3b34b48e05d`,
native evidence digest `4f3473313836b29fdbc3a473090d21b32a9227b433933f9f090fdb4c0ffe0303`,
and separate disabled Wi-Fi diagnostic digest
`0eefd25b7456bf4dbac411f560c79da87c5a7c02b7bf551c7d5935a0c537d777`.
The Go 1.27.2 A7 runtime patch changed `runtime/asm_arm64.s` from SHA-256
`47f9313d4f956b4159dcf8b292bf340a12adb6b58ee9a94360fe948045e2c5db` to
`c861b98d76ac487269c7f98c987b94ea9978c5f01740a4596eecbfdb9c8bb91f`; it
replaces the counter-register delay with a zero-guarded `YIELD` loop.

P019 code, security, migration, installer, and native build gates now pass on
`bc6967dc`; documentation changes trigger another final-head CI run before the
PR is marked Ready or merged. PR #23 remains Draft until that run and applicable
review requirements are satisfied. P018 Gate F remains pending; no physical
device, production branch, fork `main`, tag, or release was used.

## P019 exact-head qualification on documentation head `e0357ee9` — 2026-10-21

The final-head checks completed successfully on PR source
`e0357ee9b18960ef21e1bafcec1282206bdd3e56`. GitHub reports the iOS job passed
all native test and artifact steps, and the PR check rollup is clean. The source
still contains the P019 code checkpoint `bc6967dc`; the e035 source change only
updated its qualification and security documentation.

| Workflow | Run | Result | Result details |
|---|---:|---|---|
| P017 release-candidate regression | [39599085633](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39599085633) | PASS | Full Go regression and portable builds, Go vulnerability scan, Bun/npm audits, frontend build, installer/bootstrap/isolation checks, and upstream ancestry. |
| P013 Hub migration fixture | [39599085692](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39599085692) | PASS | Genuine v0.19 fixture migration, restart, backup/restore and interruption-recovery evidence. |
| iOS installer lifecycle | [39599085613](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39599085613) | PASS | Transaction, rollback, bootstrap, patch sentinel and workflow-isolation checks. |
| Helm charts | [39599085595](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39599085595) | PASS | Chart lint, render and package; registry login and publishing were skipped. |
| Native iOS Build | [39599085357](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39599085357) | PASS | Transport/race, network-monitor, macOS, Go 1.27.2 A7 patch, x509 backport, repeated Agent/Hub builds, ARM64 Mach-O, iOS 12 target, provenance, embedded frontend and linkage checks. Procursus container isolation passed; release was skipped. |

The P017 artifact metadata records regression manifest SHA-256
`23a1c5fa3c8d04517321d2e6ea3c80410f0707e66d6aba7a3f91e52e5fe3b4ed`, Go
vulnerability evidence digest
`3e67dca3cd11af78a23c8add9b9e38bec3972ca9d72ef1cde5be7fa09474d8a7`, and
JavaScript audit evidence digest
`ae82e7b52a7ec9d5cba8e92ebbd686bb10dad8b3b95cdb6ac489eedd25d2a6b7`. The
genuine P013 evidence digest is
`f9ef52018ddb74d148eb41dc3bd990ab59212abfa6f59be24714420dc37e7cb3`.

The native Agent is ARM64 iOS device Mach-O, minimum iOS 12.0, and embeds VCS
revision `e0357ee9b18960ef21e1bafcec1282206bdd3e56` with
`vcs.modified=false`. Its SHA-256 is
`e6f9df978b51c3e2360bb2f93743dcc9873967345e1cbf9992b97a6659205287` (9,576,424
bytes). The Hub has the same architecture, deployment target and provenance;
its SHA-256 is
`8e4318b71d92a98f1e6e4a030923f13b9ed4954b931d0c50e378855b2a83718e` (32,076,160
bytes) and it embeds all 97 frontend assets. Both builds use Go 1.27.2,
`CGO_ENABLED=1` and `GOARM64=v8.0`; the A7 runtime source patch and official
Go CL 851047 x509 backport passed. The private MobileWiFi linkage check passed.
GitHub artifact ZIP digests are `2572189feb3580d8ee787d0f3d673a09e52b83f4fc59cc43a8c8c4f91c2779d3`
(review payload), `3706f866295ecb5ad055c76ec29a8317e9ba0d39b72bbf07e35794fc46cc476e`
(native evidence), and `e3c5f92025e7b6d7621df6ba489f44c23608e8f360ea3c4bc90273df8cfd4dea`
(disabled Wi-Fi diagnostic).

PR #23 is clean but remains Draft; there are no submitted reviews or review
requests, and GitHub reports an empty review decision. The repository review
policy endpoint remains unavailable to this account (HTTP 403), so no
independent review or policy exemption is claimed. This documentation
checkpoint will be pushed as a documentation-only commit; its exact PR head
must complete its own hosted checks before the PR can be marked Ready or merged.
