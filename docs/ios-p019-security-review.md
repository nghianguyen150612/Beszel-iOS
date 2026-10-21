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

## JavaScript build dependency

`GHSA-vfj7-8cjw-p6xm` remains **OPEN**. Prior P015 evidence classified the
affected `braces` dependency as build-only through Lingui and recorded that no
compatible fix was available. P019 will keep this advisory open unless a
current dependency audit demonstrates remediation. It is not treated as fixed
by the Go upgrade.

## Additional findings and boundaries

The current Go and Bun/npm scans have not completed. Additional findings and
their dispositions will be recorded after exact-head CI. Do not interpret
historical P015/P017 scans as P019 results. No production secrets or data are
included in the vulnerability evidence artifact. P018 physical-device Gate F
remains pending; no release, deployment, tag or device action is part of this
review.
