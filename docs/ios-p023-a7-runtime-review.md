# P023 Apple A7 runtime review

**Reviewed candidate code commit:** `c9e68e1858bd71792ac79193257c95e406eb4c87`  
**Toolchain:** Go 1.27.3  
**Target:** iOS device ARM64, deployment minimum 12.0

## Exact-source review

The Go 1.27.3 upstream runtime source
[`src/runtime/asm_arm64.s`](https://github.com/golang/go/blob/go1.27.3/src/runtime/asm_arm64.s)
has SHA-256
`47f9313d4f956b4159dcf8b292bf340a12adb6b58ee9a94360fe948045e2c5db`.
P023 compared it with P019's Go 1.27.2 source and confirmed that
`runtime·procyieldAsm` is unchanged. The complete fixture and attribution
are recorded in
[`go-runtime-provenance.md`](../tests/audit/fixtures/go-runtime-provenance.md).

The patcher accepts only Go 1.27.3, the whole-file source hash, and the exact
executable function shape. It refuses unknown layouts, extra instructions,
repeat application, symlinks, missing files, or mismatched toolchains. It
validates before writing and atomically replaces only the guarded function.
The patched source hash recorded by the qualified Linux toolchain is
`c861b98d76ac487269c7f98c987b94ea9978c5f01740a4596eecbfdb9c8bb91f`.

The replacement returns on a zero delay count, then executes a bounded
`YIELD` / decrement loop. It contains no `CNTFRQ_EL0` or `CNTVCT_EL0`
reads. The original loop measured nanoseconds; YIELD iterations do not. P023
does not claim equivalent timing or physical-device performance.

## Binary and target checks

The unstripped Agent and Hub validation builds are checked at
`runtime.procyieldAsm` by `.github/scripts/verify-ios-runtime-binary.py`.
It compares every instruction word and branch, rejects unknown or extra
instructions and counter reads, and requires the zero guard. Production
Agent and Hub builds retain the existing `-s -w` flags.

Hosted macOS CI sets:

```text
GOOS=ios
GOARCH=arm64
GOARM64=v8.0
CGO_ENABLED=1
Go=1.27.3
Minimum iOS=12.0
```

It uses the device SDK clang wrapper, patches a separate runner-local GOROOT,
builds Agent and Hub twice with independent caches, compares outputs, inspects
Mach-O metadata, verifies embedded frontend assets and checksums, and rejects
private MobileWiFi linkage. Hosted CI is authoritative for native link and
artifact results; a cross-build does not prove an A7 device can launch.

Exact-head native workflow and artifact state are tracked in
[`ios-p023-validation.json`](ios-p023-validation.json). No physical device
was accessed. P018 Gate F remains pending.
