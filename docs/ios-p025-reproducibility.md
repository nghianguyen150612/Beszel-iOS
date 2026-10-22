# P025 Reproducibility Assessment

**Status: ASSESSMENT TOOLING IMPLEMENTED — hosted results pending.** No P025 native build has run yet.

The comparison will use the same full source SHA, Go 1.27.3 toolchain, pinned runtime patch, frontend lockfile, build flags, target (`GOOS=ios`, `GOARCH=arm64`, `GOARM64=v8.0`, `CGO_ENABLED=1`), and minimum iOS 12.0. It will record two isolated build identities, binary and asset SHA-256 values, native metadata, VCS settings, manifests, and checksums.

The report will distinguish byte-identical outputs, source/configuration-equivalent builds, functional validation, and artifact provenance. Differing bytes will be investigated before any reproducibility claim. No provenance or security checks will be removed to force matching hashes.

The hosted workflow will compare Agent and Hub builds from the same commit with separate output directories and Go build caches on one macOS runner. Synthetic unit tests cover identical and differing outputs, source and target enforcement, hashes, and mismatch classification. These tests validate the assessment tooling; they are not native reproducibility results.

**Native results:** pending exact P025 hosted run.
