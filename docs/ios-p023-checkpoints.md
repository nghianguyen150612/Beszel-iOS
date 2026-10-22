# P023 checkpoints

## Checkpoint 0 — verified baseline

- Integration source: `fd334076f53d8354a4924a4c734f29cf65a43879`.
- P022 final source tree matched the starting integration tree; PR #26 is
  merged and Gate G passed.
- Production `iOS`, fork `main`, and pinned upstream SHAs were verified.
  Live upstream main was five commits ahead; those commits remain excluded.
- Created `ios/p023-security-hardening`, pushed the initial baseline, and
  opened Draft PR [#27](https://github.com/nghianguyen150612/Beszel-iOS/pull/27)
  against `ios/upstream-v021-integration`.

## Checkpoint 1 — Go 1.27.3 and security implementation

- Candidate implementation commit
  `c9e68e1858bd71792ac79193257c95e406eb4c87` is pushed and independently
  verified as the remote branch SHA.
- Go 1.27.3 is pinned across `go.mod`, Go Actions, native iOS builds, and six
  digest-pinned Docker Go builder stages.
- The A7 runtime source guard is refreshed to exact Go 1.27.3 source.
  The x509 backport transition is validated against the native Go fix.
- Go scans keep the GO-2026-5932 module-level finding and require
  source-level OpenPGP non-reachability evidence. JavaScript scans retain the
  GHSA and use full npm-lock OSV queries only after the exact registry audit
  timeout.
- Local results include a serial full Go suite PASS, 117 audit tests PASS,
  crypto/x509 PASS, frontend PASS, six OpenPGP profiles PASS, and full-lock
  OSV scan PASS with the reviewed advisory retained.

## Checkpoint 2 — exact-head hosted qualification

GitHub workflows were triggered on
`c9e68e1858bd71792ac79193257c95e406eb4c87`:

- iOS Build: run 39715287619
- Installer lifecycle: run 39715287691
- P013 Hub migration: run 39715287632
- P017 security/regression: run 39715287614

At this checkpoint, iOS Build, installer, migration, and P017 runs were
in progress. Gate H remains pending until the exact PR head passes, release
jobs are confirmed skipped, and the native review archive is independently
checked. See [`ios-p023-validation.json`](ios-p023-validation.json) for status.

## Preservation and scope

The candidate retains the 41 pinned upstream commits, 178 frozen audit rows,
35 historical iOS commits, 13 logical iOS patchsets, and P019–P022 normal
merge ancestry. Exact candidate workflow validators will confirm these
counts.

No production `iOS` branch, fork `main`, database, device, tag, or public
release has been changed or accessed. No sudo was used. The two release-owner
security decisions remain open; P018 Gate F remains pending.
