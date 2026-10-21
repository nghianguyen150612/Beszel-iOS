# P020 durable checkpoint ledger

## Base and scope

P020 starts from the verified P019 integration merge
`c8bd3dba8656c8195ffacf56ff743dd58b01da25` on
`ios/upstream-v021-integration`. P019 PR #23 was merged with a normal merge
commit. The P019 feature source was `81e6a5e779efd61ac464b9835eb1b1658d4263d8`;
the merge has parents `f9d6500c7a68572dc8e464dd305d79bf6c1e79a9` and that feature
head. Pinned upstream `b5357343550c8e2493d0e65472a06c1007396384` is an ancestor.

The P019 merge passed its exact-head Go, security, genuine migration, installer,
Procursus, Helm, and native iOS checks. Post-merge structural validation
reported 41 pinned upstream commits, 178/178 upstream audit rows, 35/35 frozen
iOS commits, 13 logical iOS patchsets, zero unresolved rows, and 45 passing iOS
patch sentinels. The upstream sync sentinels passed 10/10 cases. Detailed P019
evidence is in `docs/ios-p019-checkpoints.md` and
`docs/ios-p019-security-review.md`.

P020 branch: `ios/p020-post-upstream-qualification`.
Target: `ios/upstream-v021-integration`.
P020 PR: not created at this initial checkpoint.

## Qualification plan

Review the merged code for regressions in Agent/Hub telemetry and connection
contracts, iOS missing-data behavior, CPU metrics/history, network monitors and
TLS alerts, WebSocket/SSH reconnect, OAuth and ownership authorization, token
security, Hub migration/history, embedded frontend assets, installer
transactions, and the native iOS build. Add focused deterministic tests or
corrections only where an uncovered risk is confirmed.

Required P020 evidence files are planned as:

- `docs/ios-p020-integration-review.md`
- `docs/ios-p020-regression-matrix.csv`
- `docs/ios-p020-security-review.md`
- `docs/ios-p020-validation.json`
- `docs/ios-p020-checkpoints.md`

## Current state

This is the pushed initial P020 checkpoint. The regression review and focused
validation have not yet been performed on the P020 branch. No P020 CI result or
merge is claimed. The branch must be pushed and verified before the Draft PR is
created, then all applicable checks must pass against the final P020 source.

No device, production `iOS` branch, fork `main`, release tag, or published
release is in scope. P018 physical-device Gate F remains pending.

## P020 security correction checkpoint

The post-upstream review found that `GET /api/beszel/universal-token` still
accepted `token`, `enable`, and `permanent` query parameters. Although the
current frontend already uses POST, the legacy GET handler and its test
permitted a universal token to appear in the request URL. P020 now keeps GET
read-only and rejects each of those query fields with a 400 response. Token
issuance, mutation and token comparison remain available through the existing
POST JSON body, with `Cache-Control: no-store` unchanged.

Focused coverage now proves a token query and the two mutation query fields are
individually rejected; the response omits the supplied token, no permanent
record is created, and a follow-up GET remains inactive. Existing POST tests
continue to cover activation, persistence, authorization and ownership.

Local Go 1.27.2 validation on the correction source:

- `go test -tags 'testing no_ui' ./internal/hub -run '^(TestApiRoutesAuthentication|TestUniversalTokenEndpoint)$' -count=1 -timeout=5m`: PASS.
- `go test -race -tags 'testing no_ui' ./internal/hub -run '^TestUniversalTokenEndpoint$' -count=3 -timeout=5m`: PASS.
- `git diff --check`: PASS.

The correction checkpoint was pushed as `2ad963066aa918e86158e31c0b60d65bacb2218e` and `git ls-remote` confirmed the same remote SHA. The focused tests above passed. The remaining post-upstream review and exact-head hosted regression, migration, and native iOS checks are still required.
