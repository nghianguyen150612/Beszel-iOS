# P023 Apple A7 runtime review

**Qualified implementation source:** fd6b1595ff2e8b4d7037229ce6d71af87f907b4e
**Toolchain:** Go 1.27.3
**Target:** iOS device ARM64, deployment minimum 12.0

## Exact-source review

Go 1.27.3 src/runtime/asm_arm64.s has SHA-256 47f9313d4f956b4159dcf8b292bf340a12adb6b58ee9a94360fe948045e2c5db. P023 compared it with P019's Go 1.27.2 source and confirmed runtime·procyieldAsm is unchanged. The fixture and attribution are recorded in tests/audit/fixtures/go-runtime-provenance.md.

The patcher accepts only Go 1.27.3, the whole-file source hash, and the exact executable function shape. It refuses unknown layouts, extra instructions, repeat application, symlinks, missing files, or mismatched toolchains. It validates before writing and atomically replaces only the guarded function. The patched source hash is c861b98d76ac487269c7f98c987b94ea9978c5f01740a4596eecbfdb9c8bb91f.

The replacement returns on a zero delay count, then executes a bounded YIELD/decrement loop. It contains no CNTFRQ_EL0 or CNTVCT_EL0 reads. The original loop measured nanoseconds; YIELD iterations do not. P023 does not claim equivalent timing or physical-device performance.

## Binary and target checks

The unstripped Agent and Hub validation builds are checked at runtime.procyieldAsm by .github/scripts/verify-ios-runtime-binary.py. It compares every instruction word and branch, rejects unknown or extra instructions and counter reads, and requires the zero guard. Production Agent and Hub builds retain existing -s -w flags.

Hosted macOS CI used:

- GOOS=ios
- GOARCH=arm64
- GOARM64=v8.0
- CGO_ENABLED=1
- Go 1.27.3
- Minimum iOS 12.0

It used the device SDK clang wrapper, patched a separate runner-local GOROOT, built Agent and Hub twice with isolated caches, compared outputs, inspected Mach-O metadata, verified embedded frontend assets and checksums, and rejected private MobileWiFi linkage.

P017 native run 39718772276 passed. Its review payload artifact 12400817906 has GitHub SHA-256 73b8080826c337b70588c023cea93ccc6b1ce222a0a375807095d560eb963d30. The independent validator confirmed the payload's Agent and Hub Go build metadata reports source fd6b1595ff2e8b4d7037229ce6d71af87f907b4e with vcs.modified=false, both are thin ARM64 iOS MH_EXECUTE binaries with minimum iOS 12.0, and Hub embeds 97 assets. The payload checksums match SHA256SUMS.

Agent binary SHA-256: 5676c4b49a0312339c0697cdf490cbda16177bebda7d311cc47a99a8fdd8719f.
Hub binary SHA-256: 2d63fa87486f0c455122886f7c9b262a0c465586d27940a8d2f3025e5592b850.

The separate disabled Wi-Fi diagnostic artifact 12400947269 reports activation disabled, no private calls executed, and not included in Agent/Hub. No physical device was accessed. Hosted cross-build evidence does not prove an A7 device can launch the binaries. P018 Gate F remains pending.
