# P026 artifact verification and transfer

**Cloud artifact verification: PASS. Physical transfer: PENDING.** The verified bytes have not been transferred to or signed on the iPad.

## Frozen candidate

- Repository: nghianguyen150612/Beszel-iOS
- P025 PR #29: merged
- Candidate source and current integration HEAD: bb945dc603ba2ae1673bb22866cc929d037fb9dd
- Tree: 8806c06d89881c9b41d5839babde84ca850dbd82
- P025 native workflow: run 39782135426, completed SUCCESS on the merge SHA
- Payload artifact: 12428152331, beszel-ios-arm64
- Native validation artifact: 12428676214, ios-native-validation
- Artifact-consumer manifest: 12428267122, ios-artifact-consumer-verification
- P025 Gate J: PASS for this exact merged source and same-run artifact verification
- P018 PR #22 remains Draft; it is not merged into the newer P025 baseline.

The current integration ref is still exactly the frozen source SHA. No later integration source update has superseded this candidate.

The inherited P025 preliminary validation and artifact-manifest files still say gate_j is pending because they were written before the post-merge run. The exact-merge run and its passing artifact consumer below provide the later evidence; P026 records that completed checkpoint without rewriting the historical preliminary P025 snapshot.

## Independently checked downloaded bytes

The GitHub artifact API associated each artifact with workflow 39782135426 and source SHA bb945dc603ba2ae1673bb22866cc929d037fb9dd. Downloaded archive hashes were calculated from the bytes obtained in this session and match GitHub's artifact digest metadata:

| Artifact | Size | Downloaded archive SHA-256 | Result |
| --- | ---: | --- | --- |
| beszel-ios-arm64, ID 12428152331 | 16,366,579 bytes | b87b8545f1104c0915e39634275d91467f88e993a9c14def2bbc24eaf71d0c99 | PASS |
| ios-native-validation, ID 12428676214 | 15,810 bytes | fafe67f1c2a8634d689f98e8dee1f1c9da070244136d2a35f1d5d48b8357bcc5 | PASS |
| ios-artifact-consumer-verification, ID 12428267122 | 3,481 bytes | e5189ca9667a4e868d87081abb536af56bd810662ea6f331515397bc05ee7640 | PASS |

The payload archive contains only the Agent, Hub, and SHA256SUMS. Its embedded checksum file passes against both binaries:

| Binary | Bytes | SHA-256 |
| --- | ---: | --- |
| beszel-agent-ios-arm64 | 9,576,424 | 3db9aa10205add17318e832ad561604bb8d0e23466d90f5da9a8ffecbf32ec42 |
| beszel-hub-ios-arm64 | 32,174,816 | e746ea219c5e30ec2e749bf1d1c247f473d00ae2b49c31d0575ba132d0c6f202 |

Local file inspection identifies both as Mach-O 64-bit arm64 executables. The exact-run consumer and native-validation manifests report MH_EXECUTE, iOS minimum 12.0, Go 1.27.3, GOOS=ios, GOARCH=arm64, GOARM64=v8.0, CGO_ENABLED=1, and vcs.revision equal to the frozen source SHA. The Hub embeds 98 frontend assets.

The consumer reports PASS for independent same-run payload download, SHA256SUMS, exact source provenance, native metadata, A7 runtime instruction inspection, frontend embedding, x509 evidence, and absence of private MobileWiFi linkage. Its A7 check covers the reported runtime.procyieldAsm.abi0 sequence and reports no counter access; this is static evidence, not a physical launch result.

The downloaded unsigned originals and a separate byte-identical transfer copy are staged outside the repository. No signing has occurred. Recheck hashes after any transfer and record unsigned and post-signing hashes separately; signing may change bytes.

## Safe acquisition and transfer

For another workstation, acquire by exact artifact ID from workflow 39782135426 and verify the source SHA and artifact identity against this record before extraction. Do not use mutable Latest assets, a public mirror, a release installer, or an unverified rebuild. Keep the original unsigned archive and extracted binaries.

Physical transfer is blocked until Phase A passes and separate operator approval is given before transfer, signing, or execution. Do not place these files in /usr/local/bin yet. Do not publish the candidate as a release.
