# P023 Go 1.27.3 migration

**Reviewed candidate code commit:** `c9e68e1858bd71792ac79193257c95e406eb4c87`  
**Go release:** `go1.27.3`, released 2026-10-19  
**Starting integration:** `fd334076f53d8354a4924a4c734f29cf65a43879`

P023 moves the active Go toolchain contract from 1.27.2 to 1.27.3. Go's
official [release history](https://go.dev/doc/devel/release) records the
release and its crypto/x509 security changes.

## Toolchain contract

- `go.mod` declares `go 1.27.3`.
- Go Actions read the version from `go.mod`; native builds assert
  `go env GOVERSION == go1.27.3`.
- Six Docker Go builder stages use digest-pinned official Go 1.27.3 Alpine or
  Bookworm images, set `GOTOOLCHAIN=local`, and assert the exact version.
  The Procursus test image uses the pinned Bookworm digest and checks its Go
  version.
- The official Linux amd64 Go archive SHA-256 is
  `65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40`.
  Native macOS setup-go CI records and verifies its downloaded GOROOT source
  hashes before patching.
- P019/P022 Go 1.27.2 runtime fixtures and the x509 patch utility remain
  historical evidence. The active A7 patcher accepts only the reviewed Go
  1.27.3 runtime source hash and function shape. The x509 backport is no longer
  applied.

## x509 backport reconciliation

The P021/P022 backport implemented Go CL
[851047](https://go-review.googlesource.com/c/go/+/851047), commit
`7f86f957633ee31a1c17a1eb8452ca7a3d9bc141`. The official Go 1.27.3 release
contains that change natively. Its source hashes match the reviewed backport
output:

| Source file | SHA-256 |
|---|---|
| `src/crypto/x509/root_darwin.go` | `69b350fe1afa679935ca831c3beea9833d59d8cd62c6f0b4d02f14526dd46ac1` |
| `src/crypto/x509/root_windows.go` | `ba910c270b9d096f50ccbf9426c03410ad16bc46a86e0ba4237d27c3381d28d0` |
| `src/crypto/x509/verify.go` | `a239364c8fd240db693be8ec4f82e84958d9b4e4f265ff3923499e2161a34d86` |

`scripts/audit/validate-go-x509-native.py` fails closed on an unknown Go
version or source file and runs `go test crypto/x509 -count=1`. The local
Go 1.27.3 run passed. No custom x509 change is applied twice. Historical P021
and P022 backport evidence remains unchanged.

## Vulnerability scan results

`govulncheck v1.9.0` ran in symbol-level source mode for Agent and Hub using
Go 1.27.3 and the vulnerability database last updated
2026-10-21 00:27:29 UTC. P023 retains raw output and requires exactly one
reviewed `GO-2026-5932` module-only finding; it fails if the finding
disappears, changes shape/version, or another advisory appears.

- **GO-2026-6608:** Fixed in Go 1.27.2. The official Go 1.27.3 source scan
  reports no finding. P023 uses the upstream fixed toolchain.
- **GO-2026-6809:** Affects Go 1.27 before 1.27.3. P023 verifies the native
  CL 851047 source and passing x509 tests; the exact Go 1.27.3 raw scan reports
  no finding. Status: `REMEDIATED_AND_VERIFIED`.
- **GO-2026-5932:** The module-level warning remains for unmaintained
  `golang.org/x/crypto/openpgp` packages, with no fixed release listed.
  P023's AST scan covered 377 repository Go files and found no OpenPGP import.
  Six host/iOS build-tag and test dependency profiles found no OpenPGP
  package. `go mod why -m golang.org/x/crypto` identifies the Agent's SSH
  dependency. The module remains for SSH. Status:
  `NOT_REACHABLE_WITH_EVIDENCE`, not fixed.

The source-level scan does not infer exploitability from the mere presence of
`golang.org/x/crypto`. The residual status and exact scanner record are in
[the advisory matrix](ios-p023-advisory-matrix.json).

## Qualification scope

Local Go 1.27.3 qualification passed `go test crypto/x509 -count=1`, all
117 audit tests, the six-profile OpenPGP guard, and the serial full Go suite.
Hosted P017 CI runs the same-version symbol scan, full Go tests, and portable
Agent/Hub builds. Hosted macOS CI patches an isolated Go 1.27.3 GOROOT for A7
compatibility and independently builds native iOS Agent and Hub. It checks
source and output provenance, Go build metadata, Mach-O device metadata,
iOS 12.0 minimum, and binary checksums.

Go 1.27.3 compatibility does not establish physical Apple A7 launchability.
P018 Gate F remains pending, and this migration is not public release
approval.
