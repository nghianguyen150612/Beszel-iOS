# P026 checkpoint ledger

This ledger separates exact cloud evidence from physical-device evidence. Hosted CI and artifact verification cannot satisfy Gate F.

## Checkpoint 0 — source and workflow identity

Recorded 2026-10-24 UTC.

- P025 PR #29 is merged at bb945dc603ba2ae1673bb22866cc929d037fb9dd.
- Current ios/upstream-v021-integration is the same SHA and tree 8806c06d89881c9b41d5839babde84ca850dbd82.
- P025 workflow 39782135426 completed SUCCESS on that exact merge SHA.
- The build, Procursus update check, and separate artifact-consumer verification jobs succeeded. Release validation and publishing were skipped.
- P025 Gate J is PASS for the frozen merge source and exact-run artifacts.
- P018 PR #22 remains Draft at head 8e951a005dbd684f1b3e09c35a27ed1d08001d14. Its old source branch is used only for the reviewed read-only preflight helper; no P018 source branch was merged.
- No later integration source update has superseded the frozen candidate.

## Checkpoint 1 — Phase B artifact verification and local staging

Completed 2026-10-24 UTC. No device transfer occurred.

- Downloaded exact-run payload artifact 12428152331, native validation artifact 12428676214, and consumer manifest artifact 12428267122.
- Independently calculated ZIP digests match GitHub artifact metadata. Exact values and sizes are in ios-p026-artifact-transfer.md.
- Verified payload SHA256SUMS. Agent SHA-256 is 3db9aa10205add17318e832ad561604bb8d0e23466d90f5da9a8ffecbf32ec42; Hub SHA-256 is e746ea219c5e30ec2e749bf1d1c247f473d00ae2b49c31d0575ba132d0c6f202.
- Host file inspection reports Mach-O 64-bit arm64 executables. Exact-run metadata reports minimum iOS 12.0 and source revision bb945dc603ba2ae1673bb22866cc929d037fb9dd.
- Unsigned originals and a byte-identical isolated transfer copy are staged outside the Git repository. No signing or execution occurred.

## Phase A — awaiting operator evidence

The operator selected the existing SSH path for the read-only preflight. Actual output has not yet been supplied. No device connection was attempted from this workspace.

Required next evidence: sanitized output from the exact P018 helper pinned at commit 8e951a005dbd684f1b3e09c35a27ed1d08001d14, plus the operator's report of physical access, stable power, and known manual Amethyst reactivation/recovery path. Phase A remains PENDING until the observed identity, rootful layout, tools, install state, service labels, and storage are reviewed.

## Subsequent checkpoints

Append an exact timestamp, source SHA/tree, evidence or approval reference, result, and blocker after each completed phase. Never promote pending work to PASS. Keep the P026 PR Draft while any mandatory physical acceptance case is unresolved. Record separate approvals before any device mutation. Do not merge P018 PR #22 automatically.

## Final gate snapshot

- Gate J: PASS.
- Gate F: PENDING.
- Release authorization: NOT GRANTED.
- Device changes: none.
- Production iOS branch changed: no.
- Public release created: no.
