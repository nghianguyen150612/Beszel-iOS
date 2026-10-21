# P021 durable checkpoint ledger

## Base and scope

P021 starts from the verified P020 normal merge commit
`42556e62c23f3595a276d2eb4a55125a96f5d31f` on
`ios/upstream-v021-integration`. P020 PR #24 merged its final documentation-only
head `c716d3c9ce8de170cade25b3921bccef8c5b97a7`; its parents are the P019
integration `c8bd3dba8656c8195ffacf56ff743dd58b01da25` and that P020 head.
P019 remains a normal merge at `c8bd3dba8656c8195ffacf56ff743dd58b01da25`.
Pinned upstream is `b5357343550c8e2493d0e65472a06c1007396384`.

P021 performs a final independent cloud review of P001–P020. It does not merge
another upstream range or add unrelated features. The source review found no
additional justified code change after P020's token URL correction and CPU
breakdown contract test. All P021 exact-head hosted qualification is pending
at this first checkpoint.

Branch: `ios/p021-integration-closeout`.
Target: `ios/upstream-v021-integration`.

## Upstream and historical accounting at the base

On base `42556e62`, the pinned upstream is an ancestor and the exact frozen
baseline range contains 41 upstream commits. The read-only P002 matrix validator
reports 178/178 upstream commits reviewed, 35/35 historical iOS commits mapped,
13 logical iOS patches, and zero unresolved audit rows. The P003 readiness
validator reports the same 178 commits and 13 patches with all 24 acceptance
cases and 33 compatibility scenarios present. P021's hosted exact-head checks
must repeat the applicable source and patch sentinels.

## P020 evidence carried into the final review

P020's code source `e7d3d36041d4bb49eeeb15ee4dcae2fe4d7d1391` passed P017
run `39609195215`, P013 run `39609195286`, native iOS/Procursus run
`39609195413`, and installer dispatch `39609283025`. Final P020 PR head
`c716d3c9ce8de170cade25b3921bccef8c5b97a7` changed only the five P020 evidence
files; its P017 `39611915082`, P013 `39611915079`, native iOS/Procursus
`39611915076`, and installer `39611969234` checks passed. P020 merged as
`42556e62c23f3595a276d2eb4a55125a96f5d31f`. Exact security dispositions and
native artifact hashes are in the P020 security and validation records.

## P021 initial checkpoint

This initial checkpoint contains the independent review, acceptance matrix,
security disposition, validation record, and this ledger. The P021 source SHA
and PR number will be recorded after the checkpoint is pushed and its Draft PR
is created. P017 full regression, genuine P013 database fixture, native iOS,
Procursus, and explicit installer dispatch must all run against the P021 source.
No result is inferred from P020 runs. Any final documentation-only P021 head
must also pass the required checks before merge.

## Boundaries

Cloud results do not constitute device validation. P018 physical Gate F remains
PENDING; native Wi-Fi RSSI remains DEFERRED. No physical device is accessed.
Production `iOS`, fork `main`, tags, releases, and deployment remain unchanged
and out of scope.
