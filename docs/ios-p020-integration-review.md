# P020 post-upstream integration review

## Scope and source

- Integration base: `c8bd3dba8656c8195ffacf56ff743dd58b01da25` (P019 normal merge).
- Implementation checkpoint: `2ad963066aa918e86158e31c0b60d65bacb2218e`.
- Initial full qualification source: `5819a82b8d628b66411335643c767e1e147de93d`.
- Latest implementation source: `e7d3d36041d4bb49eeeb15ee4dcae2fe4d7d1391`; exact-source hosted requalification passed.
- Branch: `ios/p020-post-upstream-qualification`.
- PR: [#24](https://github.com/nghianguyen150612/Beszel-iOS/pull/24), targeting `ios/upstream-v021-integration`.

P020 reviewed the newly integrated Agent, Hub, frontend, database, installer,
and native-build paths against the existing compatibility and security tests.
The one confirmed regression was a legacy universal-token GET endpoint that
still accepted secret and mutation query parameters although the frontend had
moved to POST. P020 makes GET read-only, rejects `token`, `enable`, and
`permanent` query keys with HTTP 400, and adds isolated deterministic API cases
for each key. POST JSON issuance and mutation remains covered by the existing
endpoint tests.

## Review results

| Area | Existing or added deterministic coverage | P020 assessment |
|---|---|---|
| Agent/Hub protocol and telemetry | `agent/network_monitor_protocol_compatibility_test.go`, `internal/hub/transport/monitor_compatibility_test.go`, `internal/hub/serialization_contract_test.go` | Old/new CBOR/JSON fields, omitted values, and unavailable telemetry remain explicitly tested. |
| iOS collectors and missing values | `agent/ios_metrics_test.go`, `agent/cpu_test.go`, `agent/system_test.go`, `internal/site/src/lib/metrics.test.ts` | Missing data remains distinct from measured zero; recovery after collection failure and counter reset has fixtures. |
| CPU topology and history | `agent/system_test.go:TestCpuSockets`, `internal/hub/systems/system_test.go:TestCreateRecordsSavesCpuSockets`, `internal/hub/systems/system_test.go:TestCombinedData_MigrateDeprecatedFields`, `internal/records/records_averaging_test.go` | Socket persistence, deprecated-field migration, and history averaging are covered. P020 adds `TestCpuBreakdownContract` to pin the Agent telemetry order for User/System/Iowait/Steal/Idle/IRQ/SoftIRQ/Nice. |
| Connection lifecycle | `agent/p009_client_test.go`, `agent/p009_manager_test.go`, `agent/lifecycle_test.go`, `internal/hub/ws/p009_request_test.go`, `internal/hub/transport/ssh_*_test.go` | WebSocket/SSH cancellation, retry, stale response, host-key and timeout paths have deterministic coverage. |
| Network monitors and TLS | `agent/network_monitor_lifecycle_compatibility_test.go`, `agent/network_monitor_history_test.go`, `agent/network_monitor_cert_test.go`, `internal/alerts/alerts_network_monitor_cert_test.go`, `internal/hub/systems/network_monitor_stats_test.go` | Scheduling, cancellation, history, persistence, certificate rotation and expiry alert lifecycle are covered. |
| OAuth and ownership | `internal/users/oauth_test.go`, `internal/hub/api_settings_ownership_test.go`, `internal/alerts/alerts_access_test.go`, `internal/hub/collections_test.go` | Role-preserving OAuth linking, relationship immutability, ownership and access rules are covered. |
| Universal-token security | `internal/hub/api_test.go:TestUniversalTokenEndpoint`, `internal/hub/agent_connect_test.go:TestUniversalTokenFlow` | P020 adds URL-query rejection cases; existing tests cover POST activation, persistence, authorization and agent authentication. |
| Migration, history and recovery | `internal/migrations/snapshot_integration_test.go`; P013 hosted migration fixture | Exact P020 source passed the genuine v0.19 migration, restart, full restore and interruption-recovery fixture. |
| Frontend and embedded assets | `internal/site/src/lib/metrics.test.ts`, monitor comparison tests; P017 and native workflows | Exact P020 source passed frontend tests/build and embedded 97 verified assets in the Hub. |
| Installer transaction | installer/bootstrap tests and `ios-installer` workflow | No installer source changed in P020; explicit workflow dispatch `39609283025` passed on `e7d3d36`. |
| Native iOS | `ios-build.yml` | Exact source passed ARM64/iOS 12/A7 runtime, race, provenance, checksum, linkage and artifact validation. |

## Local correction checks

On P020 implementation source `2ad963066aa918e86158e31c0b60d65bacb2218e`
with Go 1.27.2:

- `go test -tags 'testing no_ui' ./internal/hub -run '^(TestApiRoutesAuthentication|TestUniversalTokenEndpoint)$' -count=1 -timeout=5m`: PASS.
- `go test -race -tags 'testing no_ui' ./internal/hub -run '^TestUniversalTokenEndpoint$' -count=3 -timeout=5m`: PASS.
- `gofmt` and `git diff --check`: PASS.

## Exact-head hosted results

On exact code source `e7d3d36041d4bb49eeeb15ee4dcae2fe4d7d1391`, P017
regression `39609195215`, P013 migration `39609195286`, native iOS
`39609195413`, Procursus isolation, and installer dispatch `39609283025` all
passed. The Go 1.27.2 A7 runtime source and x509 backport checks passed; the
Agent and Hub were built twice with isolated caches and validated as iOS ARM64
device Mach-O files with iOS 12.0 minimum deployment, exact VCS provenance,
matching SHA256SUMS, 97 embedded frontend assets, and no private MobileWiFi
linkage. The native release job was skipped. Binary and artifact hashes and
the security findings are in `docs/ios-p020-checkpoints.md`,
`docs/ios-p020-security-review.md`, and `docs/ios-p020-validation.json`.

The 5819a82 runs are retained above as historical evidence only; the current
implementation is qualified by the e7d3d36 runs. A new documentation-only
checkpoint is required before merge; its final PR-head checks must also pass.

The Procursus isolation test ran on the hosted workflow's network-disabled,
read-only container. P020 did not run `bwrap` in the local cloud workspace.
