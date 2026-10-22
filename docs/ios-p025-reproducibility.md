# P025 Reproducibility Assessment

**Status: NOT ASSESSED — initial checkpoint.** No P025 native build has run.

The comparison will use the same full source SHA, Go 1.27.3 toolchain, pinned runtime patch, frontend lockfile, build flags, target (`GOOS=ios`, `GOARCH=arm64`, `GOARM64=v8.0`, `CGO_ENABLED=1`), and minimum iOS 12.0. It will record two isolated build identities, binary and asset SHA-256 values, native metadata, VCS settings, manifests, and checksums.

The report will distinguish byte-identical outputs, source/configuration-equivalent builds, functional validation, and artifact provenance. Differing bytes will be investigated before any reproducibility claim. No provenance or security checks will be removed to force matching hashes.

**Results:** pending.
