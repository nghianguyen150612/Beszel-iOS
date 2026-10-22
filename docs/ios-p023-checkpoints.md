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
  `f7a4525aeb7513163439b56b330ddfbd527c4da8` is pushed and independently
  verified as the remote branch SHA.
- Go 1.27.3 is pinned across `go.mod`, Go Actions, native iOS builds, and six
  digest-pinned Docker Go builder stages.
- The A7 runtime source guard is refreshed to exact Go 1.27.3 source.
  The x509 backport transition is validated against the native Go fix.
- Go scans keep the GO-2026-5932 module-level finding and require
  source-level OpenPGP non-reachability evidence. JavaScript scans retain the
  GHSA and use full npm-lock OSV queries only after the exact registry audit
  timeout.
- P017 run 39715287614 on source `7f980ea` correctly failed because the npm
  summary validator assumed one affected package node. Review of the retained
  raw npm report showed five affected package entries for one braces advisory;
  P023 tightened validation to the exact graph and retained the single root
  advisory without changing dependencies.
- Local results include a serial full Go suite PASS, 123 audit tests PASS,
  crypto/x509 PASS, frontend PASS, six OpenPGP profiles PASS, full-lock OSV
  scan PASS with the reviewed advisory retained, and 36 release workflow
  isolation checks PASS.

## Checkpoint 2 — GitHub Actions supply-chain hardening

All 19 unique action versions across 54 workflow use sites are pinned to
upstream-resolved commit SHAs. New regression tests require full SHAs. P023
narrowed Helm write permission to a push-only job, disabled reusable release
and native build caches, moved shell outputs through environment variables,
and stopped checkout from persisting write tokens. The two workflows that
push with explicit secrets use short-lived `GIT_ASKPASS` helpers.

The initial Zizmor scan found 72 static findings: 51 mutable action refs, 9
credential persistence findings, 6 cache exposure findings, 4 broad permission
findings, and 2 shell interpolation findings. After remediation,
`actionlint v1.7.12` passed and `zizmor v1.31.1` returned zero findings. P017
repeats these checks on the exact PR source and uploads raw evidence.

## Checkpoint 3 — exact-head hosted qualification

The first hosted cycle on `304cc862e79ee821080e2b217da1009e945aa9d2`
passed installer and Hub migration, and Helm validation passed with its publish
job skipped. P017 failed its hosted actionlint step; its raw workflow evidence
artifact is 12399702486. The iOS build had not completed before the workflow fix
was pushed.

The follow-up source commit `f7a4525aeb7513163439b56b330ddfbd527c4da8` starts a fresh exact-head cycle:

- iOS Build: run 39717756410 (in progress)
- Installer lifecycle: run 39717756362 (success)
- P013 Hub migration: run 39717756382 (in progress)
- P017 security/regression: run 39717756411 (in progress)
- Helm workflow validation: run 39717756365 (success; both validation and publish jobs skipped on PR)

Local actionlint passes and local Zizmor reports zero findings after the fix.
Gate H remains pending until the final PR head passes all mandatory runs,
release jobs are confirmed skipped, and the native review archive is
independently checked. See
[`ios-p023-validation.json`](ios-p023-validation.json) for status.

## Preservation and scope

The candidate retains the 41 pinned upstream commits, 178 frozen audit rows,
35 historical iOS commits, 13 logical iOS patchsets, and P019–P022 normal
merge ancestry. Exact candidate workflow validators will confirm these
counts.

No production `iOS` branch, fork `main`, database, device, tag, or public
release has been changed or accessed. No sudo was used. The initial local test
invocation created short-lived Python temporary fixtures outside the authorized
workspace; they were automatically cleaned. The subsequent successful run used
workspace-local HOME, TMPDIR, and Go caches. This boundary incident is recorded
in the machine-readable validation file. The two release-owner
security decisions remain open; P018 Gate F remains pending.
