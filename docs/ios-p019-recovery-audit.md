# P019 Recovery and Upstream Integration Audit

Status: reconstruction in progress. This checkpoint records verified starting state; it is not a passing qualification.

## Recovery decision

The prior session reported P019 merge `ede04de3`, source commit `9c1cf38dfbac82172d4600b80e2820f5ebcf117a`, and documentation commit `7d1ec806903b6ec36b44bea911ccc9b4c8a8489f`. None of these objects exists in the current repository object database or reflogs. GitHub has no P019 branch or pull request, and searches of the connected repository for each reported commit ID returned no result. The old implementation is classified as **UNRECOVERABLE UNPUSHED LOCAL WORK**. This branch is a clean reconstruction with new commit identities; no claim of byte-for-byte equivalence is made.

## Verified starting refs

- Repository: `nghianguyen150612/Beszel-iOS`
- P019 base: `ios/upstream-v021-integration` at `f9d6500c7a68572dc8e464dd305d79bf6c1e79a9`
- Frozen v0.21.0 baseline: `bb23f99b3eeaced9608e11e280e7c986dba650be`
- Pinned upstream `henrygd/beszel` `main`: `b5357343550c8e2493d0e65472a06c1007396384`
- The baseline is an ancestor of the pinned upstream, with 41 upstream commits after it.
- The baseline is an ancestor of the P019 integration base.
- The integration base contains the P001–P018 history and the tracked patch map `docs/upstream-v021-ios-patch-map.csv`.
- At start, fork `main` was `e984ee12081f4e112601c3f718c9d81c112fe718` and production `iOS` was `0c3e4cd22ac24264834c401ad412e44aeda4e4f3`; neither is a target of this milestone.

## Integration method and open gates

The pinned upstream history will be merged with a normal merge commit. Conflicts will be reviewed file by file against the iOS patch map and the frozen P002 audit records. No blanket upstream-wins conflict resolution is permitted. The 13 logical patchsets, native ARM64/iOS 12 build behavior, iOS runtime source signature guard, installer safety, and preserved historical records must be rechecked after the merge.

Tests, security review, native macOS CI, migration/restore evidence, artifact/source consistency, and final ancestry counts remain pending at this checkpoint. No result is marked PASS by omission.

## Inherited incident context

The task reports that an earlier session opened and truncated `/dev/shm/beszel_health` and that Procursus isolation tests invoked `bwrap`. Those incidents are recorded as prior-session events; they were not repeated during this reconstruction. Tests that could write outside the authorized workspace or invoke privileged namespace isolation will be audited before execution, corrected where feasible, or run in the designated hosted CI environment. P018 physical-device Gate F remains pending.
