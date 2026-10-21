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
are running on pushed source `50ae2d8f4bc63fdeb1eeb2bc9f436059e5170846`.

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
environment. Commit `50ae2d8f4bc63fdeb1eeb2bc9f436059e5170846` is pushed and its
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
