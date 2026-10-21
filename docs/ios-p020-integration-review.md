# P020 post-upstream integration review

## Scope and source

- Integration base: `c8bd3dba8656c8195ffacf56ff743dd58b01da25` (P019 normal merge).
- Reviewed implementation source: `2ad963066aa918e86158e31c0b60d65bacb2218e`.
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
| CPU topology and history | `agent/system_test.go:TestCpuSockets`, `internal/hub/systems/system_test.go:TestCreateRecordsSavesCpuSockets`, `internal/hub/systems/system_test.go:TestCombinedData_MigrateDeprecatedFields` | Socket persistence and deprecated-field migration are covered. CPU IRQ/SoftIRQ/Nice collection is platform-selected and reviewed in the native/source-selection workflow. |
| Connection lifecycle | `agent/p009_client_test.go`, `agent/p009_manager_test.go`, `agent/lifecycle_test.go`, `internal/hub/ws/p009_request_test.go`, `internal/hub/transport/ssh_*_test.go` | WebSocket/SSH cancellation, retry, stale response, host-key and timeout paths have deterministic coverage. |
| Network monitors and TLS | `agent/network_monitor_lifecycle_compatibility_test.go`, `agent/network_monitor_history_test.go`, `agent/network_monitor_cert_test.go`, `internal/alerts/alerts_network_monitor_cert_test.go`, `internal/hub/systems/network_monitor_stats_test.go` | Scheduling, cancellation, history, persistence, certificate rotation and expiry alert lifecycle are covered. |
| OAuth and ownership | `internal/users/oauth_test.go`, `internal/hub/api_settings_ownership_test.go`, `internal/alerts/alerts_access_test.go`, `internal/hub/collections_test.go` | Role-preserving OAuth linking, relationship immutability, ownership and access rules are covered. |
| Universal-token security | `internal/hub/api_test.go:TestUniversalTokenEndpoint`, `internal/hub/agent_connect_test.go:TestUniversalTokenFlow` | P020 adds URL-query rejection cases; existing tests cover POST activation, persistence, authorization and agent authentication. |
| Migration, history and recovery | `internal/migrations/snapshot_integration_test.go`; hosted P013 migration fixture | The hosted fixture is the evidence for real v0.19 upgrade, restart, restore and interruption recovery; current exact-head run is pending. |
| Frontend and embedded assets | `internal/site/src/lib/metrics.test.ts`, monitor comparison tests; P017 and native workflows | Metrics semantics and genuine embedded frontend build are covered; current exact-head workflow is pending. |
| Installer transaction | installer/bootstrap tests and `ios-installer` workflow | No installer source changed in P020; P019 exact-source installer workflow passed. The P020 PR does not claim a new installer run unless one is triggered and completed. |
| Native iOS | `ios-build.yml` | P019 exact-source ARM64/iOS 12/A7 qualification passed. P020 native workflow is running against `2ad9630`; merge waits for its final result. |

## Local correction checks

On P020 implementation source `2ad963066aa918e86158e31c0b60d65bacb2218e`
with Go 1.27.2:

- `go test -tags 'testing no_ui' ./internal/hub -run '^(TestApiRoutesAuthentication|TestUniversalTokenEndpoint)$' -count=1 -timeout=5m`: PASS.
- `go test -race -tags 'testing no_ui' ./internal/hub -run '^TestUniversalTokenEndpoint$' -count=3 -timeout=5m`: PASS.
- `gofmt` and `git diff --check`: PASS.

## CI boundary

GitHub Actions runs are attached to the commit SHA that their job checked out.
The security audit artifacts and current P020 hosted jobs are recorded in
`docs/ios-p020-validation.json` and `docs/ios-p020-checkpoints.md`. The P020
Draft PR is not merge-ready while any required final-head check is queued or
running. No result from an earlier P020 checkpoint is substituted for a later
source SHA.

The Procursus isolation test must use the hosted workflow's network-disabled,
read-only container. P020 does not run `bwrap` in this local cloud workspace.
