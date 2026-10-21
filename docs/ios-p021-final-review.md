# P021 final integrated cloud review

## Scope and exact base

P021 independently reviews P001–P020 after P020 merged normally at `42556e62c23f3595a276d2eb4a55125a96f5d31f`. P019 is normal merge `c8bd3dba8656c8195ffacf56ff743dd58b01da25`; P020's merged feature head was `c716d3c9ce8de170cade25b3921bccef8c5b97a7`. Pinned upstream is `b5357343550c8e2493d0e65472a06c1007396384`.

Review covered Agent and Hub telemetry contracts, iOS missing-data behavior, CPU breakdown order, A7 runtime selection, battery parsing, connection and network-monitor lifecycles, authorization and ownership, TLS expiry alerts, Hub migrations/history, embedded frontend serving, installer transactions, native provenance, and CI/release boundaries. P020 addressed universal-token GET URL secret/mutation parameters and added deterministic CPU breakdown contract coverage. No further product implementation change was justified. P021 added path-only CI routing so documentation checkpoints run the P013 fixture and installer lifecycle suite.

## Ancestry and patch accounting

On base `42556e62c23f3595a276d2eb4a55125a96f5d31f` and P021 candidate `b20ed2b636d95bca26c87161de06d8e1e7f03862`, the pinned upstream is reachable and all 41 commits after `bb23f99b3eeaced9608e11e280e7c986dba650be` are present. Validators report 178/178 frozen upstream-only commits reviewed, 35/35 historical iOS commits mapped, 13/13 logical iOS patches, and zero unresolved audit rows. Native CI passed `tests/ios-patch-sentinels.sh` and `tests/upstream-sync-test.sh` on `b20ed2b`.

## Qualification matrix

| Area | Exact candidate evidence | Result |
|---|---|---|
| Agent metrics and missing data | P017 run `39620427058`; native telemetry/race suites `39620426859` | PASS |
| CPU behavior/history | P017 full Go suite, native race tests; focused P020 `TestCpuBreakdownContract` | PASS |
| A7 runtime and battery | Go 1.27.2 exact-source runtime patch, fail-closed signature, battery parser; P017 and native run `39620426859` | PASS |
| Agent lifecycle | Cancellation, reconnect, read timeouts, callback publication, transport races; native run `39620426859` | PASS |
| Network monitors | Scheduling, cancellation, recovery, history, certificate expiry/rotation; P017 and native run `39620426859` | PASS |
| Unsupported collectors | iOS source selection, Linux-only collector exclusion, platform fallbacks; native run `39620426859` | PASS |
| Hub authorization | OAuth role preservation, protected ownership, universal-token authorization and GET URL rejection; P017 run `39620427058` | PASS |
| TLS alerts | Expiry, rotation/recovery, alert cleanup and access controls; P017 run `39620427058` | PASS |
| Database | Genuine disposable v0.19 upgrade, ordered migrations, identity/history, backup/restore and interruption cases; P013 run `39620427023` | PASS |
| Frontend | P017 tested/built frontend; 97 assets; native embedding; runs `39620427058` and `39620426859` | PASS |
| Installer | Rootful/Procursus, bootstrap integrity, service order, transaction rollback, redaction and no purge; installer run `39620426909` and Procursus job in native run | PASS |
| Native build | Go 1.27.2 A7 patch, ARM64 iOS device Mach-O, iOS 12.0, provenance, checksums, embedding and no private MobileWiFi; run `39620426859` | PASS |
| Security | Go/Bun scans, x509 backport, retained advisories; P017 `39620427058`, native `39620426859` | PASS WITH DOCUMENTED OPEN ADVISORIES |
| CI/release boundary | PR workflows nonpublishing, release job skipped, protected refs verified, no device/tag/release/deployment | PASS |

## Artifact and source consistency

P013, P017, native, and installer run metadata identify candidate head `b20ed2b636d95bca26c87161de06d8e1e7f03862`. The P017 manifest reports the same source, 97 frontend assets, and no device test/tag/release. Native validation reports clean VCS metadata, ARM64 iOS device Mach-O, iOS 12.0 minimum, 97 embedded assets, verified checksums, and no private MobileWiFi linkage. Exact binary SHA-256 values and artifact digests are in `ios-p021-validation.json`.

## Test disposition

P017 run `39618985621` on `9ef6d85` exposed a timeout-test race: the 100 ms test deadline could fire before the helper wrote its PID marker, and the repeated loop did not require a fresh marker each time. The test was corrected at `b20ed2b` to await each distinct start and verify that exact process is reaped. The focused final Go 1.27.2 race test passed locally; full P017 and native test suites passed on `b20ed2b`. No product implementation failure remains.

## Review and merge disposition

`GO-2026-5932` and `GHSA-vfj7-8cjw-p6xm` remain open and are assessed in `ios-p021-security-disposition.md`; zero critical JavaScript findings were reported. The Go x509 and A7 runtime patch tests passed. No critical unresolved regression was identified. The final evidence head must rerun and pass P013, P017, native iOS/Procursus, and installer workflows before merge. Their exact head/run IDs are recorded on PR #25; no check may be bypassed. No independent review is claimed; PROMPT021 supplies the owner's explicit review-policy exception.

P018 physical Gate F remains PENDING. Cloud checks do not claim device validation. Production `iOS`, fork `main`, tags, releases, and deployment remain unchanged.
