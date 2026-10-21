# P021 final integrated cloud review

## Scope and exact base

P021 independently reviews the unified P001–P020 source after P020 merged at
`42556e62c23f3595a276d2eb4a55125a96f5d31f`. Its first parent is P019 merge
`c8bd3dba8656c8195ffacf56ff743dd58b01da25`; its second parent is the P020
feature head `c716d3c9ce8de170cade25b3921bccef8c5b97a7`. P019 and P020 were
normal merges. Pinned upstream remains
`b5357343550c8e2493d0e65472a06c1007396384`.

P021 does not add another upstream merge or unrelated features. Code review
covered Agent and Hub telemetry contracts, iOS missing-data behavior, CPU
breakdown order, A7 runtime source selection, battery parsing, connection and
network-monitor lifecycles, authentication and ownership, TLS expiry alerts,
Hub migrations/history, embedded frontend serving, installer transactions,
native-build provenance, and CI/release boundaries. P020 closed the two
confirmed qualification gaps: universal-token GET URL mutation/secrets and
missing deterministic CPU breakdown order coverage. The P021 review found no
additional product-code change justified. P021 adds `docs/ios-p021-*` to the
P013 migration workflow's pull-request path filter so every documentation-only
P021 checkpoint still runs the genuine v0.19 migration and restore fixture.
This focused CI trigger correction does not change migration or application
behavior.

## Historical ancestry and patch accounting

At the integration base the pinned upstream is reachable and all 41 commits
after `bb23f99b3eeaced9608e11e280e7c986dba650be` are present. The read-only
P002/P003 validators report 178/178 frozen upstream-only commits accounted,
35/35 historical iOS commits mapped, all 13 logical iOS patchsets present, and
zero unresolved audit rows. P021's exact-head CI is required to reconfirm these
invariants after the final source is pushed.

## Review areas

| Area | Reviewed contract and evidence | P021 exact-head status |
|---|---|---|
| Agent metrics | CPU/RAM/disk/network fields, omissions vs zero, counter reset/recovery; `agent/ios_metrics_test.go`, `agent/cpu_test.go`, `agent/system_test.go`, P017 and native suites | Pending |
| CPU behavior | Socket/IRQ/SoftIRQ/Nice ordering, history, and deprecated fields; `TestCpuBreakdownContract`, `internal/records/records_averaging_test.go` | Pending |
| A7 and battery | Exact Go 1.27.2 runtime source signature, fail-closed patcher, ARM64 target; recursive ioreg battery parser and fixtures | Pending |
| Agent lifecycle | Cancellation, reconnect, timeout, callback publication, network-monitor scheduling/history and unsupported iOS collectors | Pending |
| Hub authorization | OAuth role preservation, ownership relationship rules, universal token authorization and GET URL rejection | Pending |
| Alerts and monitors | TLS certificate expiry/rotation, monitor identity, history, scheduling and resynchronization | Pending |
| Database | Genuine disposable v0.19 upgrade, migration order, restart idempotence, role/account/SSH identity, WAL history, backup/restore and interruption recovery | Pending |
| Frontend | Built and embedded assets served from `internal/site/embed.go`; metrics normalization and monitor UI tests | Pending |
| Installer | Rootful and Procursus paths, bootstrap checksum/pin, LaunchDaemon order, transaction rollback, redaction, and no unauthorized purge | Pending |
| Native build | A7 patch, Go 1.27.2, ARM64 device Mach-O, iOS 12 minimum, provenance/checksums, frontend embedding, private MobileWiFi absence | Pending |
| Security | Go 1.27.2 scan/backport, Bun/npm audit and retained advisories | Pending |
| CI/release boundary | PR workflows nonpublishing; release job tag-gated; production/fork refs and devices untouched | Pending |

The P020 regression matrix and security review provide the detailed file/test
crosswalk. P021 runs the full P017 and native workflows, the genuine P013
fixture, and the explicit installer lifecycle workflow against the P021 source.
The results at this initial checkpoint are pending and are not inferred from
P020.

## Review disposition

P020's unresolved security advisories remain explicitly open and are assessed
in `docs/ios-p021-security-disposition.md`. P018 physical Gate F remains
PENDING. Cloud builds do not claim device validation. No production branch,
fork `main`, release tag, GitHub release, or deployment is authorized.
