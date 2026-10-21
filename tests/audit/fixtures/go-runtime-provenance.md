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

## P019 Go 1.27.2 source refresh

P019 pins the candidate to Go 1.27.2. The current patch fixture is
[`go1.27.2-procyield-arm64.s`](go1.27.2-procyield-arm64.s), taken from the
official [`go1.27.2/src/runtime/asm_arm64.s`](https://github.com/golang/go/blob/go1.27.2/src/runtime/asm_arm64.s#L698)
source (GitHub blob `d8b91b27a08248711961a2ca56a994443793961b`). The complete
executable function and next-function boundary retain the same normalized
source shape as the earlier 1.27.1 fixture. Fixture SHA-256:
`1fe45b2665b1544c7564d8f68719e302224328c1f9796abba4f809bce6afa666`.
P019's patch script accepts only Go 1.27.2; a version or source-shape change
fails closed. Native hosted CI must
still verify the actual copied toolchain source hash and assembled Agent/Hub
symbols before this refresh is qualified.
