# P019 security review

Status: **IN PROGRESS**. This review is tied to the P019 candidate and is not
complete until hosted source scanning and native build provenance pass.

## Go standard library

The Go vulnerability database entry `GO-2026-6608` is reviewed at
[`golang/vulndb` commit `e0bad56276e98711a598adbab976de7cd914013a`](https://github.com/golang/vulndb/blob/e0bad56276e98711a598adbab976de7cd914013a/data/reports/GO-2026-6608.yaml).
It affects `net/textproto` and `mime/multipart`, with a multipart MIME header
memory limit bypass. The authoritative fixed version is Go 1.27.2; the candidate
`go.mod` pins Go 1.27.2. This is an expected fix, not yet a P019 PASS.

The P019 PR workflow runs pinned `govulncheck` v1.9.0 against the candidate with
the exact Go version from `go.mod`. Its machine summary requires a symbol-level
source scan, records all findings and database freshness, and fails if
`GO-2026-6608` remains a finding. Native iOS artifacts must independently report
Go 1.27.2 and the exact clean candidate source revision. Both pieces of evidence
are required to close the P019 disposition.

The exact-source P017 scan at `de40fbe8d3b5782d979bfbc2cbd76c5944b46764`
(run [39583459164](https://github.com/nghianguyen150612/Beszel-iOS/actions/runs/39583459164))
used Go 1.27.2 and govulncheck v1.9.0. It confirms `GO-2026-6608` is absent,
but reports three reachable `GO-2026-6809` traces, fixed in Go 1.27.3. The
trace reaches `crypto/x509.Certificate.Verify`, TLS chain handling, and
`internal/ghupdate.FetchLatestRelease`. This is an open P019 merge gate for the
requested Go 1.27.2 candidate. Do not merge until the owner chooses a fixed
Go 1.27.3 toolchain or explicitly accepts the 1.27.2 advisory disposition.

The same scan reports one module-level `GO-2026-5932` finding against
`golang.org/x/crypto` v0.58.0, without a symbol trace. That reviewed advisory
concerns OpenPGP; no OpenPGP imports were found in repository source. Keep it
recorded as a module-level finding without claiming remediation.

## JavaScript build dependency

`GHSA-vfj7-8cjw-p6xm` remains **OPEN**. Prior P015 evidence classified the
affected `braces` dependency as build-only through Lingui and recorded that no
compatible fix was available. P019 will keep this advisory open unless a
current dependency audit demonstrates remediation. It is not treated as fixed
by the Go upgrade.

## Additional findings and boundaries

The exact-source Bun and npm audits on `de40fbe8` report zero critical
findings and retain high `GHSA-vfj7-8cjw-p6xm` (`braces`) through the build-only
Lingui dependency chain. The advisory remains **OPEN** as required. Bun report
SHA-256 is `5a9adf8fbbd2a854066d8ebb81515723fc033927c4bcee0f7ed3ea13b300efba`;
npm report SHA-256 is
`1e69c0bd69e82835575ee872a1c48032a09b340d2bce52f46bbbd0db9b7e3d3b`.
The npm report records five affected dependency records for the same advisory.
No production secrets or data are included. P018 physical-device Gate F remains
pending; no release, deployment, tag, or device action is part of this review.
