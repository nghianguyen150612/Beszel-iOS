# P027 first implementation batch — 2026-11-10

Verified baseline: `2c4e0db3da1b2c074d8b3155ce3eb9179fdf40a3`.
Official upstream: `3392dcc6f0c9e3b7858a0c377577a4b8a40b8b8d (2026-11-09T14:27:13-05:00)`.
Development branch: `ios/p027-upstream-port`.

P027 preserves P026 and the v0.21 integration ancestry. Production iOS remains
0c3e4cd22ac24264834c401ad412e44aeda4e4f3; P026 remains unmerged (PR #30).
The new PR targets ios/p026-device-acceptance, never production.

Fork main is 85d168c385907293c116814590bc468c82783154, five commits behind
official main. The latest common ancestor of P026 and upstream is
b97abf7662a7c7a2dd7542ea2062405abd22d4dc. There are 33 upstream ancestry
commits beyond that point. See ios-p027-upstream-ledger.csv for full SHAs,
classifications and pending dispositions. An ancestry delta is not a claim
that every patch is missing: P023 already supplies Go 1.27.3.

Implemented upstream sources:
- 2de112436d99c6147838e1b85a106e329e3b74ae: authorize destination of monitor moves.
- cd9ab245463ba5ae801400b4092ed76f2c93c664: admin-only system sharing updates.
- 9b11e7f8051af5835541f10a41e7739b55f37d71: preserve shared users in edit form.
- 7f8b0bb2eeb866dc4e27f265d64d9871d33e00fe: retain pending WebSocket during slow authentication.
- 9760e6ec524eda4beeea82d35c8a0f9d7f8b6fab: first-run password character/byte limits.

The WebSocket patch retains P026's locked timeSinceConnectAttempt accessor,
not upstream's unsynchronized field read. Existing telemetry, byte-memory,
A7 runtime patches, installer, rootful packaging, migrations and evidence are
unchanged. Existing API authorization and WebSocket regressions were ported.
The iOS Build PR trigger now includes the stacked P026 target; upstream
synchronization workflows and schedules are untouched.

Next batch: qualify x/net 0.62.0, PocketBase 0.40.6, CBOR and shoutrrr
upgrades (5750cead); run full synthetic migration harness before accepting
PocketBase changes. Review upstream Battery json/v2, swap telemetry and
no_self_update against existing iOS adapters/build contracts. Then remaining
portable alerts, frontend and agent changes in the ledger. Linux RAID, dnf,
systemd and container collectors must not replace iOS implementations.

Baseline remote CI: latest production sync succeeded (run 42770543783);
Upstream Drift Check failed (42709285871). Neither qualifies this candidate.
No PR-triggered runs returned for the P026 head at audit time. Historical
P026 evidence remains authoritative for its original source and is preserved.

Device qualification: **DEFERRED — OWNER APPROVED**. No SSH/device access,
production service/database access, deployment, installation, migration on a
device, release/tag, production branch update, merge or watcher change occurred.
Host validation and exact results are recorded below before final handoff.

Implementation commits and modified files (documentation commit follows):

```text
a7a9edc177c29cf558c588724c44dd922c7136a5 ci(p027): validate stacked P026 pull requests with existing iOS gates
348984e865ba6df21e4a3bbbda20adcfd8d2c458 fix(hub): enforce password char and byte limits on first-run form (#2744, #2745)
8ce571b59a864c99559c4586e25cfca6c6217840 fix(agent): preserve pending WebSocket during slow hub connection (#2772)
75708a8dcd0834a473e5e2a547b1af3e729d98eb fix(site): don't overwrite system users when editing a system
6bc77c658dce5650a87c301fa86190c16c7ed6cc fix(hub): only allow admins to change system users
ac6f73f025d7e8e468e44cf793c2849c49e7f1df fix(hub): verify write access to target system when moving a network monitor (#2714)

.github/workflows/ios-build.yml                  |  2 +-
 agent/connection_manager.go                      |  4 ++
 agent/connection_manager_test.go                 | 61 ++++++++++++++++
 internal/hub/collections.go                      |  5 +-
 internal/hub/collections_test.go                 | 76 ++++++++++++++++++--
 internal/hub/network_monitors.go                 | 17 +++++
 internal/hub/network_monitors_test.go            | 88 ++++++++++++++++++++++++
 internal/site/src/components/add-system.tsx      |  7 +-
 internal/site/src/components/login/auth-form.tsx |  1 +
 internal/users/users.go                          |  6 ++
 internal/users/users_test.go                     | 53 ++++++++++++++
 11 files changed, 312 insertions(+), 8 deletions(-)
```
