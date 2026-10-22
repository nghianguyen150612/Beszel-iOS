# P023 Go 1.27.3 migration

**Qualified implementation source:** fd6b1595ff2e8b4d7037229ce6d71af87f907b4e
**Go release:** go1.27.3, released 2026-10-19
**Starting integration:** fd334076f53d8354a4924a4c734f29cf65a43879

Go's official [release history](https://go.dev/doc/devel/release) records Go 1.27.3 and its crypto/x509 security changes.

## Toolchain contract

- go.mod declares Go 1.27.3. GitHub Actions read the version from go.mod.
- Native builds assert go env GOVERSION is go1.27.3.
- Six Docker Go builder stages use digest-pinned official Go 1.27.3 Alpine or Bookworm images, set GOTOOLCHAIN=local, and assert the exact version. The Procursus test image uses a pinned Bookworm digest and checks its Go version.
- The official Linux amd64 Go archive SHA-256 is 65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40.
- P019/P022 Go 1.27.2 runtime fixtures and x509 patch evidence remain unchanged as historical records. The active A7 patcher accepts only the reviewed Go 1.27.3 runtime source hash and exact function shape. The custom x509 backport is no longer applied.

## x509 backport reconciliation

The P021/P022 backport implemented Go CL [851047](https://go-review.googlesource.com/c/go/+/851047), commit 7f86f957633ee31a1c17a1eb8452ca7a3d9bc141. Go 1.27.3 contains the fix natively. Its source hashes match the reviewed backport output:

| Source file | SHA-256 |
|---|---|
| src/crypto/x509/root_darwin.go | 69b350fe1afa679935ca831c3beea9833d59d8cd62c6f0b4d02f14526dd46ac1 |
| src/crypto/x509/root_windows.go | ba910c270b9d096f50ccbf9426c03410ad16bc46a86e0ba4237d27c3381d28d0 |
| src/crypto/x509/verify.go | a239364c8fd240db693be8ec4f82e84958d9b4e4f265ff3923499e2161a34d86 |

scripts/audit/validate-go-x509-native.py fails closed on an unknown Go version or source file and runs go test crypto/x509 -count=1. Local and hosted native Go 1.27.3 checks passed. No custom x509 change is applied twice. Historical P021/P022 backport evidence remains unchanged.

## Vulnerability scan results

P017 ran govulncheck v1.9.0 in symbol-level source mode for Agent and Hub using Go 1.27.3 and a Go vulnerability database last updated 2026-10-21 00:27:29 UTC. The raw source-bound report contains exactly one finding: GO-2026-5932 with a module-only trace to golang.org/x/crypto v0.58.0. The policy fails if this finding disappears, changes shape/version, or another advisory appears.

- **GO-2026-6608:** Fixed in Go 1.27.2. The official Go 1.27.3 source scan reports no finding. P023 uses the fixed upstream toolchain.
- **GO-2026-6809:** Affects Go 1.27 before 1.27.3. The exact Go 1.27.3 source scan reports no finding. P023 verifies native CL 851047 source hashes, passes crypto/x509 tests, and does not apply the custom backport. Status: REMEDIATED_AND_VERIFIED.
- **GO-2026-5932:** The module-level warning remains for the unmaintained golang.org/x/crypto/openpgp packages, with no fixed release listed. The AST scan covered 377 repository Go files and found no OpenPGP imports. Six host/iOS build-tag and test dependency profiles found no OpenPGP package. The module remains for Agent SSH. Status: NOT_REACHABLE_WITH_EVIDENCE, not fixed.

The official Go vulnerability records do not publish severity scores for these findings. The P021 historical security review classified GO-2026-6608 as HIGH; the other two remain unscored in the reviewed Go records.

## Qualification

Local Go 1.27.3 qualification passed go test crypto/x509 -count=1, all 123 audit tests, the six-profile OpenPGP guard, and the serial full Go suite. Hosted P017 passed the same-version symbol scan, full Go tests, and portable Agent/Hub builds.

Hosted macOS CI patched an isolated Go 1.27.3 GOROOT for A7 compatibility and independently built native iOS Agent and Hub. The native check validated Go build metadata, Mach-O device metadata, iOS 12.0 minimum, runtime instructions, source provenance, embedded frontend assets, and checksums.

Go 1.27.3 compatibility does not prove physical Apple A7 launchability. P018 Gate F remains pending, and this migration is not public release approval.
