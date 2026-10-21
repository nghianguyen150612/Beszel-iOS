# P019 Pinned Upstream Merge Review

Status: P019 technical qualification passed on source `de40fbe8`; merge is
held pending an acceptable Go 1.27.2 security disposition.

## Source and ancestry

- First parent: P019 audit checkpoint `52967f47c91b82d9ea7455a541ea7c7ca099a19d`, whose parent is integration `f9d6500c7a68572dc8e464dd305d79bf6c1e79a9`.
- Second parent: pinned upstream `henrygd/beszel` `main` at `b5357343550c8e2493d0e65472a06c1007396384`.
- Common base: frozen v0.21.0 commit `bb23f99b3eeaced9608e11e280e7c986dba650be`.
- Upstream range: all 41 commits after the frozen baseline.
- The tracked iOS patch map has 13 distinct `logical_patch_id` values. Source sentinels and full ancestry checks remain required after the merge commit.

## Explicit conflict decisions

The normal merge preview found ten content-conflict files. Each was resolved against the P018/P017 iOS contracts and the pinned upstream implementation:

- `agent/client.go`: retain the cancellable `ConnectContext` and iOS connection lifecycle notifications; use the upstream proxy-aware dialer and emit upstream WebSocket state events as well. The connection remains bound to the Agent lifetime while setup has a finite deadline.
- `agent/connection_manager.go`: keep asynchronous cancellable connection attempts, lifecycle cleanup, package-update shutdown, network-monitor scheduling and health markers. Adopt the upstream `Start(ctx, ...)` API and parent-context shutdown. Use the locked connect-attempt timestamp accessor and keep retry handling from the integration side.
- `agent/agent_test_helpers.go`: retain the P018 synchronized state accessor and cancel hook, and add the upstream synchronized accessor.
- `agent/system.go`: preserve the iOS exclusion around Darwin `cpu.Info()` while adding upstream CPU socket reporting on platforms where the probe is supported.
- `agent/server_test.go`: keep upstream listener and cleanup coverage, with a short temporary socket path for Darwin. Cloud runs set `TMPDIR` inside the authorized workspace.
- `agent/p009_manager_test.go`: update the existing P009 tests to pass the new caller context introduced upstream.
- `go.mod` and `go.sum`: select upstream Go 1.27.2 and newer dependency versions at overlapping requirements; retain the dependency checksum union until `go mod tidy` validation.
- `internal/hub/agent_connect_test.go`: use upstream context-aware test-agent cleanup and synchronized state access, while retaining iOS SSH isolation and deterministic polling.
- `internal/site/src/components/routes/system/charts/disk-charts.tsx`: use the integration's shared `lib/disk` helpers, which preserve missing-data and older telemetry semantics, instead of duplicating the upstream helpers in the component.
- `internal/users/oauth_test.go`: retain upstream OAuth default-role and bootstrap-admin tests, plus the integration tests for invalid identities, duplicate provider identities and role-preserving account linking.

The merge also expands the review-only PR workflow path filters to run on the P019–P021 code, test and evidence changes. The Go version assertion now follows the exact `go.mod` pin rather than the superseded 1.27.1 value.

## Cloud test isolation

The prior-session incidents reported in the task were not reproduced. The cloud test environment uses `.task/` for `HOME`, `TMPDIR`, Go, Python, Bun and npm caches. `BESZEL_HEALTH_DIR` routes Linux health markers into `.task/health` so local Agent tests do not touch `/dev/shm/beszel_health`.

The Procursus read-only APT fixture does not invoke `bwrap`. Its required Linux case runs in the hosted Ubuntu workflow inside an official Go 1.27.2 container with `--network none`, a read-only root, a read-only checkout/module cache, and a writable container-local `/tmp` for disposable fixture data. The test fails if the workflow does not explicitly confirm that isolation. The hosted health marker is redirected to container `/tmp`; it never uses `/dev/shm/beszel_health`.

## Qualification and merge status

The P019 code source `de40fbe8d3b5782d979bfbc2cbd76c5944b46764` passed the
full Go suite, affected race suites, Bun tests, Lingui, TypeScript, Vite,
installer/bootstrap checks, upstream accounting, P013 migration and restore
fixture, Procursus isolation test, and native iOS ARM64 validation. See
`docs/ios-p019-checkpoints.md` for exact run IDs, manifests, binary checksums,
and artifact digests.

The Go scan confirms `GO-2026-6608` is absent, but reports three reachable
`GO-2026-6809` traces in Go 1.27.2; the fixed Go release is 1.27.3. The
module-level `GO-2026-5932` finding remains recorded without a symbol trace.
The Bun/npm audits report zero critical findings and keep
`GHSA-vfj7-8cjw-p6xm` **OPEN**. P019 cannot merge until its Go security
disposition is acceptable. PR #23 remains Draft and unmerged; no review was
claimed or impersonated.

This checkpoint update changes documentation only. The native Agent and Hub
artifacts embed VCS revision `de40fbe8d3b5782d979bfbc2cbd76c5944b46764` with
`vcs.modified=false`; the tested source tree is unchanged by this documentation
update. That is the immutable-source evidence boundary for any documentation-
only commit that follows.
