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

The exact-head scan at P019 source `070c4119fa5ef13d90ea26e3d7b6a9ed84613436`
used govulncheck v1.9.0 and a vulnerability database last modified
`2026-10-21T00:27:29Z`. It confirmed `GO-2026-6608` is absent, but surfaced the
newly published reviewed advisory `GO-2026-6809` three times on the reachable
path `crypto/x509.Certificate.Verify` → TLS session handling →
`internal/ghupdate.FetchLatestRelease`. The authoritative Go record says the
fix is Go 1.27.3, released 2026-10-19. This is an open P019 security gate for
the requested Go 1.27.2 candidate; it must be fixed or explicitly accepted
before merge. The scan also reports `GO-2026-5932` against `golang.org/x/crypto`
without a symbol trace. That reviewed advisory concerns `openpgp`; no OpenPGP
imports were found in the repository source. Preserve this as a module-level
finding pending final disposition.

## JavaScript build dependency

`GHSA-vfj7-8cjw-p6xm` remains **OPEN**. Prior P015 evidence classified the
affected `braces` dependency as build-only through Lingui and recorded that no
compatible fix was available. P019 will keep this advisory open unless a
current dependency audit demonstrates remediation. It is not treated as fixed
by the Go upgrade.

## Additional findings and boundaries

The exact-head Bun and npm audits report zero critical findings and retain the
known high `GHSA-vfj7-8cjw-p6xm` (`braces`) advisory through the build-only
Lingui dependency chain. This advisory remains **OPEN** as required. The npm
report has five affected dependency records for that same advisory. Do not
interpret historical P015/P017 scans as P019 results. The Go and JavaScript
reports are tied to the exact source SHA above; no production secrets or data
are included. P018 physical-device Gate F remains pending; no release,
deployment, tag or device action is part of this review.
