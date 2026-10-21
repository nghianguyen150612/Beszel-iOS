# P019 Pinned Upstream Merge Review

Status: P019 code, security, and hosted CI gates passed on code source
`bc6967dc`; final documentation-head checks remain required before merge.

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

The reconstructed P019 code checkpoint
`bc6967dc3cffe2d46be5b427aa490e87227ab571` passed the full hosted P017 Go
suite/build, vulnerability and Bun/npm audits, frontend build,
installer/bootstrap/isolation checks, frozen upstream accounting, genuine
P013 migration/recovery fixture, installer lifecycle, Helm validation, and
native iOS ARM64 validation. Exact run IDs, scanner disposition, artifact
digests, and native binary checksums are recorded in
`docs/ios-p019-checkpoints.md`.

The Go 1.27.2 source scan retains three reachable `GO-2026-6809` traces, fixed
upstream in Go 1.27.3. P019 backports official Go CL 851047 to the isolated
toolchains used by P017 and native iOS CI; exact source fingerprints and
passing `crypto/x509` tests qualify that mitigation. `GO-2026-6608` is absent.
The module-level `GO-2026-5932` finding remains recorded without a symbol
trace. Bun/npm report zero critical findings and keep
`GHSA-vfj7-8cjw-p6xm` **OPEN**.

The native artifacts embed VCS revision
`bc6967dc3cffe2d46be5b427aa490e87227ab571` with `vcs.modified=false`. They are
ARM64 iOS device Mach-O files with minimum iOS 12.0, Go 1.27.2,
`CGO_ENABLED=1`, `GOARM64=v8.0`, and 97 embedded frontend assets. The no-private-
MobileWiFi linkage check passed. The release job was skipped.

All code gates pass on `bc6967dc`. This documentation update triggers the
required workflows again; merge is held until they pass on the final PR head
and applicable review requirements are satisfied. PR #23 remains Draft while
those checks run. No independent review is claimed. The review-policy endpoint
returned GitHub 403; the PR currently shows no review requests, no submitted
reviews, and an empty review decision. The repository permissions API reports
the authenticated account has admin access, but no branch protection exemption
has been used.
