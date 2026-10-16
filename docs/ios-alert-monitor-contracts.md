# P014 alert and monitor contracts

All execution uses disposable synthetic PocketBase records and fake test mailers
or loopback notification endpoints. No real user notifications are sent.

## Alert policy

CPU, memory, disk, bandwidth, load, CPU state, temperature, GPU and battery use the
repository's supported threshold alerts. Battery thresholds are inverted.
Status, container health, SMART, ZFS, systemd and network monitor alerts have their
existing separate handlers. There is no invented Wi-Fi, fan or package alert.

Unavailable optional telemetry must hold an existing incident until fresh valid
data can evaluate recovery. Missing GPU data is not measured zero GPU usage.
Zero battery with a known charge state is a valid reading. Repeat unchanged
incidents follow the existing transition policy. Notification delivery rechecks
user/system access; revoked membership must prevent delivery.

## Monitor policy

ICMP, TCP, DNS and HTTP/HTTPS use existing probe/fixture suites. TLS verifies by
default; `skipTlsVerify` belongs to the individual HTTP monitor. Failed windows
use loss/counts and no successful latency. Certificate metadata remains separate.

Monitor results are attributed only to an existing monitor owned by the reporting
system. Deleted monitors must not gain records from delayed results. Duplicate
probe identities do not create history. Agent clock rollback is supported by
P010, so a timestamp alone cannot safely identify an old task response. Newer
sample-count ordering is being validated without changing that clock policy.

Weighted history uses total/success counts and response sum; loss-only windows
produce null frontend latency. Reconnect resends the entire current configuration,
including an empty set. Old Agents below the monitor version boundary receive no
unsupported monitor actions.

Execution status and exact evidence belong in `ios-hub-monitoring-validation.json`;
this contract does not itself assert passing tests.
