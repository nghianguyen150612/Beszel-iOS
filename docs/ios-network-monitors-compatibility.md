# P010 — iOS network monitors and unsupported collectors

P010 source/fixture checks and the native source-candidate build pass. The final
integration decision requires documentation-inclusive HEAD checks and normal
merge recorded in PR #12. The committed JSON provides exact tested candidates
and workflow results. A commit cannot contain its own SHA; final-head checks and post-merge
verification are recorded in the PR progress record.

## Provenance and scope

Fresh clone first selected `iOS`. Starting integration:
`0c48b1378393b325e9b61ba716f832b1bf807aa6`; product iOS:
`0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; frozen main:
`bb23f99b3eeaced9608e11e280e7c986dba650be`. All three matched before edits.
The frozen main is an ancestor of integration. Matrix/readiness validators
account for all 178 frozen upstream commits and 13 logical iOS patches.
Only `ios/network-monitor-compatibility` targets
`ios/upstream-v021-integration`, through a normal merge commit.

Reviewed the acceptance contracts, P007 native/A7/source-selection report,
P008 telemetry/battery report, P009 transport report, monitor production and
test files, Agent/handler/collector entry points, monitor entities, Hub sync,
version guards, ownership, serialization and storage paths. Disposable fixtures
use temporary directories and loopback endpoints, never `/var/lib/beszel-hub`.

## Architecture and result contract

The Hub synchronizes stable record IDs through replace/upsert/delete requests.
Agent MonitorManager owns immutable monitor tasks. Each task shares a single
in-flight check between immediate and scheduled requests, then records one
sample into short raw history and a fixed 61-slot minute ring. Failures use
`-1` internally and the existing loss/count/sum fields on the wire. Loss-only
windows have zero latency aggregates and 100% loss; those zeros are not measured
successes. Cancelled task results do not enter replacement history. Certificate
inspection runs independently from health verification and publishes metadata
only through existing fields. Resume generation changes discard stale checks.

Hub `MinVersionNetworkMonitors=0.20.0`, `MinVersionCbor=0.12.0`,
`MinVersionAgentResponse=0.13.0`, monitor CBOR integer keys and JSON behavior are
unchanged. No PocketBase migration or production database is changed.

## Confirmed defects and changes

| Surface | Established starting behavior | P010 behavior |
| --- | --- | --- |
| HTTP | Direct helper with nil/unbounded client had no deadline of its own | Maximum 10-second context budget, preserving any shorter caller deadline. |
| DNS configuration | IPv6 normalization, invalid ports and literal-IP cancellation/config shortcuts were not guarded | IPv6/scoped literals normalized; reject unmatched brackets and invalid ports. Validate configuration and cancellation before literal lookup; no system-resolver substitution. Empty answers return an error. |
| ICMP | Native/fallback choice cached forever; a later permission/mode failure could pin a broken choice | Independent IPv4/IPv6 rejected modes and five-minute capability retry; loss/timeout does not become a capability failure. |
| iOS ping | No dedicated GOOS=ios fallback | Procursus Apple network-cmds ping/ping6 argument scheme, executable lookup plus bounded loopback validation before use. Missing/unsupported tools are unavailable, not assumed installed. |
| Ping process | `cmd.Output()` captured unbounded output and could wait on inherited pipes | 64 KiB stdout limit, discarded stderr, 250ms WaitDelay, context kill/reap, no shell or privilege escalation. |
| Immediate publication | Deterministic test returned a result after deletion while certificate I/O was pending | Final publication serializes with task cancellation and returns no result for a cancelled task. |
| Replacement history | Every configuration edit copied the previous series | Interval-only edits retain history; target/protocol/TCP-port/DNS-server/TLS-policy changes reset it. HTTPS metadata is not copied into another protocol. |
| Timing/history | Last arrival could replace latest timestamp; old ring-slot collision could overwrite newer data | Ordered samples, retention anchored to newest timestamp, reject replacement of a newer ring slot by a delayed sample. |
| CBOR snapshots | Real WS test retained 100% loss after success omitted its zero-valued loss key | Fresh typed decode before publication; malformed data leaves previous destination untouched. Applied to modern generic and legacy raw WS payloads; legacy typed fallbacks preserved. |
| Collectors | iOS could discover unrelated GPU tools, smartctl and utility-only ZFS; temperature hook selected macOS SMC/IOKit | Early unsupported constructors, no local pool backend, direct iOS ZFS guards, dedicated absent temperature hook; no automatic local Docker discovery. |

The stale-publication, cancelled DNS-literal and stale-CBOR-loss bugs were reproduced by deterministic
and real transport tests before repair. Other findings are established source
semantics with negative/recovery fixtures, not observed physical iPad failures.
Existing TCP budget splitting, cryptographic ICMP correlation, scheduled probe
sharing, secure/insecure per-monitor transports, certificate retry cadence,
Linux collectors and Hub remote telemetry were retained.

## Probe and permission behavior

TCP resolves within a three-second budget but measures only connection time,
including fallback. Each resolved address gets a share of remaining time; a
blackhole cannot consume the alternatives' whole deadline. IPv4/IPv6, multiple
addresses, refusal, DNS error, empty resolution, invalid port, cancellation,
budget fairness and connection close are covered with local/injected boundaries.

DNS measures lookup latency under three seconds. Empty default selection uses
Go's system resolver. An explicit server always uses the custom resolver Dial
closure, including its chosen numeric port and scoped IPv6 literal. Cancellation and invalid server settings fail before Go’s literal/hosts shortcut. Tests cover UDP
IPv4/IPv6, NXDOMAIN/empty answers, invalid configuration, nonresponsive server,
shorter timeout/cancellation, recovery and a working default resolver that must
not replace a failing custom resolver. Go's normal literal/hosts-file lookup
semantics are retained; this is not an authoritative DNS-only query adapter.

HTTP uses upstream GET, redirects, User-Agent and status >=400 failure semantics;
closes response bodies and honors timeout/cancellation. Local servers exercise
redirect success, error status, invalid URL, connection closure and measured
latency. TLS verifies by default. Opt-out is specific to one monitor/client,
with trusted, self-signed/unknown-CA, hostname mismatch and expired fixtures.
Secure→insecure→secure remains isolated; WebSocket TLS retains independent P009
trust/rotation tests. There is no global TLS bypass.

ICMP retains socket-first selection: raw, datagram, then executable. Root does
not establish socket availability and iOS does not use Linux CAP_NET_RAW.
Permission/protocol/unsupported/not-found errors invalidate a cached choice;
unsupported modes are suppressed for five minutes, then re-detected. IPv4/IPv6
have independent caches. Socket deadlines honor the earlier caller deadline;
cancellation closes the socket, and only source/type/code/ID/sequence/token
matched replies produce measured RTT. Sixteen cryptographic token bytes remain,
protecting sequence wrap and delayed/duplicate replies. No failed socket path
substitutes process elapsed time or a fabricated RTT.

Procursus `makefiles/network-cmds.mk` packages Apple network_cmds-669 ping/ping6;
its build recipe is source justification, not device availability evidence.
The iOS executable path and `-n -c 1` behavior are validated separately against
127.0.0.1/::1 within one second, with bounded output and actual RTT parsing.
Unavailable validation is cached; caller cancellation does not poison capability
state. Actual target probing remains within the original three-second budget.
No shell injection, automatic sudo/root escalation, entitlement or device
configuration change is introduced. Linux/Windows/macOS/BSD argument behavior
remains, and command failure/missing RTT cannot be a success.

## Lifecycle, certificate and collector coverage

Tests cover initial/repeated/empty replacement, incremental updates/delete,
interval and target changes, scheduled/immediate/shared/overlapping probes,
cancellation/publication, resync after WS/SSH reconnect, shutdown/resume,
certificate wait, history preservation/reset, loss transitions and out-of-order
samples. No task/history publication lock spans blocking network I/O. Slow
checks remain serial and drop missed ticks rather than queue unbounded work.
Production probes and certificate checks have deadlines; cancellation releases
timers/sockets/subprocesses. Certificate fixtures cover expiration, rotation,
failed inspection, timeout/cancellation, retry/recovery and one-time metadata
publication. Inspecting an untrusted leaf does not change the health outcome.

See [the capability matrix](ios-collector-capability-matrix.md). Unsupported
local optional maps are absent/empty through existing schemas. Required scalar
defaults do not prove measured zero readings. Shared remote Linux/macOS/Windows
GPU, SMART/NVMe, ZFS/Btrfs, systemd, containers and sensor paths remain.
P011 owns read-only Procursus/APT telemetry and P012 owns Wi-Fi RSSI; their safe
fallbacks remain and no adapter from either milestone is added.

## Validation and acceptance limits

Exact commands/results, Git-derived SHA/tree/blob evidence and native artifact
metadata are in [the validation JSON](ios-network-monitors-validation.json).
Focused probe/ICMP/lifecycle/TLS/collector/protocol tests and relevant race tests
are required, along with canonical tagged Go, P007–P009/audit/shell regressions,
real frozen Bun catalogs/tests/production build, actual Hub embedding and Linux
Agent/Hub builds. iOS/darwin/linux ARM64 CGO file selection uses real go-list
JSON, with mutation fixtures rejecting unsupported selections.

Cloud algorithm/permission/subprocess tests are separate from real sockets.
Real host raw IPv4, datagram IPv4, raw IPv6 and datagram IPv6 integration subtests
all SKIP due EPERM/EACCES. **These SKIPs are unavailable, not PASS.** No physical
iOS socket, current hardware RTT, A7 execution or iPad network reliability is
claimed. P018 / Gate F owns real-device ICMP, trust/DNS/network behavior, Apple
A7 execution and service/reboot acceptance on iPad4,4 / iOS 12.5.7. The historical
0.19 device evidence is unchanged. Optional full Windows Agent compilation has
the known missing LHM assets limitation; portable transport compilation and
remote schema tests are separate from Windows hardware execution.

IOS-AGENT-04 and IOS-AGENT-06 can pass their P010 cloud source/fixture/native-build
gate only after the recorded checks pass and the reviewed normal merge completes.
P010 does not complete Gate C: P011/P012 and applicable physical acceptance
remain. No device installation, jailbreak privilege/entitlement/LaunchDaemon
change, real database access, release job, tag, registry publication or deployment
occurs. Final-head native runs and post-merge ref/tree checks belong in
[PR #12](https://github.com/nghianguyen150612/Beszel-iOS/pull/12).

## Executed source-candidate evidence

Final production-source local candidate:
`dac735270cf8caf85ef8d8473e24b49e2fd7b743`. Canonical Go passes, as do the full
selected-package races (Agent, battery, delta tracker, entities, Hub WS/SSH,
systems), three final-source P010 race repetitions, ten DNS/correlation and
lifecycle repetitions, five focused ICMP repetitions and three TLS/protocol
repetitions. There are 23 new portable P010 test functions plus an iOS-only ZFS
test selected with `testing`; that iOS-only test is not counted as a host or
physical-device execution. All 49 audit tests, matrix/readiness, upstream sync
10, iOS sentinels 45, bootstrap 54 and installer 814 assertions pass. Frozen Bun
1.4.0 catalogs, 44 tests and genuine production build pass. Windows WS/SSH test
binaries compile; full Windows Agent LHM assets remain a separate known limit.

Native source-candidate [iOS Build 38657535072](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38657535072)
and [installer 38657538922](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38657538922)
PASS at `14a778f1ad56bd8f7e251f9e61b196f6725dc953`. Later DNS/ZFS repairs require
final-head runs; these earlier candidate runs alone do not authorize merge.
The independent download verifier confirms both ARM64 MH_EXECUTE binaries,
iOS platform 2/minimum 12.0, CGO=1/GOARM64=v8.0, clean matching VCS metadata,
Go 1.27.1, checksums and all 95 real frontend files embedded in the native Hub.
The runner used Bun 1.4.0, Xcode 15.4, iPhoneOS SDK 17.5 and SDK clang, isolated
patched GOROOT, unchanged original runtime hash, guarded YIELD/no counter
access in both linked binaries, and byte-identical independent-cache builds.
Release job: **SKIPPED**. Candidate Agent SHA256:
`a824edd2afcc8be5acd754905cef07c58987f27afd8e79ffa097827f1a8653f4`;
Hub: `f880491c82c03ecf5d5bd1b87c9d020af388f01b04b4c9d1f79398ed4a38354c`.
Both ZIP bytes match GitHub's uploaded artifact digests. The initial CLI storage
redirect returned Forbidden; the connected GitHub artifact download path
succeeded, without changing network policy, credentials or TLS verification.

IOS-AGENT-04 / IOS-AGENT-06: source/fixture acceptance **PASS**; native candidate
acceptance **PASS**. P010's final integration gate is **PASS only when** the
final-head CI/review/normal merge and post-merge checks recorded in PR #12 all
succeed. Final runs, hashes and the merge SHA are recorded there to avoid
self-referential evidence commits. P018 hardware procedures remain not run.
