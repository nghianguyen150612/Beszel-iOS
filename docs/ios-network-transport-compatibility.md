# P009 — Network transport compatibility

**P009 source/fixture/native-build gate: PASS** at `7472f3e86ab8b5825d64cf5ef461c8dae98491ba`.
Final documentation-inclusive HEAD checks and the integration merge decision are
recorded in PR #11. Source, loopback integration, fault injection and race checks pass.
[PR #11](https://github.com/nghianguyen150612/Beszel-iOS/pull/11) records the final
tested SHA and merge verification without a self-referential evidence commit.
Gate C remains pending P010–P012. Physical A7 execution remains P018 / Gate F;
database migration/restore remain P013/P017. No P010 implementation is included.

## Provenance and preservation

Starting integration: `212227f53d1e14ae164e0b5be77b88680b3566e0`.
Product `iOS`: `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`.
Frozen `main`: `bb23f99b3eeaced9608e11e280e7c986dba650be`.
The mandatory fresh clone selected the case-sensitive `iOS` branch; the original
workspace checkout was preserved. All three refs matched, with no intervening
integration commits. Work uses `ios/network-transport-compatibility`, targeting
only `ios/upstream-v021-integration` with ordinary commits and a normal merge.
All **178/178** canonical upstream commits are reachable and **13/13** logical
iOS patches remain accounted for. P001–P008 evidence files remain unchanged.
The JSON records the exact upstream commit-set hash and patch IDs.

## Production architecture and behavior before repair

The Agent's existing connection manager prefers outbound WebSocket. TCP and HTTP
upgrade are followed by a CBOR fingerprint challenge: the Agent verifies the
Hub's SSH signature over its token, sends its fingerprint and becomes ready.
The handler registry rejects protected requests before verification. CBOR uses
numeric keys, optional request IDs, generic `AgentResponse.Data` and legacy typed
fields. Requests and replies remain bound to their original connection.

When WebSocket is unavailable, the Agent starts its SSH listener unless
`DISABLE_SSH=true`. Without `HUB_URL`, SSH-only operation is supported.
Authenticated SSH TCP connections are counted once across multiple sessions.
Healthy SSH stops further WS attempts under the existing policy; an already
pending WS attempt may authenticate and take over. WS readiness stops the obsolete
SSH server. SSH loss leaves/restarts the listener and resumes WS retries.

Before P009, WS auth/close callbacks synchronously sent to a one-slot event queue;
startup/ticker dialing blocked the event loop; setup and retry sleeps were not
cancellable by shutdown. The custom proxy dialer had no explicit TCP timeout.
`wss://` was rewritten to `ws://`. Readiness was announced before fingerprint
write success. Request response lookup, enqueue and removal were separate, and
any malformed/no-ID response could be routed to the oldest pending request.
Remote close did not immediately cancel the request manager. SSH cancellation
already closed the captured shared client safely, but `Close()` missed setup
before the client was published.

## Confirmed defects and focused repairs

| Finding | Repair and regression |
| --- | --- |
| WSS downgraded to plaintext; arbitrary URL schemes accepted | Preserve WS/WSS and HTTP/HTTPS conversions; reject other schemes without printing URL secrets. `TestP009WSSPreservesTLS`, `TestP009EndpointSchemes`. |
| Auth/close callback blocked while the event loop stopped consuming | Coalesced nonblocking WS notification, followed by current-owner/state re-evaluation. Full-queue regression reproduced before repair. |
| Setup blocked shutdown; a queued connect ignored cancellation | Async manager attempts joined at stop; cancellable proxy-aware setup and connection slot, 5s TCP / 10s overall setup bounds, cancellable retry delay, stopped health/retry tickers. Both shutdown and queued-cancel regressions pass. |
| Failed fingerprint write announced readiness | Publish verification/readiness only after successful fingerprint send and a second connection identity check. Broken-write regression reproduced. |
| A superseded handler could use another connection's verified flag | Verify the carrying connection and reject late auth/close callbacks. Explicit rejection tests cover both handler and signature paths. |
| Duplicate responses could block and abandoned responses retain pooled data | Atomically claim pending requests, nonblocking delivery, cancellation-ordered draining, release invalid/unroutable messages, cancel pending requests on close, reject sends after shutdown. |
| Oldest-request selection could associate a no-ID reply with the wrong parallel request | Serialize peers below `MinVersionAgentResponse`; retire their connection after an abandoned request so late replies cannot satisfy a successor. Modern peers retain concurrent IDs. |
| SSH close left an unpublished handshake alive; concurrent callers performed redundant handshakes | Serialize cancellable setup and cancel it on transport close; check cancellation before publication. Keep captured-client cleanup and one shared transport owner. |
| Control whitespace could be trimmed before header validation | Reject forbidden controls before normalization; bound configuration at 64 KiB. Preserve comma/CR/LF separators, duplicate values, colons in values and canonical reserved-header protection. |
| Existing SSH test leaked fixed-port listeners and masked its negative key case | Bind dynamic ports synchronously, pin an ephemeral host key, join cleanup and offer an actually unauthorized key. No production key checks were relaxed. |

The pre-repair failure logs are hashed in the JSON. Changes preserve the
connection manager architecture, upstream cryptographic algorithms and P004–P006
fixes. No transport dependency, entity schema or migration changed.

## Fault injection, authentication and request ownership

The tests use actual Agent handlers/manager, Hub `WsConn`/request manager and
SSH transports. Loopback endpoints inject upgrade rejection, immediate close,
challenge silence, invalid signature, malformed CBOR, text frames, broken writes,
idle deadlines, repeated drops and recovery. DNS, TCP timeout and refusal errors
are deterministic dial-boundary injections; they are not Internet DNS tests.
A production ten-second retry is observed after an asynchronous upgrade failure.
Concurrent connect/close, shutdown during setup and old-connection callbacks are
covered. Two uncached stats replies are also collected from real Linux Agent
processes, with receipt timestamps recorded; synthetic recovery counters prove
that each subsequent response replaces the previous sample.

Request tests cover 64 concurrent out-of-order replies, cancellation/delivery
races, late replies, duplicates, ID wrap over an existing pending request,
malformed frames, send failure, close/shutdown and legacy serialization.
Separate connections deliberately use the same numeric ID and cannot receive
each other's result. Pending maps return to zero and abandoned pooled messages
are released. Modern response timeout keeps a healthy connection reusable;
legacy timeout/cancellation retires the ambiguous connection. Cancellation while
a socket write is blocked closes that captured socket to release I/O.

The actual Hub header/token/fingerprint and authorization suites pass, including
bad/missing tokens and malformed versions. Agent `TOKEN` takes precedence over
`TOKEN_FILE`; the latter permits one token, ignores blanks/comments and rejects
multiple tokens without logging them. Reserved `X-Token`/`X-Beszel` are controlled
by Beszel after case normalization, even with repeated override attempts.
Header errors omit values. The legacy empty-token-file behavior is preserved;
the Hub rejects an empty required token.

## SSH security, deadlines and fallback

Existing tests cover silent banner/handshake timeout, session timeout, blocked
shell/write/read/exit, per-request cancellation, concurrent sessions, shared
connection failure/recovery, old-client cleanup and late cancellation after
replacement. New tests cover close during setup and wrong pinned host-key
rejection followed by successful recovery. Actual Agent public-key rejection and
acceptance are tested. A request cancellation intentionally closes its captured
shared SSH connection; other pending sessions fail and a later request reconnects.
Version initialization remains inside the owner lock before publication.

**Upstream trust-policy limitation:** the Hub's existing
`createSSHClientConfig()` uses `ssh.InsecureIgnoreHostKey()`, and Agent host keys
are generated automatically. P009 preserves this upstream default; it does not
claim production SSH host-key pinning. Strict supplied verification callbacks
are exercised and enforced. Introducing persistent host-key provisioning would
change that compatibility/configuration contract and is outside these repairs.
Agent authorization still requires the configured Hub public key. Disposable
subprocess smoke tests pin their first observed ephemeral loopback key for that
process only; this is not evidence of production server identity trust.

## TLS, custom roots and iOS trust

Verification remains on by default. With no `CA_CERT_FILE`, Go's default trust
path is used. A custom file extends `x509.SystemCertPool()`; it never enables
`InsecureSkipVerify`. Missing/unreadable files, invalid PEM and unavailable/nil
system roots fail clearly. Tests use generated disposable CAs and leaf keys:
valid IP/DNS identities, multiple CAs, expired certificates, unknown issuer,
hostname mismatch, wrong EKU, rotation and corrected trust configuration.
The per-monitor TLS exception remains separate and cannot configure WS TLS.

HTTP/HTTPS/WS/WSS paths and `BESZEL_AGENT_ALL_PROXY` handling remain compatible;
the existing environment proxy selection is retained with bounded cancellable
TCP setup. There is no direct-network retry after a proxy/TLS failure. TLS state
is cloned per attempt. No real credentials or production certificates were used.
A live publicly issued Internet certificate was not tested: generated local
roots exercise trust mechanics, while default public trust is delegated to Go.

Go 1.27.1 selects Darwin-family x509/Security-framework verification for iOS.
A native link cannot demonstrate which trust APIs/certificates are available to
a jailbroken iOS 12 command-line process. Actual default/public trust-store
behavior, CA file permissions and API availability remain physical P018 work.
Explicit `CA_CERT_FILE` remains available; no private Apple API was introduced.

## Slow polling and monitor convergence

Upstream timings remain unchanged: 30s WS stats request, 100s Hub connection
silence deadline, 120s Agent deadline and 3m stale-peer cutoff. Existing scaled
poll tests prove that slow collection and a request timeout retain a healthy
connection, idle ping/pong works, prolonged silence eventually expires the peer,
and later data is decoded into a fresh struct. The isolation repair disables SSH
only in a fixture that intentionally has no SSH manager; fallback policy is
unchanged. No tight reconnect loop or indefinitely pinged stale peer is added.

Full monitor replacement is still requested on WS connect and SSH reconnect,
with failed sync retried after successful stats. Existing Hub tests cover initial,
changed, deleted/disabled and empty sets and failed-first-sync convergence. New
real Agent SSH → WS tests cover duplicate delivery retaining one task, SSH
reconnect, updated config and empty replacement surviving another WS reconnect.
The probe is inert/injected: native probe behavior and ICMP permissions remain
P010. Monitor actions remain suppressed below Agent 0.20.0.

## P003 pairing evidence

The canonical P003 CSV remains unchanged. Supplementary P009 evidence uses its
precise scenario IDs; none of these rows is promoted to a physical iOS pass.

| Scenario | Available P009 evidence | Remaining exact acceptance |
| --- | --- | --- |
| COMP-PAIR-001, iOS 0.19 ↔ iOS 0.19 | Existing historical device record retained | Historical reference only; not rerun |
| COMP-PAIR-002, iOS 0.19 → upgraded Hub | Real product-commit 0.19 Agent compiled/run on Linux against current WS/SSH transports | Exact iOS/native Hub execution P017/P018 |
| COMP-PAIR-003, upgraded Agent → iOS 0.19 Hub | Current Linux Agent against unchanged product-commit 0.19 WS/SSH Hub transport code | Exact iOS Hub/device execution P017/P018 |
| COMP-PAIR-004, upgraded iOS pair | Current Linux binary/real transport integration; both iOS binaries compiled and verified | Physical iOS pair and full DB/API acceptance P013/P017/P018 |
| COMP-PAIR-005, Linux 0.21 → iOS Hub | Real Linux Agent and shared Hub transport integration | Native iOS Hub API/history/UI P014/P017/P018 |
| COMP-PAIR-006, Windows 0.21 → iOS Hub | Shared source/schema regressions; Windows WS/SSH test binaries cross-compiled | Native Windows Agent/runtime and iOS Hub P014/P017/P018; full binary blocked by existing missing LHM assets |
| COMP-PAIR-007, macOS 0.21 → iOS Hub | Shared source and hosted macOS transport/race tests and Agent/Hub builds pass | Exact remote macOS → physical iOS Hub P014/P017/P018 |
| COMP-PAIR-008, older remote Agent → upgraded Hub | Legacy/no-ID, one-shot SSH, malformed/unknown-version and typed/generic response fixtures | Exact lowest supported remote versions P017 |

`MinVersionCbor=0.12.0`, `MinVersionAgentResponse=0.13.0` and
`MinVersionNetworkMonitors=0.20.0` are unchanged. Legacy SSH emits exactly one
JSON/CBOR response, modern responses retain generic and typed fields, and fresh
partial replies do not retain stale data. JSON v1/v2 zero/absent and remote
payload regressions from P008 remain passing. Linux runs do not establish iOS
trust, kernel behavior or all eight native pairings.

## Validation and resource bounds

Canonical Go suite and all four applicable transport race packages pass with
`env -u NETWORK` and tags `testing no_ui`. Fault tests were repeated ten times, including three consecutive failed
upgrades before recovery and malformed/empty/duplicate response rejection;
queued-connect/SSH fixture repairs separately ten times, and strengthened stale
signature/cross-system assertions twenty times under race detection. All audit,
readiness/matrix, 49 Python tests, sync/sentinels/bootstrap and 814 installer
assertions pass. P007 build-target/A7 source fixtures, source selection and P008
telemetry/battery regressions pass. Linux Agent/Hub build with clean VCS metadata.
Bun 1.4.0 frozen install, catalogs, 44 tests and production build pass; all 95
frontend assets are genuine and the lock hash is unchanged.

There is one manager connection attempt, one WS owner, one serialized SSH setup
and one shared SSH client per transport. WS retries retain the 10s ticker and
5s minimum-attempt policy; setup is bounded by 10s with 5s TCP, and shutdown
cancels the retry delay. Health/retry tickers are stopped. Request cleanup retains
at most one queued message per request and removes abandoned pending entries.
The stress fixtures exercise 12 recovery cycles per repetition, 64 concurrent
requests and 200 response/cancel races per repetition. Fatal fixture assertions run outside monitor locks; listener cleanup is bounded
and TLS sockets are registered for failure-path cleanup. No persistent leak or
race was observed in these bounded tests. This is a source/test bound, not a physical
A7 CPU, power, memory or reconnect-storm measurement.

## Hosted native evidence and gate

Source-candidate native run:
[38559144606](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38559144606),
SHA `7472f3e86ab8b5825d64cf5ef461c8dae98491ba`.
Installer [38559149068](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/38559149068): PASS.
Native artifact validation: **PASS**. Both downloaded ZIPs match GitHub's upload
digests. Independent Linux-side payload validation confirms ARM64 MH_EXECUTE,
iOS platform 2/minimum 12.0, Go 1.27.1, CGO=1, clean source-candidate VCS metadata,
checksums and all 95 embedded frontend assets. Native Agent/Hub repeat builds are
byte-identical; unstripped runtime instruction checks find the guarded YIELD
loop and no counter-register access. The source toolchain remains unchanged;
the runner uses its own patched GOROOT and the iPhoneOS SDK. Release job skipped.
Exact payload hashes and runtime/source reports are in the JSON. The final tested
feature SHA and final-head workflow URLs belong in PR #11 and the final report.

IOS-AGENT-05 has passing cloud source, fixture and portable integration evidence.
Native Agent/Hub acceptance passes at the source-candidate level. Normal merge
requires final-head checks and review recorded in PR #11; the final SHA is
recorded outside its own evidence commit.
Physical iPad connectivity, intermittent jailbreak networking, iOS DNS/socket
behavior, trust-store access and reboot/reactivation remain A7-DEVICE-01 / P018.
No production database or device was accessed or modified. No release, tag,
deployment or production-branch change is authorized or performed by P009.
