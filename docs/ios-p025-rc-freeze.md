# P025 Cloud Release Candidate Freeze

**Status: INITIAL CHECKPOINT — qualification not started.** This is a cloud-only milestone. It does not authorize signing, installation, device access, tagging, publishing, or production promotion.

## Verified baseline

| Identity | Value |
|---|---|
| Repository | `nghianguyen150612/Beszel-iOS` |
| Integration baseline | `07f6fd5f87c99348ec33549fe8dfed099464eb7f` |
| Integration tree | `e1cd43c81b131c05c4e4dfcde40d598dc962b330` |
| P024 tested source | `13c63387c588f65e288449ba611e5a83c6d6eef7` |
| Pinned upstream | `b97abf7662a7c7a2dd7542ea2062405abd22d4dc` |
| Production `iOS` | `0c3e4cd22ac24264834c401ad412e44aeda4e4f3` |
| Fork `main` | `e984ee12081f4e112601c3f718c9d81c112fe718` |
| Proposed version | `v0.21.0-ios.1` |
| Public iOS release | `v0.19.0-ios.1` |

GitHub reports PR #28 merged normally with merge commit `07f6fd5f87c99348ec33549fe8dfed099464eb7f`, parent commits `8e03afadb3a03480414fa47f87d40c13331e05b3` and `13c63387c588f65e288449ba611e5a83c6d6eef7`. The merge tree is `e1cd43c81b131c05c4e4dfcde40d598dc962b330`, identical to the tested P024 source tree. P024 ancestry checks retain 46 cumulative upstream commits, 178 frozen audit rows, 35 historical iOS commits, and 13 logical iOS patchsets.

## P024 workflow checkpoint

The following runs completed successfully on P024 feature head `13c63387c588f65e288449ba611e5a83c6d6eef7`:

- Native iOS build and Procursus checks: [39759219855](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39759219855)
- P017 regression/security: [39759219569](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39759219569)
- P013 migration fixture: [39759219686](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39759219686)
- Installer lifecycle: [39759219620](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39759219620)

P024 reports that external artifact ZIP downloads returned HTTP 403. GitHub-reported archive digests are recorded in the P024 PR description; they are not independent byte verification. P025 still needs an uploaded-artifact consumer job and exact downloaded-byte verification.

## P025 state at checkpoint

- Feature branch: `ios/p025-artifact-rc-freeze` from the exact integration baseline.
- Candidate source, build outputs, artifact IDs, and immutable manifest: pending.
- Gate J: **PENDING**.
- P018 Gate F: **PENDING**.
- Physical-device testing: not performed and out of scope for P025.
- Release authorization: not granted.
- No production branch, tag, or release has been changed or created.

See [validation](ios-p025-validation.json), [artifact manifest](ios-p025-artifact-manifest.json), [checkpoint log](ios-p025-checkpoints.md), and the remaining phase records for evolving evidence. Preliminary observations are not final qualification results.
