# P023 checkpoints

## Checkpoint 0 — verified baseline

- Verified integration source: `fd334076f53d8354a4924a4c734f29cf65a43879`.
- P022 final source tree is identical to the integration tree.
- Created `ios/p023-security-hardening` from that integration SHA; the remote ref
  currently points to the baseline SHA.
- Verified P022 PR #26 is merged and Gate G passed.
- Verified production `iOS`, fork `main`, and the pinned upstream commit match the
  requested SHAs. Live upstream `main` is five commits ahead; excluded from P023.
- Initial audit and machine-readable baseline are in
  `ios-p023-security-audit.md`, `ios-p023-advisory-matrix.json`, and
  `ios-p023-validation.json`.
- P023 qualification has not run yet. Gate H is pending; P018 Gate F remains
  pending. No public release is authorized.

- Initial baseline commit `48308ecd44198ac79cb827c8ccbeacb62688dd3c` was pushed
  and independently verified as the remote branch SHA.
- Draft PR [#27](https://github.com/nghianguyen150612/Beszel-iOS/pull/27) is
  open against `ios/upstream-v021-integration`.
