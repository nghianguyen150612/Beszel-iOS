# Go ARM64 runtime fixture provenance

`go1.27.1-procyield-arm64.s` is the existing P006 fragment of
[`go1.27.1/src/runtime/asm_arm64.s`](https://github.com/golang/go/blob/go1.27.1/src/runtime/asm_arm64.s#L698).
The function and its next-function comment boundary were compared byte for byte
with the actual installed Go 1.27.1 source during P007. They match. The fixture's
surrounding attribution/sentinel comments are local; its license is
`LICENSE.go-runtime`. Fixture SHA-256:
`6a778321962ff3d351a2167246eabd79643b8a00e5fd14274f965993dfeb702d`.

The full unmodified Linux toolchain `src/runtime/asm_arm64.s` SHA-256 is
`47f9313d4f956b4159dcf8b292bf340a12adb6b58ee9a94360fe948045e2c5db`.
CI records the actual macOS source and replacement hashes separately.

The binary validator's six instruction words were obtained with Go 1.27.1
`GOARCH=arm64 go tool asm` followed by `go tool objdump` on the replacement
function. Hosted CI independently inspects the linked Agent and Hub validation
binaries; fixture assembly alone is not native-link or physical-device evidence.
