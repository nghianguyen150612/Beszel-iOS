# P014 alert and monitor contracts

Execution uses disposable synthetic PocketBase records and fake test mailers or
loopback notification endpoints. No real user notifications are sent.

## Alert policy

CPU, memory, disk, bandwidth, load, CPU state, temperature, GPU and battery use the
repository's supported threshold alerts. Battery thresholds are inverted. Status,
container health, SMART, ZFS, systemd and network monitor alerts use their existing
handlers. No Wi-Fi, fan, latency or package alert type was invented.

Unavailable optional telemetry holds an existing incident until fresh valid data
can evaluate recovery. Missing GPU data is not measured zero GPU usage. Zero battery
with a known charge state and a fresh zero temperature remain valid observations.
Per-sensor and per-filesystem alert history uses only samples for that sensor or
filesystem. Notification delivery rechecks user/system access; revoked membership
prevents delivery.

The P014 Hub integration triggers a CPU threshold incident through the real alert
manager. P014 focused tests also cover missing GPU data, fresh zero recovery, and
sparse temperature/filesystem history. The full existing alert package suite passes
its supported battery, CPU, memory, disk, bandwidth, SMART, ZFS, systemd, container,
monitor, ownership and delivery regressions. Fake delivery is used; no production
destination is configured.

## Monitor policy

ICMP, TCP, DNS and HTTP/HTTPS use the existing protocol implementation and synthetic
peers. TLS verifies by default; `skipTlsVerify` belongs to each HTTP monitor.
Failed windows preserve total/success/loss counts and carry no successful latency.
Certificate metadata remains separate.

Monitor results are attributed only to an existing monitor owned by the reporting
system. Deleted monitors cannot gain records from delayed results. Sample sequence
ordering rejects repeated or stale task results even if the Agent wall clock moves
backward; older peers without a sample count retain timestamp deduplication. A
reconnect sends the entire current configuration, including an empty set. Older
Agents below the monitor version boundary receive no unsupported monitor actions.

History averages use probe counts and response sums. Loss-only windows have null
frontend latency; empty windows create no history or fabricated zero latency. P014
Hub, systems, transport, records, alert, API and frontend tests passed, including
three race repetitions for monitor ordering, immediate result lifetime, cancelled
poll rejection and sparse alert history. Exact commands and CI evidence are in
`docs/evidence/p014/final-validation.md`.
